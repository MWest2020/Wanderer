package scanner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/MWest2020/wanderer/internal/store"
	"github.com/MWest2020/wanderer/pkg/models"
)

// NetnlSchemaV1 is the only netnl-findings schema version this
// importer understands. A file declaring anything else aborts the
// import before anything is persisted (spec.md "Schema version
// mismatch").
const NetnlSchemaV1 = "netnl-findings/v1"

// NetnlDomain is one entry of the netnl-findings/v1 "domains" array —
// one Internet.nl measurement (web or mail) for one domain. Measured
// against a real batch v2.7.0 export (see design.md "Design gate
// outcome"): there is no per-domain request ID, and category is
// carried on each NetnlResult, not on the domain block.
type NetnlDomain struct {
	Domain       string        `json:"domain"`
	Type         string        `json:"type"` // "web" | "mail"
	Status       string        `json:"status"`
	MeasuredAt   time.Time     `json:"measured_at"`
	ScorePercent int           `json:"score_percent"`
	ReportURL    string        `json:"report_url"`
	Results      []NetnlResult `json:"results"`
}

// NetnlResult is one Internet.nl subtest result. detail is measured
// to be always null on the real API (design.md finding 1) — it stays
// an opaque `any` so a future API version can populate it without a
// schema bump, and no rule may be built on it.
type NetnlResult struct {
	Test     string `json:"test"`
	Category string `json:"category"`
	Status   string `json:"status"`
	Verdict  string `json:"verdict"`
	Detail   any    `json:"detail"`
}

// NetnlFindingsFile is a parsed netnl-findings/v1 document.
type NetnlFindingsFile struct {
	Schema  string
	Domains []NetnlDomain
}

// LoadNetnlFindings reads a netnl-findings/v1 file from path. Malformed
// domain entries are logged at WARN and skipped — following the Amass
// precedent (internal/scanner/amass.go), a partial file must not block
// the rest of the import. A schema version other than NetnlSchemaV1 is
// fatal: the caller must abort before persisting anything.
func LoadNetnlFindings(path string, logger *slog.Logger) (*NetnlFindingsFile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("netnl: open %s: %w", path, err)
	}
	defer f.Close()
	if logger == nil {
		logger = slog.Default()
	}
	return ParseNetnlFindings(f, logger)
}

// ParseNetnlFindings parses a netnl-findings/v1 document from r. The
// top-level object is walked token by token so that a malformed
// element in the middle or at the end of "domains" (spec.md "Truncated
// file") stops the array walk without losing the entries already
// decoded, while a malformed-but-still-valid-JSON entry (missing a
// required field) is warned about and skipped without affecting its
// neighbours.
func ParseNetnlFindings(r io.Reader, logger *slog.Logger) (*NetnlFindingsFile, error) {
	if logger == nil {
		logger = slog.Default()
	}
	dec := json.NewDecoder(r)

	if t, err := dec.Token(); err != nil || t != json.Delim('{') {
		return nil, fmt.Errorf("netnl: not a JSON object")
	}

	out := &NetnlFindingsFile{}
	for dec.More() {
		keyTok, err := dec.Token()
		if err != nil {
			return nil, fmt.Errorf("netnl: read key: %w", err)
		}
		key, _ := keyTok.(string)
		switch key {
		case "schema":
			if err := dec.Decode(&out.Schema); err != nil {
				return nil, fmt.Errorf("netnl: decode schema: %w", err)
			}
		case "domains":
			domains, truncated, err := decodeNetnlDomains(dec, logger)
			if err != nil {
				return nil, err
			}
			out.Domains = domains
			if truncated {
				// The stream broke mid-array (spec.md "Truncated file"):
				// there is nothing reliable left to read after it, so
				// stop here with whatever valid entries were decoded
				// rather than trying to read further top-level keys.
				if out.Schema != NetnlSchemaV1 {
					return nil, fmt.Errorf("netnl: unsupported schema %q, expected %q", out.Schema, NetnlSchemaV1)
				}
				return out, nil
			}
		default:
			// Forward-compatible: skip any field this importer does not
			// know about rather than failing the whole file.
			var discard json.RawMessage
			if err := dec.Decode(&discard); err != nil {
				return nil, fmt.Errorf("netnl: skip field %q: %w", key, err)
			}
		}
	}

	if out.Schema != NetnlSchemaV1 {
		return nil, fmt.Errorf("netnl: unsupported schema %q, expected %q", out.Schema, NetnlSchemaV1)
	}
	return out, nil
}

