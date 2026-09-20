package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/MWest2020/wanderer/pkg/models"
)

// ErrEnrolmentFailed is returned for every enrolment failure — an
// unknown token, an expired one, or one already exchanged. Callers
// must not translate this into three different responses: that
// distinction is exactly what an attacker probing enrolment would
// want to learn.
var ErrEnrolmentFailed = errors.New("store: enrolment failed")

// secretBytes is how many random bytes back a plain token or agent
// secret before hex-encoding (256 bits).
const secretBytes = 32

// hashSecret returns the hex-encoded SHA-256 digest of raw. Used for
// both enrolment tokens and agent secrets: only the digest is ever
// persisted, so a database leak cannot hand out a working credential.
func hashSecret(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// generateSecret returns secretBytes of cryptographically random
// data, hex-encoded.
func generateSecret() (string, error) {
	buf := make([]byte, secretBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("store: generate secret: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

// CreateEnrolmentToken issues a new enrolment token valid for ttl.
// It returns the plain token — shown to the operator exactly once —
// and the persisted record, which carries only the token's hash.
func (s *Store) CreateEnrolmentToken(ctx context.Context, ttl time.Duration) (plainToken string, tok *models.EnrolmentToken, err error) {
	if ttl <= 0 {
		return "", nil, errors.New("store: enrolment token ttl must be positive")
	}
	plainToken, err = generateSecret()
	if err != nil {
		return "", nil, err
	}
	tok = &models.EnrolmentToken{
		ID:        newID("tok"),
		TokenHash: hashSecret(plainToken),
		ExpiresAt: time.Now().UTC().Add(ttl),
	}
	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO enrolment_tokens (id, token_hash, expires_at) VALUES (?,?,?)`,
		tok.ID, tok.TokenHash, tok.ExpiresAt); err != nil {
		return "", nil, fmt.Errorf("store: insert enrolment token: %w", err)
	}
	return plainToken, tok, nil
}

// EnrolAgent exchanges a plain enrolment token for a new agent
// secret. The token must exist, be unexpired, and be unused; on
// success it is marked used (so it cannot be exchanged again) and a
// new Agent row is recorded for hostname. The plain secret is
// returned exactly once — only its hash is persisted.
//
// Every failure path (unknown token, expired token, already-used
// token) returns ErrEnrolmentFailed, so the caller cannot tell which
// of the three happened.
func (s *Store) EnrolAgent(ctx context.Context, plainToken, hostname string) (plainSecret string, ag *models.Agent, err error) {
	host, err := models.NormaliseHost(hostname)
	if err != nil {
		return "", nil, ErrEnrolmentFailed
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", nil, fmt.Errorf("store: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	row := tx.QueryRowContext(ctx,
		`SELECT id, expires_at, used_at FROM enrolment_tokens WHERE token_hash = ?`,
		hashSecret(plainToken))
	var tokID string
	var expiresAt time.Time
	var usedAt sql.NullTime
	switch scanErr := row.Scan(&tokID, &expiresAt, &usedAt); {
	case errors.Is(scanErr, sql.ErrNoRows):
		return "", nil, ErrEnrolmentFailed
	case scanErr != nil:
		return "", nil, fmt.Errorf("store: lookup enrolment token: %w", scanErr)
	}
	now := time.Now().UTC()
	if usedAt.Valid || now.After(expiresAt) {
		return "", nil, ErrEnrolmentFailed
	}

	plainSecret, err = generateSecret()
	if err != nil {
		return "", nil, err
	}
	ag = &models.Agent{
		ID:         newID("agt"),
		Hostname:   host,
		SecretHash: hashSecret(plainSecret),
		EnrolledAt: now,
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO agents (id, hostname, secret_hash, enrolled_at) VALUES (?,?,?,?)`,
		ag.ID, ag.Hostname, ag.SecretHash, ag.EnrolledAt); err != nil {
		return "", nil, fmt.Errorf("store: insert agent: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE enrolment_tokens SET used_at = ? WHERE id = ?`, now, tokID); err != nil {
		return "", nil, fmt.Errorf("store: mark enrolment token used: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return "", nil, fmt.Errorf("store: commit: %w", err)
	}
	return plainSecret, ag, nil
}

// RevokeAgent marks hostname's agent as revoked. Idempotent — calling
// it again on an already-revoked agent leaves the original revoked_at
// untouched and returns success. An unknown hostname returns
// ErrNotFound.
func (s *Store) RevokeAgent(ctx context.Context, hostname string) error {
	host, err := models.NormaliseHost(hostname)
	if err != nil {
		return err
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE agents SET revoked_at = ? WHERE hostname = ? AND revoked_at IS NULL`,
		time.Now().UTC(), host)
	if err != nil {
		return fmt.Errorf("store: revoke agent: %w", err)
	}
	if n, _ := res.RowsAffected(); n > 0 {
		return nil
	}
	var exists bool
	if err := s.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM agents WHERE hostname = ?)`, host).Scan(&exists); err != nil {
		return fmt.Errorf("store: check agent existence: %w", err)
	}
	if !exists {
		return ErrNotFound
	}
	return nil
}

// GetAgentByHostname returns the agent row for hostname, or
// ErrNotFound if no agent ever enrolled with that hostname.
func (s *Store) GetAgentByHostname(ctx context.Context, hostname string) (*models.Agent, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, hostname, secret_hash, enrolled_at, revoked_at FROM agents WHERE hostname = ?`, hostname)
	return scanAgent(row)
}

// ListAgents returns every agent, ordered by hostname.
func (s *Store) ListAgents(ctx context.Context) ([]models.Agent, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, hostname, secret_hash, enrolled_at, revoked_at FROM agents ORDER BY hostname`)
	if err != nil {
		return nil, fmt.Errorf("store: list agents: %w", err)
	}
	defer rows.Close()
	var out []models.Agent
	for rows.Next() {
		a, err := scanAgent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

// scanAgent uses the rowScanner abstraction (see assessment.go) so it
// serves both a single-row lookup and a Rows loop.
func scanAgent(row rowScanner) (*models.Agent, error) {
	a := &models.Agent{}
	var revokedAt sql.NullTime
	if err := row.Scan(&a.ID, &a.Hostname, &a.SecretHash, &a.EnrolledAt, &revokedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("store: select agent: %w", err)
	}
	if revokedAt.Valid {
		t := revokedAt.Time
		a.RevokedAt = &t
	}
	return a, nil
}
