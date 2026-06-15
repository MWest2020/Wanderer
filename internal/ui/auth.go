package ui

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/MWest2020/wanderer/internal/auth/oidc"
	"github.com/MWest2020/wanderer/internal/store"
	"github.com/MWest2020/wanderer/pkg/models"
)

// Options carry the authentication wiring for Handler. The
// combinations that matter:
//
//   - zero Options                       → no auth (development mode)
//   - HtpasswdPath only                  → HTTP Basic on every route
//   - Auth only                          → OIDC login, redirect gate
//   - HtpasswdPath + Auth                → OIDC login, htpasswd as
//     break-glass (works even when the provider is unreachable)
type Options struct {
	// HtpasswdPath is the bcrypt htpasswd file protecting /ui/.
	// Empty disables Basic auth.
	HtpasswdPath string
	// Auth is the OIDC authenticator. Nil disables OIDC.
	Auth *oidc.Authenticator
	// MountPrefix is the externally visible path the UI is mounted
	// under (e.g. "/ui"), used to build absolute redirect targets.
	// Defaults to "/ui".
	MountPrefix string
	// SessionTTL is the hard lifetime of a login session. Defaults
	// to 12h.
	SessionTTL time.Duration
	// RevalidateInterval is the minimum time between userinfo
	// revalidations of a live session against the OIDC provider.
	// Zero means revalidate on every request (a Nextcloud-side
	// disable then cuts access on the very next request, at the
	// cost of one userinfo call per page load).
	RevalidateInterval time.Duration
	// CookieSecure sets the Secure flag on the session + state
	// cookies. Should be true behind TLS; false eases local http.
	CookieSecure bool

	// Scanner, when set (serve --ui-allow-scan, the dev-mode toggle),
	// enables the opt-in "Scan a target" form + the single sanctioned
	// mutating route POST /ui/scan. Nil keeps the UI fully read-only
	// (the prod default). Gate it behind Auth when the UI is exposed.
	Scanner ScanTrigger
}

// ScanTrigger runs a scan for the dev-mode UI scan form. It is
// satisfied by *scanner.Scanner; the ui package depends only on this
// narrow interface, not the scanner package.
type ScanTrigger interface {
	Scan(ctx context.Context, target models.Target) (*models.Scan, error)
}

const (
	sessionCookieName = "wanderer_session"
	stateCookieName   = "wanderer_oidc_state"
	pendingFlowTTL    = 10 * time.Minute
	defaultSessionTTL = 12 * time.Hour
)

// authGate enforces the session → htpasswd → OIDC-redirect policy
// and serves the login/callback/logout routes. A nil *authGate
// means "no authentication configured" and Handler leaves every
// route open.
type authGate struct {
	store        *store.Store
	auth         *oidc.Authenticator
	htpasswdPath string
	prefix       string
	sessionTTL   time.Duration
	revalidate   time.Duration
	cookieSecure bool

	mu      sync.Mutex
	pending map[string]pendingFlow // state → in-flight login
}

// pendingFlow is the server-side half of an in-flight login: the
// nonce expected back in the ID token, with an expiry so abandoned
// logins are swept rather than leaked.
type pendingFlow struct {
	nonce     string
	expiresAt time.Time
}

// newAuthGate returns the configured gate, or (nil, nil) when no
// authentication is configured (zero Options) so Handler can leave
// the routes open. An htpasswd path that fails to load is a hard
// error — the operator asked for it explicitly.
func newAuthGate(st *store.Store, opts Options) (*authGate, error) {
	if opts.HtpasswdPath == "" && opts.Auth == nil {
		return nil, nil
	}
	if opts.HtpasswdPath != "" {
		if _, err := LoadHtpasswd(opts.HtpasswdPath); err != nil {
			return nil, err
		}
	}
	prefix := opts.MountPrefix
	if prefix == "" {
		prefix = "/ui"
	}
	ttl := opts.SessionTTL
	if ttl <= 0 {
		ttl = defaultSessionTTL
	}
	return &authGate{
		store:        st,
		auth:         opts.Auth,
		htpasswdPath: opts.HtpasswdPath,
		prefix:       strings.TrimRight(prefix, "/"),
		sessionTTL:   ttl,
		revalidate:   opts.RevalidateInterval,
		cookieSecure: opts.CookieSecure,
		pending:      map[string]pendingFlow{},
	}, nil
}

