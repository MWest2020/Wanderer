package models

import (
	"errors"
	"fmt"
	"regexp"
	"time"
)

// Organisation groups Targets (perimeter domains and agent
// hosts) under one operator-facing handle. Every Target SHALL
// belong to exactly one Organisation; the migration that
// introduced the entity seeds a `default` Organisation and
// attaches every pre-existing Target to it.
type Organisation struct {
	ID          string    `json:"id"`
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
}

// DefaultOrganisationID is the seeded ID used by migration 005
// for the `default` organisation that picks up every
// pre-pivot Target during backfill. Code that needs to attach
// a Target without an explicit slug uses this constant directly
// and avoids a slug → ID round-trip.
const DefaultOrganisationID = "o_default"

// DefaultOrganisationSlug is the matching slug operators see in
// the CLI / UI / MCP surface until they rename the organisation
// via `wanderer org rename`.
const DefaultOrganisationSlug = "default"

// slugPattern enforces the slug rules: 2..40 chars, lowercase
// letters, digits, and hyphens; MUST NOT start or end with a
// hyphen. The same regex backs both the model's Validate and
// the SQLite uniqueness check at the store layer.
var slugPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,38}[a-z0-9])?$`)

// Validate enforces the slug + name invariants. A zero-value
// Organisation (empty slug + name) is invalid; the seed migration
// inserts a fully-populated row so this never fails at runtime.
func (o *Organisation) Validate() error {
	if o.Slug == "" {
		return errors.New("organisation: slug is required")
	}
	if len(o.Slug) < 2 || len(o.Slug) > 40 {
		return fmt.Errorf("organisation: slug %q must be 2..40 characters", o.Slug)
	}
	if !slugPattern.MatchString(o.Slug) {
		return fmt.Errorf("organisation: slug %q must be lowercase letters/digits/hyphens, no leading or trailing hyphen", o.Slug)
	}
	if o.Name == "" {
		return errors.New("organisation: name is required")
	}
	return nil
}
