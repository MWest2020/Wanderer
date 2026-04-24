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

// CreateAssessment persists an Assessment. It assigns a new ID and the
// current UTC timestamp; callers that want to supply their own ID (for
// test determinism) may set Assessment.ID before calling.
func (s *Store) CreateAssessment(ctx context.Context, a *models.Assessment) error {
	if a == nil {
		return errors.New("store: nil assessment")
	}
	if a.ID == "" {
		a.ID = newID("a")
	}
	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now().UTC()
	}
	if err := a.Validate(); err != nil {
		return err
	}
	dims, err := json.Marshal(a.Dimensions)
	if err != nil {
		return fmt.Errorf("store: marshal dimensions: %w", err)
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO assessments (id, scan_id, framework, dimensions, report, created_at)
		 VALUES (?,?,?,?,?,?)`,
		a.ID, a.ScanID, a.Framework, string(dims), a.Report, a.CreatedAt)
	if err != nil {
		return fmt.Errorf("store: insert assessment: %w", err)
	}
	return nil
}

// GetAssessment returns an Assessment by its ID.
func (s *Store) GetAssessment(ctx context.Context, id string) (*models.Assessment, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, scan_id, framework, dimensions, report, created_at
		 FROM assessments WHERE id = ?`, id)
	return scanAssessment(row)
}

// ListAssessmentsForScan returns every Assessment persisted for a given
// scan, most recent first.
func (s *Store) ListAssessmentsForScan(ctx context.Context, scanID string) ([]models.Assessment, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, scan_id, framework, dimensions, report, created_at
		 FROM assessments WHERE scan_id = ? ORDER BY created_at DESC`, scanID)
	if err != nil {
		return nil, fmt.Errorf("store: select assessments: %w", err)
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
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: rows: %w", err)
	}
	return out, nil
}

// rowScanner abstracts over *sql.Row and *sql.Rows so scanAssessment
// can serve both Get and List.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanAssessment(r rowScanner) (*models.Assessment, error) {
	var a models.Assessment
	var dims string
	err := r.Scan(&a.ID, &a.ScanID, &a.Framework, &dims, &a.Report, &a.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("store: scan assessment: %w", err)
	}
	if err := json.Unmarshal([]byte(dims), &a.Dimensions); err != nil {
		return nil, fmt.Errorf("store: unmarshal dimensions: %w", err)
	}
	return &a, nil
}