// decodeNetnlDomains walks the "domains" array. A token-level decode
// error (invalid JSON — e.g. a truncated final entry) stops the walk,
// reports truncated=true, and returns the entries decoded so far; a
// structurally valid entry that fails NetnlDomain's own field
// requirements is warned about and skipped without affecting the walk
// or the stream position.
func decodeNetnlDomains(dec *json.Decoder, logger *slog.Logger) (domains []NetnlDomain, truncated bool, err error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, false, fmt.Errorf("netnl: read domains: %w", err)
	}
	if tok != json.Delim('[') {
		return nil, false, fmt.Errorf("netnl: \"domains\" is not an array")
	}

	var out []NetnlDomain
	for dec.More() {
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			logger.Warn("scanner.netnl.malformed_entry", "err", err.Error(), "action", "stopping (rest of file unreadable)")
			return out, true, nil
		}
		var d NetnlDomain
		if err := json.Unmarshal(raw, &d); err != nil {
			logger.Warn("scanner.netnl.malformed_entry", "err", err.Error(), "action", "skipped")
			continue
		}
		if reason := netnlDomainInvalidReason(d); reason != "" {
			logger.Warn("scanner.netnl.malformed_entry", "reason", reason, "domain", d.Domain, "action", "skipped")
			continue
		}
		out = append(out, d)
	}
	// Consume the closing ']' — dec.More() already returned false, so
	// Token() here only fails if the stream is broken in some other
	// way, which the caller surfaces as a parse error.
	if _, err := dec.Token(); err != nil {
		return nil, false, fmt.Errorf("netnl: close domains array: %w", err)
	}
	return out, false, nil
}

// netnlDomainInvalidReason reports why d is not usable, or "" if it
// is well-formed. A domain measured with status "error" and empty
// Results is valid on purpose (design.md "valkuil": that means
// "measured, but it broke", not "not delivered") — it must not be
// rejected here.
func netnlDomainInvalidReason(d NetnlDomain) string {
	if d.Domain == "" {
		return "missing domain"
	}
	if d.Type != "web" && d.Type != "mail" {
		return fmt.Sprintf("unknown type %q", d.Type)
	}
	if d.Status == "" {
		return "missing status"
	}
	if d.MeasuredAt.IsZero() {
		return "missing or unparseable measured_at"
	}
	return ""
}

// NetnlFindings converts one parsed NetnlDomain into the Findings the
// importer persists — ProbeID "internetnl.<type>.<test>" per subtest,
// carrying the API's verdict verbatim plus category, measured_at, and
// the (opaque, never self-constructed — NETNL-CONTRACT.md requirement
// 5) report URL as evidence. A domain with no Results (the status
// "error" valkuil from design.md — measured, but it broke, not "not
// delivered") yields a single domain-level status Finding instead of
// per-test ones, since there is nothing per-test to carry.
func NetnlFindings(d NetnlDomain) []models.Finding {
	measuredAt := d.MeasuredAt.UTC().Format(time.RFC3339)
	if len(d.Results) == 0 {
		return []models.Finding{
			{
				ProbeID:     fmt.Sprintf("internetnl.%s.status", d.Type),
				SourceModus: models.SourceModusImport,
				Subject:     d.Domain,
				Severity:    netnlDomainSeverity(d.Status),
				Attributes: map[string]any{
					"domain":        d.Domain,
					"type":          d.Type,
					"status":        d.Status,
					"measured_at":   measuredAt,
					"score_percent": d.ScorePercent,
					"report_url":    d.ReportURL,
				},
			},
		}
	}
	out := make([]models.Finding, 0, len(d.Results))
	for _, r := range d.Results {
		out = append(out, models.Finding{
			ProbeID:     fmt.Sprintf("internetnl.%s.%s", d.Type, r.Test),
			SourceModus: models.SourceModusImport,
			Subject:     d.Domain,
			Severity:    netnlResultSeverity(r.Status),
			Attributes: map[string]any{
				"domain":        d.Domain,
				"type":          d.Type,
				"test":          r.Test,
				"category":      r.Category,
				"status":        r.Status,
				"verdict":       r.Verdict,
				"detail":        r.Detail,
				"measured_at":   measuredAt,
				"score_percent": d.ScorePercent,
				"report_url":    d.ReportURL,
			},
		})
	}
	return out
}

