package nextcloud

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/MWest2020/wanderer/internal/probe/egress"
	"github.com/MWest2020/wanderer/internal/store"
	"github.com/MWest2020/wanderer/pkg/models"
)

const (
	// publishTimeout bounds one full publish (both files + the
	// MKCOLs) so the post-scan hook cannot hang a scan caller.
	publishTimeout = 30 * time.Second
	// publishAttempts is the bounded retry count per file.
	publishAttempts = 3
	retryBackoff    = 500 * time.Millisecond
)

// Publisher gathers a completed scan's artefacts from the store,
// redacts them per ADR-0008, and drops a JSON-LD + Markdown bundle
// into Nextcloud. It satisfies the scanner's post-scan Publisher
// seam: Publish never returns an error and never blocks scan
// completion — a publish failure is logged and the local /ui/ view
// stays authoritative.
type Publisher struct {
	store  *store.Store
	client *Client
	logger *slog.Logger
}

// NewPublisher wires the store, WebDAV client, and logger.
func NewPublisher(st *store.Store, client *Client, logger *slog.Logger) *Publisher {
	if logger == nil {
		logger = slog.Default()
	}
	return &Publisher{store: st, client: client, logger: logger}
}

// Publish runs the post-scan hook for scanID. It is fire-and-forget
// from the scanner's perspective: it owns its own bounded context
// (independent of the scan's budget) and logs failures rather than
// propagating them.
func (p *Publisher) Publish(scanID string) {
	ctx, cancel := context.WithTimeout(context.Background(), publishTimeout)
	defer cancel()
	if err := p.publish(ctx, scanID); err != nil {
		p.logger.Error("wanderer.nextcloud.publish.error", "scan_id", scanID, "err", err)
	}
}

func (p *Publisher) publish(ctx context.Context, scanID string) error {
	scan, err := p.store.GetScan(ctx, scanID)
	if err != nil {
		return fmt.Errorf("load scan: %w", err)
	}
	orgSlug, subject := p.resolveScope(ctx, scan)
	assessments, err := p.store.ListAssessmentsForScan(ctx, scanID)
	if err != nil {
		// Assessments are optional (a scheduled scan may not be
		// assessed yet); a lookup error is non-fatal — publish the
		// findings-only bundle.
		p.logger.Warn("wanderer.nextcloud.assessments_unavailable", "scan_id", scanID, "err", err)
		assessments = nil
	}

	bundle := buildBundle(scan, assessments, subject)
	jsonld, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json-ld: %w", err)
	}
	markdown := buildMarkdown(scan, assessments, subject)

	// If the .md PUT fails after the .jsonld succeeded, the JSON-LD
	// is left without its Markdown sibling. We intentionally do not
	// clean it up: the next scan writes a fresh pair under a new
	// scan ID (no overwrite inconsistency), and the local UI stays
	// authoritative — a lone .jsonld is harmless.
	if err := p.putWithRetry(ctx, orgSlug, scan.ID+".jsonld", "application/ld+json", jsonld); err != nil {
		return err
	}
	if err := p.putWithRetry(ctx, orgSlug, scan.ID+".md", "text/markdown; charset=utf-8", []byte(markdown)); err != nil {
		return err
	}
	p.logger.Info("wanderer.nextcloud.published", "scan_id", scan.ID, "org", orgSlug, "findings", len(scan.Findings))
	return nil
}

func (p *Publisher) putWithRetry(ctx context.Context, orgSlug, name, contentType string, body []byte) error {
	var lastErr error
	for attempt := 1; attempt <= publishAttempts; attempt++ {
		if err := p.client.PutFile(ctx, orgSlug, name, contentType, body); err == nil {
			return nil
		} else {
			lastErr = err
		}
		if attempt == publishAttempts {
			break // no backoff after the final attempt
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("publish %s: %w", name, ctx.Err())
		case <-time.After(retryBackoff):
		}
	}
	return fmt.Errorf("publish %s after %d attempts: %w", name, publishAttempts, lastErr)
}

// resolveScope returns the organisation slug used as the publish
// sub-directory and a human subject (the target domain) for the
// bundle. Both fall back to safe defaults when the lookups fail —
// scope is a filing convenience, never a reason to drop the bundle.
func (p *Publisher) resolveScope(ctx context.Context, scan *models.Scan) (orgSlug, subject string) {
	orgSlug = "unscoped"
	t, err := p.store.GetTarget(ctx, scan.TargetID)
	if err != nil || t == nil {
		return orgSlug, scan.TargetID
	}
	subject = t.Domain
	if o, oerr := p.store.GetOrganisation(ctx, t.OrganisationID); oerr == nil && o != nil && o.Slug != "" {
		orgSlug = o.Slug
	}
	return orgSlug, subject
}

