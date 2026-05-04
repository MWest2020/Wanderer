package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/MWest2020/wanderer/pkg/models"
)

// Selectors narrow which records an export pulls out of the store.
// Zero-valued fields are treated as "no filter".
type Selectors struct {
	ScanID         string
	ProbePref      string // matched as "probe_id LIKE prefix.%", trivially also matches probe_id == prefix
	Dimension      string
	OrganisationID string // limits ListScans to scans whose Target belongs to this org
	Since          time.Time
	Until          time.Time
}

// whereAndArgs turns a Selectors into a SQL fragment and the matching
// argument slice. An empty fragment (no filters) returns an empty
// string.
func (s Selectors) whereAndArgs(idColumn, probeColumn, dimensionColumn, createdColumn string) (string, []any) {
	var clauses []string
	var args []any
	if s.ScanID != "" {
		clauses = append(clauses, fmt.Sprintf("%s = ?", idColumn))
		args = append(args, s.ScanID)
	}
	if s.ProbePref != "" && probeColumn != "" {
		clauses = append(clauses, fmt.Sprintf("(%s = ? OR %s LIKE ?)", probeColumn, probeColumn))
		args = append(args, s.ProbePref, s.ProbePref+".%")
	}
	if s.Dimension != "" && dimensionColumn != "" {
		clauses = append(clauses, fmt.Sprintf("%s = ?", dimensionColumn))
		args = append(args, s.Dimension)
	}
	if !s.Since.IsZero() && createdColumn != "" {
		clauses = append(clauses, fmt.Sprintf("%s >= ?", createdColumn))
		args = append(args, s.Since.UTC())
	}
	if !s.Until.IsZero() && createdColumn != "" {
		clauses = append(clauses, fmt.Sprintf("%s <= ?", createdColumn))
		args = append(args, s.Until.UTC())
	}
	if len(clauses) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

// FindingsIter streams Findings from the store. Callers must Close.
// The iterator yields records ordered by created_at then id so that
// two runs produce the same sequence — the exporters rely on this.
type FindingsIter struct {
	rows    *sql.Rows
	scanErr error
}

// ListFindings opens an iterator over findings matching sel.
func (s *Store) ListFindings(ctx context.Context, sel Selectors) (*FindingsIter, error) {
	where, args := sel.whereAndArgs("scan_id", "probe_id", "dimension_hint", "created_at")
	q := `SELECT id, scan_id, probe_id, COALESCE(dimension_hint,''), COALESCE(criterium_hint,''),
	             subject, severity, attributes, evidence, created_at
	      FROM findings` + where + ` ORDER BY created_at, id`
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("store: list findings: %w", err)
	}
	return &FindingsIter{rows: rows}, nil
}

// Next advances to the next row. Returns false when the iterator is
// exhausted or on error — check Err afterwards.
func (it *FindingsIter) Next() bool { return it.rows.Next() }

// Finding returns the current row as a models.Finding.
func (it *FindingsIter) Finding() (models.Finding, error) {
	var f models.Finding
	var dim, crit, sev, attrs string
	if err := it.rows.Scan(&f.ID, &f.ScanID, &f.ProbeID, &dim, &crit, &f.Subject, &sev, &attrs, &f.Evidence, &f.CreatedAt); err != nil {
		it.scanErr = err
		return f, err
	}
	f.DimensionHint = models.DimensionHint(dim)
	f.CriteriumHint = crit
	f.Severity = models.Severity(sev)
	if err := json.Unmarshal([]byte(attrs), &f.Attributes); err != nil {
		it.scanErr = err
		return f, err
	}
	return f, nil
}

// Err returns the first error encountered by Scan or Next.
func (it *FindingsIter) Err() error {
	if it.scanErr != nil {
		return it.scanErr
	}
	return it.rows.Err()
}

// Close releases the underlying *sql.Rows.
func (it *FindingsIter) Close() error { return it.rows.Close() }

// ScanRow is the flat row shape the scans CSV exporter produces. It
// joins scans to targets so the domain is available without a second
// query per row.
type ScanRow struct {
	ID            string
	TargetID      string
	Domain        string
	StartedAt     time.Time
	EndedAt       sql.NullTime
	Status        string
	Error         string
	FindingCount  int
}

// ListScans streams a flat row per scan matching sel.
func (s *Store) ListScans(ctx context.Context, sel Selectors) ([]ScanRow, error) {
	where, args := sel.whereAndArgs("scans.id", "", "", "scans.started_at")
	if sel.OrganisationID != "" {
		// Append the organisation filter manually so existing whereAndArgs
		// callers don't have to know about the targets join.
		if where == "" {
			where = " WHERE targets.organisation_id = ?"
		} else {
			where += " AND targets.organisation_id = ?"
		}
		args = append(args, sel.OrganisationID)
	}
	q := `SELECT scans.id, scans.target_id, targets.domain, scans.started_at, scans.ended_at, scans.status, COALESCE(scans.error,''),
	             (SELECT COUNT(*) FROM findings WHERE findings.scan_id = scans.id)
	      FROM scans LEFT JOIN targets ON targets.id = scans.target_id` + where + ` ORDER BY scans.started_at, scans.id`
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("store: list scans: %w", err)
	}
	defer rows.Close()
	var out []ScanRow
	for rows.Next() {
		var r ScanRow
		if err := rows.Scan(&r.ID, &r.TargetID, &r.Domain, &r.StartedAt, &r.EndedAt, &r.Status, &r.Error, &r.FindingCount); err != nil {
			return nil, fmt.Errorf("store: scan scans row: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ListAssessments returns Assessments matching sel, newest first.
// The selector's ProbePref is not applicable for assessments and is
// ignored; Dimension filters on the emitted dimension rows, not on
// the assessment itself, so it is also ignored here.
func (s *Store) ListAssessments(ctx context.Context, sel Selectors) ([]models.Assessment, error) {
	where, args := sel.whereAndArgs("scan_id", "", "", "created_at")
	q := `SELECT id, scan_id, framework, dimensions, report, created_at
	      FROM assessments` + where + ` ORDER BY created_at, id`
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("store: list assessments: %w", err)
	}
	defer rows.Close()
	var out []models.Assessment
	for rows.Next() {
		a, err := scanAssessment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}