// netnlResultSeverity gives a per-test Finding its coarse
// classification from the API's own status. This is a triage hint
// only — Fine-grained scoring (the standards dimension's verdict
// mapping) is the assessor's job, not this importer's; see
// design.md "Verdict mapping".
func netnlResultSeverity(status string) models.Severity {
	switch status {
	case "passed", "not_tested":
		return models.SeverityInfo
	case "info":
		return models.SeverityObservation
	case "warning":
		return models.SeverityConcern
	case "failed":
		return models.SeverityFinding
	case "error":
		return models.SeverityConcern
	default:
		return models.SeverityObservation
	}
}

// netnlDomainSeverity classifies the single status Finding emitted
// for a domain with no per-test results.
func netnlDomainSeverity(status string) models.Severity {
	if status == "error" {
		return models.SeverityConcern
	}
	return models.SeverityInfo
}

// ImportNetnlDomains matches each parsed domain entry to an existing
// target, groups entries by target (a batch file can carry both a web
// and a mail entry for the same domain), and persists one import-kind
// scan per matched target. Unknown domains are logged at WARN and
// skipped — spec.md "Unknown domains SHALL be logged at WARN and
// skipped".
//
// This is the one write path for netnl imports: both the CLI
// (`wanderer import internetnl`) and the HTTP route
// (`POST /imports/internetnl`) call through here rather than each
// having their own copy — the server is the only writer of its
// SQLite database, so a second implementation would risk drifting
// from the first one's idempotency and matching rules.
//
// Idempotency is checked per domain against fileHash, not once for
// the whole file: a domain skipped on an earlier run because its
// target did not exist yet must still be imported once the target
// exists, even though this exact file was already seen (habitat run
// 02b — the earlier file-level check made that second run a silent
// no-op).
func ImportNetnlDomains(ctx context.Context, st *store.Store, logger *slog.Logger, fileHash string, domains []NetnlDomain) (imported, skippedUnknown, skippedAlready int, err error) {
	byDomain := map[string][]NetnlDomain{}
	var order []string
	for _, d := range domains {
		if _, ok := byDomain[d.Domain]; !ok {
			order = append(order, d.Domain)
		}
		byDomain[d.Domain] = append(byDomain[d.Domain], d)
	}

	for _, domain := range order {
		already, err := st.NetnlImportRecorded(ctx, fileHash, domain)
		if err != nil {
			return imported, skippedUnknown, skippedAlready, fmt.Errorf("check import record for %q: %w", domain, err)
		}
		if already {
			skippedAlready++
			continue
		}

		target, err := st.GetTargetByDomain(ctx, domain)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				logger.Warn("import.internetnl.unknown_domain", "domain", domain)
				skippedUnknown++
				continue
			}
			return imported, skippedUnknown, skippedAlready, fmt.Errorf("lookup target %q: %w", domain, err)
		}

		scan, err := st.CreateScan(ctx, target.ID)
		if err != nil {
			return imported, skippedUnknown, skippedAlready, fmt.Errorf("create scan for %q: %w", domain, err)
		}
		var findings []models.Finding
		for _, d := range byDomain[domain] {
			findings = append(findings, NetnlFindings(d)...)
		}
		if err := st.AppendFindings(ctx, scan.ID, findings); err != nil {
			return imported, skippedUnknown, skippedAlready, fmt.Errorf("persist findings for %q: %w", domain, err)
		}
		if err := st.FinishScan(ctx, scan.ID, models.ScanStatusComplete, ""); err != nil {
			return imported, skippedUnknown, skippedAlready, fmt.Errorf("finish scan for %q: %w", domain, err)
		}
		if err := st.RecordNetnlImport(ctx, fileHash, domain); err != nil {
			return imported, skippedUnknown, skippedAlready, fmt.Errorf("record import for %q: %w", domain, err)
		}
		imported++
	}
	return imported, skippedUnknown, skippedAlready, nil
}