// buildBundle assembles the JSON-LD document for a scan: a small
// @context, the scan metadata, redacted findings, and any
// assessments. Findings are redacted per ADR-0008 and stripped of
// raw Evidence before serialisation.
func buildBundle(scan *models.Scan, assessments []models.Assessment, subject string) map[string]any {
	findings := make([]models.Finding, 0, len(scan.Findings))
	for _, f := range scan.Findings {
		findings = append(findings, redactFinding(f))
	}
	doc := map[string]any{
		"@context": map[string]any{
			"@vocab": "https://wanderer.observer/ns#",
			"schema": "https://schema.org/",
		},
		"@type":      "SovereigntyScan",
		"id":         scan.ID,
		"subject":    subject,
		"status":     string(scan.Status),
		"started_at": scan.StartedAt.UTC().Format(time.RFC3339),
		"findings":   findings,
	}
	if scan.EndedAt != nil {
		doc["ended_at"] = scan.EndedAt.UTC().Format(time.RFC3339)
	}
	if len(assessments) > 0 {
		doc["assessments"] = assessments
	}
	return doc
}

// buildMarkdown renders a human-readable summary that opens nicely
// in Nextcloud's text app: a header, the per-framework verdict, and
// a findings-by-probe roll-up.
func buildMarkdown(scan *models.Scan, assessments []models.Assessment, subject string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Wanderer scan — %s\n\n", subject)
	fmt.Fprintf(&b, "- **Scan ID:** `%s`\n", scan.ID)
	fmt.Fprintf(&b, "- **Status:** %s\n", scan.Status)
	fmt.Fprintf(&b, "- **Started:** %s\n", scan.StartedAt.UTC().Format(time.RFC3339))
	if scan.EndedAt != nil {
		fmt.Fprintf(&b, "- **Ended:** %s\n", scan.EndedAt.UTC().Format(time.RFC3339))
	}
	fmt.Fprintf(&b, "- **Findings:** %d\n\n", len(scan.Findings))

	if len(assessments) > 0 {
		b.WriteString("## Verdict\n\n")
		for _, a := range assessments {
			fmt.Fprintf(&b, "### %s\n\n", a.Framework)
			for _, d := range a.Dimensions {
				fmt.Fprintf(&b, "- %s: **%s** (%s)\n", d.Dimension, d.Score, d.Completeness)
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("## Findings by probe\n\n")
	byProbe := map[string]int{}
	for _, f := range scan.Findings {
		prefix := f.ProbeID
		if i := strings.IndexByte(prefix, '.'); i >= 0 {
			prefix = prefix[:i]
		}
		byProbe[prefix]++
	}
	for _, prefix := range sortedKeys(byProbe) {
		fmt.Fprintf(&b, "- `%s`: %d\n", prefix, byProbe[prefix])
	}
	b.WriteString("\n_Published by Wanderer. The authoritative view lives in the operator UI._\n")
	return b.String()
}

// redactFinding returns a copy of f with secret-shaped attribute
// values scrubbed (ADR-0008) and raw Evidence dropped. The original
// is never mutated.
func redactFinding(f models.Finding) models.Finding {
	f.Evidence = nil
	f.Attributes = redactValue("", f.Attributes).(map[string]any)
	return f
}

// redactValue walks an attribute value, applying egress.Apply to
// every string regardless of nesting depth and recursing into maps
// and slices. The key carried in is the nearest enclosing map key,
// so egress.Apply's key-name heuristics still fire on strings buried
// inside slices; its value-shape heuristics fire regardless. Non-
// string scalars pass through unchanged.
func redactValue(key string, v any) any {
	switch val := v.(type) {
	case string:
		red, _ := egress.Apply(key, val)
		return red
	case map[string]any:
		out := make(map[string]any, len(val))
		for k, inner := range val {
			out[k] = redactValue(k, inner)
		}
		return out
	case []any:
		out := make([]any, len(val))
		for i, inner := range val {
			out[i] = redactValue(key, inner)
		}
		return out
	default:
		return v
	}
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// Small maps; an insertion sort keeps the dependency surface nil.
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j-1] > keys[j]; j-- {
			keys[j-1], keys[j] = keys[j], keys[j-1]
		}
	}
	return keys
}