// middleware is the request gate. Order: public routes pass; a
// valid session passes; valid Basic credentials pass (break-glass,
// works during an OIDC outage); otherwise redirect to the OIDC
// login or, htpasswd-only, issue a Basic challenge.
func (g *authGate) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if g.isPublicPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		if g.auth != nil && g.hasValidSession(w, r) {
			next.ServeHTTP(w, r)
			return
		}
		if g.htpasswdPath != "" && g.basicAuthOK(r) {
			next.ServeHTTP(w, r)
			return
		}
		if g.auth != nil {
			http.Redirect(w, r, g.prefix+"/login", http.StatusFound)
			return
		}
		w.Header().Set("WWW-Authenticate", `Basic realm="wanderer"`)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
}

// isPublicPath reports whether a route bypasses the gate. Paths are
// relative to the UI router root (after the /ui StripPrefix), so
// static assets and the OIDC dance must not require a session.
func (g *authGate) isPublicPath(p string) bool {
	switch p {
	case "/login", "/oauth/callback", "/logout":
		return true
	}
	return strings.HasPrefix(p, "/static/")
}

// basicAuthOK re-reads the htpasswd file on every request so an
// operator can rotate credentials without restarting the process.
func (g *authGate) basicAuthOK(r *http.Request) bool {
	user, pass, ok := r.BasicAuth()
	if !ok {
		return false
	}
	creds, err := LoadHtpasswd(g.htpasswdPath)
	if err != nil {
		return false
	}
	return verifyAgainst(creds, user, pass)
}

// hasValidSession looks up the session cookie and, when the
// revalidation window has elapsed, confirms the session is still
// honoured by the OIDC provider (refreshing the access token and
// calling userinfo). A failed revalidation deletes the session so
// a Nextcloud-side disable cuts access. The refreshed token set is
// persisted back.
func (g *authGate) hasValidSession(w http.ResponseWriter, r *http.Request) bool {
	c, err := r.Cookie(sessionCookieName)
	if err != nil || c.Value == "" {
		return false
	}
	ctx := r.Context()
	sess, err := g.store.GetSession(ctx, c.Value)
	if err != nil {
		g.clearCookie(w, sessionCookieName)
		return false
	}
	if time.Since(sess.LastValidatedAt) < g.revalidate {
		return true
	}
	toks, err := g.auth.Revalidate(ctx, oidc.Tokens{
		AccessToken:  sess.AccessToken,
		RefreshToken: sess.RefreshToken,
		Expiry:       sess.TokenExpiry,
	})
	if err != nil {
		_ = g.store.DeleteSession(ctx, sess.ID)
		g.clearCookie(w, sessionCookieName)
		return false
	}
	sess.AccessToken = toks.AccessToken
	sess.RefreshToken = toks.RefreshToken
	sess.TokenExpiry = toks.Expiry
	_ = g.store.RefreshSession(ctx, sess)
	return true
}

// loginHandler starts an authorization-code flow: it mints a state
// + nonce, records the nonce server-side keyed by state, pins the
// state to the browser with a short-lived cookie, and redirects to
// the provider's authorize endpoint.
func (g *authGate) loginHandler(w http.ResponseWriter, r *http.Request) {
	state, err := randomToken()
	if err != nil {
		http.Error(w, "login unavailable", http.StatusInternalServerError)
		return
	}
	nonce, err := randomToken()
	if err != nil {
		http.Error(w, "login unavailable", http.StatusInternalServerError)
		return
	}
	g.putPending(state, nonce)
	g.setCookie(w, stateCookieName, state, pendingFlowTTL)
	authURL, err := g.auth.AuthCodeURL(r.Context(), state, nonce)
	if err != nil {
		// Discovery failed — the provider is unreachable. Operators
		// with htpasswd configured still have break-glass access.
		http.Error(w, "oidc provider unavailable: "+err.Error(), http.StatusServiceUnavailable)
		return
	}
	http.Redirect(w, r, authURL, http.StatusFound)
}

