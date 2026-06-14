// Package oidc is the OIDC client mechanics for the optional
// Nextcloud login on `wanderer serve --ui`. It is deliberately
// thin: discovery, the authorization-code exchange, ID-token
// verification, and a userinfo revalidation call. The HTTP wiring
// (routes, session cookies, the htpasswd fallback policy) lives in
// internal/ui — this package knows nothing about chi or sessions.
//
// The provider is discovered lazily on first use, not at New(),
// so `wanderer serve` starts even when Nextcloud is unreachable:
// the configured htpasswd file stays available as break-glass
// access, and OIDC begins working as soon as the provider answers
// discovery.
//
// Boring on purpose: it composes github.com/coreos/go-oidc with
// golang.org/x/oauth2 — the de-facto standard, audited libraries —
// rather than hand-rolling the protocol.
package oidc

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"time"

	coreoidc "github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// Config is the resolved oidc: block from serve.yaml. ClientSecret
// is the secret itself (the serve layer reads it from
// client_secret_file before constructing this).
type Config struct {
	ProviderURL  string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

// Validate enforces that every field needed to start an
// authorization-code flow is present. It performs no network I/O —
// an unreachable provider is a runtime concern, not a config error.
func (c Config) Validate() error {
	if c.ProviderURL == "" {
		return errors.New("oidc: provider_url is required")
	}
	if _, err := url.Parse(c.ProviderURL); err != nil {
		return fmt.Errorf("oidc: provider_url: %w", err)
	}
	if c.ClientID == "" {
		return errors.New("oidc: client_id is required")
	}
	if c.ClientSecret == "" {
		return errors.New("oidc: client_secret is required (set client_secret_file)")
	}
	if c.RedirectURL == "" {
		return errors.New("oidc: redirect_url is required")
	}
	return nil
}

// Claims is the operator identity extracted from a verified ID
// token. Subject is the stable OIDC `sub`; Email/Name are
// best-effort (present only when the profile/email scopes are
// granted).
type Claims struct {
	Subject string
	Email   string
	Name    string
}

// Tokens carries the OAuth2 token set the session layer persists so
// it can later revalidate and refresh.
type Tokens struct {
	AccessToken  string
	RefreshToken string
	Expiry       time.Time
}

// Authenticator holds the resolved config and the lazily-discovered
// provider. It is safe for concurrent use.
type Authenticator struct {
	cfg Config

	mu       sync.Mutex
	provider *coreoidc.Provider
	verifier *coreoidc.IDTokenVerifier
	oauth    *oauth2.Config
}

// New validates the config and returns an Authenticator. It does
// NOT contact the provider — discovery is deferred to the first
// AuthCodeURL/Exchange/Revalidate call.
func New(cfg Config) (*Authenticator, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if len(cfg.Scopes) == 0 {
		cfg.Scopes = []string{coreoidc.ScopeOpenID, "profile", "email"}
	}
	return &Authenticator{cfg: cfg}, nil
}

// ensureReady performs OIDC discovery on first use and caches the
// provider, verifier, and oauth2 config. A prior failure is not
// cached: the next call retries discovery, so a provider that was
// down at first login recovers without a restart.
func (a *Authenticator) ensureReady(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.provider != nil {
		return nil
	}
	provider, err := coreoidc.NewProvider(ctx, a.cfg.ProviderURL)
	if err != nil {
		return fmt.Errorf("oidc: discovery against %s: %w", a.cfg.ProviderURL, err)
	}
	a.provider = provider
	a.verifier = provider.Verifier(&coreoidc.Config{ClientID: a.cfg.ClientID})
	a.oauth = &oauth2.Config{
		ClientID:     a.cfg.ClientID,
		ClientSecret: a.cfg.ClientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  a.cfg.RedirectURL,
		Scopes:       a.cfg.Scopes,
	}
	return nil
}

// AuthCodeURL returns the provider authorize URL for a login. state
// is the CSRF token (validated on callback); nonce binds the ID
// token to this login attempt and is checked in Exchange.
func (a *Authenticator) AuthCodeURL(ctx context.Context, state, nonce string) (string, error) {
	if err := a.ensureReady(ctx); err != nil {
		return "", err
	}
	return a.oauth.AuthCodeURL(state, coreoidc.Nonce(nonce)), nil
}

// Exchange swaps the authorization code for tokens, verifies the
// ID token's signature/issuer/audience, checks the nonce against
// expectedNonce, and returns the operator claims plus the token set.
func (a *Authenticator) Exchange(ctx context.Context, code, expectedNonce string) (*Claims, *Tokens, error) {
	if err := a.ensureReady(ctx); err != nil {
		return nil, nil, err
	}
	oauthToken, err := a.oauth.Exchange(ctx, code)
	if err != nil {
		return nil, nil, fmt.Errorf("oidc: code exchange: %w", err)
	}
	rawID, ok := oauthToken.Extra("id_token").(string)
	if !ok || rawID == "" {
		return nil, nil, errors.New("oidc: token response missing id_token")
	}
	idToken, err := a.verifier.Verify(ctx, rawID)
	if err != nil {
		return nil, nil, fmt.Errorf("oidc: verify id_token: %w", err)
	}
	if idToken.Nonce != expectedNonce {
		return nil, nil, errors.New("oidc: id_token nonce mismatch")
	}
	var raw struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := idToken.Claims(&raw); err != nil {
		return nil, nil, fmt.Errorf("oidc: decode claims: %w", err)
	}
	claims := &Claims{Subject: idToken.Subject, Email: raw.Email, Name: raw.Name}
	tokens := &Tokens{
		AccessToken:  oauthToken.AccessToken,
		RefreshToken: oauthToken.RefreshToken,
		Expiry:       oauthToken.Expiry,
	}
	return claims, tokens, nil
}

// Revalidate confirms the session's tokens are still honoured by
// the provider. It refreshes an expired access token (refresh
// failure ⇒ the account was disabled or consent revoked) and then
// calls the userinfo endpoint. On success it returns the possibly
// refreshed token set; on any auth failure it returns an error and
// the caller deletes the session.
func (a *Authenticator) Revalidate(ctx context.Context, in Tokens) (*Tokens, error) {
	if err := a.ensureReady(ctx); err != nil {
		return nil, err
	}
	src := a.oauth.TokenSource(ctx, &oauth2.Token{
		AccessToken:  in.AccessToken,
		RefreshToken: in.RefreshToken,
		Expiry:       in.Expiry,
	})
	tok, err := src.Token()
	if err != nil {
		return nil, fmt.Errorf("oidc: token refresh: %w", err)
	}
	if _, err := a.provider.UserInfo(ctx, oauth2.StaticTokenSource(tok)); err != nil {
		return nil, fmt.Errorf("oidc: userinfo: %w", err)
	}
	return &Tokens{
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		Expiry:       tok.Expiry,
	}, nil
}
