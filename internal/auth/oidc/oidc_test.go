package oidc_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/MWest2020/wanderer/internal/auth/oidc"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

const testClientID = "wanderer"

func TestConfigValidate(t *testing.T) {
	base := oidc.Config{
		ProviderURL:  "https://cloud.example.nl",
		ClientID:     "wanderer",
		ClientSecret: "shh",
		RedirectURL:  "https://wanderer.example.nl/ui/oauth/callback",
	}
	if err := base.Validate(); err != nil {
		t.Fatalf("complete config should validate: %v", err)
	}
	for name, mutate := range map[string]func(c *oidc.Config){
		"missing provider": func(c *oidc.Config) { c.ProviderURL = "" },
		"missing client":   func(c *oidc.Config) { c.ClientID = "" },
		"missing secret":   func(c *oidc.Config) { c.ClientSecret = "" },
		"missing redirect": func(c *oidc.Config) { c.RedirectURL = "" },
	} {
		c := base
		mutate(&c)
		if err := c.Validate(); err == nil {
			t.Errorf("%s: expected validation error", name)
		}
	}
}

// New must not contact the network — discovery is deferred.
func TestNewDefersDiscovery(t *testing.T) {
	_, err := oidc.New(oidc.Config{
		ProviderURL:  "https://offline.invalid",
		ClientID:     "wanderer",
		ClientSecret: "shh",
		RedirectURL:  "https://wanderer.example.nl/ui/oauth/callback",
	})
	if err != nil {
		t.Fatalf("New should not contact the provider: %v", err)
	}
}

func TestExchangeAndRevalidate(t *testing.T) {
	p := newMockProvider(t)
	defer p.Close()

	auth, err := oidc.New(oidc.Config{
		ProviderURL:  p.issuer,
		ClientID:     testClientID,
		ClientSecret: "shh",
		RedirectURL:  "https://wanderer.example.nl/ui/oauth/callback",
	})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	ctx := context.Background()

	// AuthCodeURL triggers discovery and embeds state + nonce.
	authURL, err := auth.AuthCodeURL(ctx, "state-123", "nonce-abc")
	if err != nil {
		t.Fatalf("authcodeurl: %v", err)
	}
	u, _ := url.Parse(authURL)
	if u.Query().Get("state") != "state-123" || u.Query().Get("nonce") != "nonce-abc" {
		t.Fatalf("authorize URL missing state/nonce: %s", authURL)
	}

	// Happy-path exchange: the mock mints an ID token with the
	// matching nonce.
	p.nonce = "nonce-abc"
	claims, tokens, err := auth.Exchange(ctx, "code-1", "nonce-abc")
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	if claims.Subject != "alice" || claims.Email != "alice@example.nl" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
	if tokens.AccessToken == "" {
		t.Fatal("expected an access token")
	}

	// Nonce mismatch must be rejected.
	if _, _, err := auth.Exchange(ctx, "code-1", "different-nonce"); err == nil {
		t.Fatal("expected nonce mismatch error")
	}

	// Revalidate against a live userinfo endpoint succeeds...
	if _, err := auth.Revalidate(ctx, *tokens); err != nil {
		t.Fatalf("revalidate (enabled user): %v", err)
	}
	// ...and fails once the provider starts rejecting the token
	// (the Nextcloud-side disable case).
	p.userinfoUnauthorized = true
	if _, err := auth.Revalidate(ctx, *tokens); err == nil {
		t.Fatal("expected revalidation failure for a disabled user")
	}
}

// mockProvider is a minimal OIDC provider: discovery, JWKS, a token
// endpoint that mints a signed ID token, and a userinfo endpoint
// that can be flipped to 401 to model a disabled account.
type mockProvider struct {
	*httptest.Server
	issuer               string
	key                  *rsa.PrivateKey
	nonce                string
	userinfoUnauthorized bool
}

func newMockProvider(t *testing.T) *mockProvider {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("genkey: %v", err)
	}
	p := &mockProvider{key: key}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]any{
			"issuer":                                p.issuer,
			"authorization_endpoint":                p.issuer + "/authorize",
			"token_endpoint":                        p.issuer + "/token",
			"userinfo_endpoint":                     p.issuer + "/userinfo",
			"jwks_uri":                              p.issuer + "/jwks",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{
			Key:       &key.PublicKey,
			KeyID:     "test",
			Algorithm: "RS256",
			Use:       "sig",
		}}})
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]any{
			"access_token":  "access-token",
			"refresh_token": "refresh-token",
			"token_type":    "Bearer",
			"expires_in":    3600,
			"id_token":      p.mintIDToken(t),
		})
	})
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, _ *http.Request) {
		if p.userinfoUnauthorized {
			http.Error(w, "user disabled", http.StatusUnauthorized)
			return
		}
		writeJSON(w, map[string]any{"sub": "alice", "email": "alice@example.nl"})
	})
	p.Server = httptest.NewServer(mux)
	p.issuer = p.Server.URL
	return p
}

func (p *mockProvider) mintIDToken(t *testing.T) string {
	t.Helper()
	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: p.key},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", "test"),
	)
	if err != nil {
		t.Fatalf("signer: %v", err)
	}
	claims := map[string]any{
		"iss":   p.issuer,
		"sub":   "alice",
		"aud":   testClientID,
		"exp":   time.Now().Add(time.Hour).Unix(),
		"iat":   time.Now().Unix(),
		"nonce": p.nonce,
		"email": "alice@example.nl",
		"name":  "Alice Operator",
	}
	raw, err := jwt.Signed(signer).Claims(claims).Serialize()
	if err != nil {
		t.Fatalf("sign id_token: %v", err)
	}
	return raw
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