// callbackHandler completes the flow: it double-checks the state
// (query vs the browser-pinned cookie and the server-side record),
// exchanges the code, verifies the ID token, and establishes a
// session.
func (g *authGate) callbackHandler(w http.ResponseWriter, r *http.Request) {
	queryState := r.URL.Query().Get("state")
	cookieState, err := r.Cookie(stateCookieName)
	if err != nil || queryState == "" || queryState != cookieState.Value {
		http.Error(w, "oidc state mismatch", http.StatusBadRequest)
		return
	}
	nonce, ok := g.takePending(queryState)
	if !ok {
		http.Error(w, "oidc state expired or unknown", http.StatusBadRequest)
		return
	}
	g.clearCookie(w, stateCookieName)

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "oidc callback missing code", http.StatusBadRequest)
		return
	}
	ctx := r.Context()
	claims, tokens, err := g.auth.Exchange(ctx, code, nonce)
	if err != nil {
		http.Error(w, "oidc exchange failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	sessionID, err := randomToken()
	if err != nil {
		http.Error(w, "session create failed", http.StatusInternalServerError)
		return
	}
	now := time.Now().UTC()
	sess := &store.Session{
		ID:           sessionID,
		Subject:      claims.Subject,
		Email:        claims.Email,
		Name:         claims.Name,
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		TokenExpiry:  tokens.Expiry,
		ExpiresAt:    now.Add(g.sessionTTL),
	}
	if err := g.store.CreateSession(ctx, sess); err != nil {
		http.Error(w, "session create failed", http.StatusInternalServerError)
		return
	}
	g.setCookie(w, sessionCookieName, sess.ID, g.sessionTTL)
	http.Redirect(w, r, g.prefix+"/", http.StatusFound)
}

// logoutHandler deletes the server-side session and clears the
// cookie. It does not log the operator out of Nextcloud itself.
func (g *authGate) logoutHandler(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookieName); err == nil && c.Value != "" {
		_ = g.store.DeleteSession(r.Context(), c.Value)
	}
	g.clearCookie(w, sessionCookieName)
	http.Redirect(w, r, g.prefix+"/", http.StatusFound)
}

// putPending records the nonce for an in-flight login keyed by
// state, sweeping expired entries opportunistically.
func (g *authGate) putPending(state, nonce string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	now := time.Now()
	for k, v := range g.pending {
		if now.After(v.expiresAt) {
			delete(g.pending, k)
		}
	}
	g.pending[state] = pendingFlow{nonce: nonce, expiresAt: now.Add(pendingFlowTTL)}
}

// takePending consumes the pending entry for state, returning the
// nonce and whether it was present and unexpired. The entry is
// always removed (single-use).
func (g *authGate) takePending(state string) (string, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	pf, ok := g.pending[state]
	if !ok {
		return "", false
	}
	delete(g.pending, state)
	if time.Now().After(pf.expiresAt) {
		return "", false
	}
	return pf.nonce, true
}

func (g *authGate) setCookie(w http.ResponseWriter, name, value string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     g.prefix,
		MaxAge:   int(ttl.Seconds()),
		HttpOnly: true,
		Secure:   g.cookieSecure,
		// Lax (not Strict): the OIDC callback is a top-level
		// cross-site navigation back from Nextcloud, and Strict
		// would withhold the state cookie on that request.
		SameSite: http.SameSiteLaxMode,
	})
}

func (g *authGate) clearCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     g.prefix,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   g.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

// randomToken returns a 256-bit URL-safe random string for state,
// nonce, and session cookie values. A crypto/rand failure is
// surfaced rather than swallowed: silently returning an all-zero
// (predictable) token would defeat the CSRF state, the nonce, and
// the session identifier all at once, so callers must fail the
// request instead.
func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("ui: read random token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
