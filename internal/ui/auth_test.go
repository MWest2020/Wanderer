package ui_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MWest2020/wanderer/internal/auth/oidc"
	"github.com/MWest2020/wanderer/internal/store"
	"github.com/MWest2020/wanderer/internal/ui"

	"golang.org/x/crypto/bcrypt"
)

// writeHtpasswd creates a one-line bcrypt htpasswd file and returns
// its path.
func writeHtpasswd(t *testing.T, user, pass string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	path := filepath.Join(t.TempDir(), "htpasswd")
	if err := os.WriteFile(path, []byte(user+":"+string(hash)+"\n"), 0o600); err != nil {
		t.Fatalf("write htpasswd: %v", err)
	}
	return path
}

// newOIDCServer mounts the UI with a (never-discovered) OIDC
// authenticator. The provider URL is unreachable on purpose: the
// gate's redirect path must not contact it, so the test stays
// hermetic. revalidate is large so an established session is
// honoured without a userinfo round-trip.
func newOIDCServer(t *testing.T, htpasswd string, revalidate time.Duration) (*httptest.Server, *store.Store) {
	t.Helper()
	st := newTestStore(t)
	auth, err := oidc.New(oidc.Config{
		ProviderURL:  "https://provider.invalid",
		ClientID:     "wanderer",
		ClientSecret: "shh",
		RedirectURL:  "https://wanderer.example.nl/ui/oauth/callback",
	})
	if err != nil {
		t.Fatalf("oidc.New: %v", err)
	}
	h, err := ui.Handler(st, ui.Options{
		HtpasswdPath:       htpasswd,
		Auth:               auth,
		MountPrefix:        "/ui",
		RevalidateInterval: revalidate,
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	mux := http.NewServeMux()
	mux.Handle("/ui/", http.StripPrefix("/ui", h))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, st
}

func noRedirectClient() *http.Client {
	return &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
}

func TestOIDC_UnauthenticatedRedirectsToLogin(t *testing.T) {
	srv, _ := newOIDCServer(t, "", time.Hour)
	resp, err := noRedirectClient().Get(srv.URL + "/ui/")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("status = %d, want 302", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc != "/ui/login" {
		t.Fatalf("Location = %q, want /ui/login", loc)
	}
}

func TestOIDC_LoginRouteIsPublic(t *testing.T) {
	// /ui/login must bypass the gate (otherwise: redirect loop).
	// With an unreachable provider, discovery fails and the handler
	// returns 503 — which still proves the route is reachable
	// without a session.
	srv, _ := newOIDCServer(t, "", time.Hour)
	resp, err := noRedirectClient().Get(srv.URL + "/ui/login")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusFound {
		t.Fatalf("/ui/login should not redirect back to itself")
	}
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503 (provider unreachable)", resp.StatusCode)
	}
}

func TestOIDC_ValidSessionIsHonoured(t *testing.T) {
	srv, st := newOIDCServer(t, "", time.Hour) // large revalidate window
	// Seed a live session directly, bypassing the OIDC dance.
	sess := &store.Session{
		ID:        "sess-1",
		Subject:   "alice",
		ExpiresAt: time.Now().UTC().Add(time.Hour),
	}
	if err := st.CreateSession(context.Background(), sess); err != nil {
		t.Fatalf("seed session: %v", err)
	}
	req, _ := http.NewRequest("GET", srv.URL+"/ui/", nil)
	req.AddCookie(&http.Cookie{Name: "wanderer_session", Value: "sess-1"})
	resp, err := noRedirectClient().Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("authenticated browse status = %d, want 200", resp.StatusCode)
	}
}

func TestOIDC_HtpasswdBreakGlass(t *testing.T) {
	// With OIDC configured AND htpasswd present, valid Basic
	// credentials must pass even though the provider is down — the
	// break-glass path from the spec's OIDC-outage scenario.
	htpasswd := writeHtpasswd(t, "ops", "s3cret")
	srv, _ := newOIDCServer(t, htpasswd, time.Hour)

	req, _ := http.NewRequest("GET", srv.URL+"/ui/", nil)
	req.SetBasicAuth("ops", "s3cret")
	resp, err := noRedirectClient().Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("break-glass status = %d, want 200", resp.StatusCode)
	}

	// Wrong Basic credentials still fall through to the OIDC
	// redirect rather than rendering the page.
	req2, _ := http.NewRequest("GET", srv.URL+"/ui/", nil)
	req2.SetBasicAuth("ops", "wrong")
	resp2, err := noRedirectClient().Do(req2)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusFound || !strings.HasSuffix(resp2.Header.Get("Location"), "/ui/login") {
		t.Fatalf("bad creds: status=%d loc=%q, want 302 -> /ui/login", resp2.StatusCode, resp2.Header.Get("Location"))
	}
}
