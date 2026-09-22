package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/MWest2020/wanderer/pkg/models"
)

// GetTargetByDomain returns the Target matching domain exactly, or
// ErrNotFound if no target is registered under that domain.
// targets.domain is globally unique (see AddFleetDomain's doc
// comment), so unlike UpsertTarget this never creates a row — it is
// the "match against existing targets, never invent one" lookup the
// netnl importer needs (spec.md "Unknown domains SHALL be logged at
// WARN and skipped").
func (s *Store) GetTargetByDomain(ctx context.Context, domain string) (*models.Target, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id FROM targets WHERE domain = ?`, domain)
	var id string
	if err := row.Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("store: lookup target by domain: %w", err)
	}
	return s.GetTarget(ctx, id)
}

// NetnlImportRecorded reports whether domain has already been
// imported from the netnl-findings file with this exact sha256 hash,
// so the caller can skip re-persisting it (spec.md "Re-importing an
// identical file SHALL be idempotent"). The key is (fileHash, domain)
// rather than fileHash alone: a domain skipped on a previous run
// because its target did not exist yet is not "done" for this file —
// it must still be imported once the target shows up, even though the
// file itself was already seen (habitat run 02b).
func (s *Store) NetnlImportRecorded(ctx context.Context, fileHash, domain string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM netnl_imports WHERE file_hash = ? AND domain = ?)`, fileHash, domain).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("store: check netnl import: %w", err)
	}
	return exists, nil
}

// RecordNetnlImport marks domain as imported from fileHash. Call this
// once that domain's findings have been persisted, so a later re-run
// of the same file recognises this domain as a no-op without
// affecting any other domain in the file.
func (s *Store) RecordNetnlImport(ctx context.Context, fileHash, domain string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO netnl_imports (file_hash, domain, imported_at) VALUES (?, ?, ?)`,
		fileHash, domain, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("store: record netnl import: %w", err)
	}
	return nil
}

// LatestScanByModus returns the most recently started scan for
// targetID that carries at least one Finding with the given
// SourceModus, or ErrNotFound if none exists. An import scan and a
// perimeter scan for the same target coexist independently (neither
// overwrites the other — spec.md "Imports and perimeter scans do not
// erase each other"); this is how a consumer reads each kind's
// newest scan on its own timeline instead of the target's overall
// newest scan, which might be of the other kind.
func (s *Store) LatestScanByModus(ctx context.Context, targetID string, modus models.SourceModus) (*models.Scan, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT s.id FROM scans s
		 JOIN findings f ON f.scan_id = s.id
		 WHERE s.target_id = ? AND f.source_modus = ?
		 ORDER BY s.started_at DESC, s.id DESC
		 LIMIT 1`,
		targetID, string(modus))
	var id string
	if err := row.Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("store: latest scan by modus: %w", err)
	}
	return s.GetScan(ctx, id)
}

// LatestImportFindings returns targetID's import-modus Findings,
// correlated across scan kinds the way the standards assessor needs
// (design.md "Import semantics"): web and mail arrive in separate
// netnl-findings files and therefore separate import scans (a single
// file can still carry both for one domain, in which case both share
// a scan — see importNetnlDomains), so a per-scan read would show
// only whichever type's scan happens to be newest, or nothing at all
// when the scan under assessment is a perimeter scan.
//
// Findings are grouped by their "type" attribute ("web"/"mail", set
// by scanner.NetnlFindings) rather than by scan: for each type, only
// the Findings from that type's most-recently-started scan are kept,
// so a fresh mail import supersedes the previous mail Findings
// without touching web, and vice versa — never a whole-scan
// replacement. A Finding without a "type" attribute (not a netnl
// import) is grouped under the empty string, on the same rule.
func (s *Store) LatestImportFindings(ctx context.Context, targetID string) ([]models.Finding, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT f.id, f.scan_id, f.probe_id, COALESCE(f.dimension_hint,''), COALESCE(f.criterium_hint,''),
		        f.subject, f.severity, f.attributes, f.evidence, f.created_at
		 FROM findings f
		 JOIN scans s ON s.id = f.scan_id
		 WHERE s.target_id = ? AND f.source_modus = ?
		 ORDER BY s.started_at DESC, s.id DESC, f.rowid`,
		targetID, string(models.SourceModusImport))
	if err != nil {
		return nil, fmt.Errorf("store: latest import findings: %w", err)
	}
	defer rows.Close()

	winnerScanByType := map[string]string{}
	var out []models.Finding
	for rows.Next() {
		var f models.Finding
		var dim, crit, sev, attrs string
		if err := rows.Scan(&f.ID, &f.ScanID, &f.ProbeID, &dim, &crit, &f.Subject, &sev, &attrs, &f.Evidence, &f.CreatedAt); err != nil {
			return nil, fmt.Errorf("store: scan import finding: %w", err)
		}
		f.SourceModus = models.SourceModusImport
		f.DimensionHint = models.DimensionHint(dim)
		f.CriteriumHint = crit
		f.Severity = models.Severity(sev)
		if err := json.Unmarshal([]byte(attrs), &f.Attributes); err != nil {
			return nil, fmt.Errorf("store: decode import finding attributes: %w", err)
		}

		typ, _ := f.Attributes["type"].(string)
		winner, seen := winnerScanByType[typ]
		if !seen {
			winnerScanByType[typ] = f.ScanID
			winner = f.ScanID
		}
		if f.ScanID != winner {
			// Superseded: a newer scan already produced this type's
			// winning findings (rows arrive newest-scan-first).
			continue
		}
		out = append(out, f)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: latest import findings: %w", err)
	}
	return out, nil
}

// FindingsForAssessment returns the Findings the assessor should
// score scan against: scan's own non-import Findings, plus
// targetID's correlated import Findings (LatestImportFindings) —
// never scan's own import Findings directly, even if scan is itself
// an import scan. That indirection is what keeps a superseded import
// scan (an older mail import re-viewed after a newer one landed) from
// contributing stale Findings alongside the correlated set: only the
// winning scan per type ever reaches the assessor, whichever scan the
// caller asked about (design.md "Import semantics" — "a perimeter
// scan never erases imported findings and vice versa").
//
// Non-import Findings (perimeter, inventory, egress, drift) pass
// through unchanged, so every rule outside the `standards` dimension
// keeps reading exactly its own scan's evidence, as before.
func (s *Store) FindingsForAssessment(ctx context.Context, scan *models.Scan) ([]models.Finding, error) {
	imported, err := s.LatestImportFindings(ctx, scan.TargetID)
	if err != nil {
		return nil, err
	}
	out := make([]models.Finding, 0, len(scan.Findings)+len(imported))
	for _, f := range scan.Findings {
		if f.SourceModus == models.SourceModusImport {
			continue
		}
		out = append(out, f)
	}
	out = append(out, imported...)
	return out, nil
}
