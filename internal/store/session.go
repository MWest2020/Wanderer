package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Session is one active operator browser session established
// through the OIDC login flow. ID is the opaque cookie value;
// Subject/Email/Name come from the verified ID token. The token
// fields let the UI gate revalidate against the OIDC provider's
// userinfo endpoint (and refresh an expired access token) so a
// Nextcloud-side disable cuts access on the next request.
type Session struct {
	ID              string
	Subject         string
	Email           string
	Name            string
	AccessToken     string
	RefreshToken    string
	TokenExpiry     time.Time // zero when the provider returned no expiry
	LastValidatedAt time.Time
	ExpiresAt       time.Time
	CreatedAt       time.Time
}

// CreateSession persists a new session row. The caller supplies
// the opaque ID (a cryptographically random token) and the hard
// expiry; CreatedAt and LastValidatedAt are stamped to now.
func (s *Store) CreateSession(ctx context.Context, sess *Session) error {
	if sess.ID == "" {
		return fmt.Errorf("store: session id required")
	}
	now := time.Now().UTC()
	sess.CreatedAt = now
	sess.LastValidatedAt = now
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO ui_sessions
		   (id, subject, email, name, access_token, refresh_token, token_expiry, last_validated_at, expires_at, created_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?)`,
		sess.ID, sess.Subject, sess.Email, sess.Name,
		sess.AccessToken, sess.RefreshToken, nullTime(sess.TokenExpiry),
		sess.LastValidatedAt, sess.ExpiresAt, sess.CreatedAt)
	if err != nil {
		return fmt.Errorf("store: insert session: %w", err)
	}
	return nil
}

// GetSession returns a session by its opaque ID. A session whose
// ExpiresAt is in the past is treated as absent (and lazily
// deleted) so an expired cookie never authenticates.
func (s *Store) GetSession(ctx context.Context, id string) (*Session, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, subject, email, name, access_token, refresh_token,
		        token_expiry, last_validated_at, expires_at, created_at
		 FROM ui_sessions WHERE id = ?`, id)
	sess := &Session{}
	var tokenExpiry sql.NullTime
	if err := row.Scan(&sess.ID, &sess.Subject, &sess.Email, &sess.Name,
		&sess.AccessToken, &sess.RefreshToken, &tokenExpiry,
		&sess.LastValidatedAt, &sess.ExpiresAt, &sess.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("store: select session: %w", err)
	}
	if tokenExpiry.Valid {
		sess.TokenExpiry = tokenExpiry.Time
	}
	if time.Now().UTC().After(sess.ExpiresAt) {
		_ = s.DeleteSession(ctx, id)
		return nil, ErrNotFound
	}
	return sess, nil
}

// RefreshSession updates the mutable token + validation fields of
// an existing session after a successful revalidation or token
// refresh. The session's identity (subject/email/name) and hard
// expiry are immutable for the lifetime of the cookie.
func (s *Store) RefreshSession(ctx context.Context, sess *Session) error {
	sess.LastValidatedAt = time.Now().UTC()
	_, err := s.db.ExecContext(ctx,
		`UPDATE ui_sessions
		   SET access_token = ?, refresh_token = ?, token_expiry = ?, last_validated_at = ?
		 WHERE id = ?`,
		sess.AccessToken, sess.RefreshToken, nullTime(sess.TokenExpiry),
		sess.LastValidatedAt, sess.ID)
	if err != nil {
		return fmt.Errorf("store: refresh session: %w", err)
	}
	return nil
}

// DeleteSession removes a session row. Used on logout and when a
// revalidation against the OIDC provider fails.
func (s *Store) DeleteSession(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM ui_sessions WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("store: delete session: %w", err)
	}
	return nil
}

// DeleteExpiredSessions purges sessions past their hard expiry. A
// caller can run this periodically; correctness does not depend on
// it because GetSession also treats an expired row as absent.
func (s *Store) DeleteExpiredSessions(ctx context.Context) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM ui_sessions WHERE expires_at < ?`, time.Now().UTC())
	if err != nil {
		return 0, fmt.Errorf("store: purge sessions: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// nullTime maps a zero time.Time to a SQL NULL so the optional
// token_expiry column round-trips as NULL rather than year-0001.
func nullTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.UTC()
}
