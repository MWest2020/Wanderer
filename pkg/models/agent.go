package models

import "time"

// Agent is one enrolled agent host: once enrolled it may deliver
// findings to the core using the secret it received when it
// exchanged its enrolment token. Only the secret's hash is
// persisted — the plain value is shown exactly once, at enrolment
// time — so a database leak never hands out a working credential.
type Agent struct {
	ID         string
	Hostname   string
	SecretHash string
	EnrolledAt time.Time
	RevokedAt  *time.Time
}

// Revoked reports whether an operator has revoked this agent.
func (a *Agent) Revoked() bool { return a.RevokedAt != nil }

// EnrolmentToken is a short-lived, single-use token an operator
// creates so a new agent can exchange it for its own secret. Only
// the token's hash is persisted; the plain value is shown exactly
// once, at creation time.
type EnrolmentToken struct {
	ID        string
	TokenHash string
	ExpiresAt time.Time
	UsedAt    *time.Time
}

// Used reports whether this token has already been exchanged.
func (t *EnrolmentToken) Used() bool { return t.UsedAt != nil }
