package store

import (
	"context"
	"testing"
	"time"

	"github.com/MWest2020/wanderer/pkg/models"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	st, err := Open(context.Background(), "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func seedScan(t *testing.T, st *Store) string {
	t.Helper()
	ctx := context.Background()
	tgt := &models.Target{Domain: "example.nl"}
	if err := st.UpsertTarget(ctx, tgt); err != nil {
		t.Fatalf("upsert target: %v", err)
	}
	sc, err := st.CreateScan(ctx, tgt.ID)
	if err != nil {
		t.Fatalf("create scan: %v", err)
	}
	return sc.ID
}

func TestCreateAssessment_RoundTrip(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()
	scanID := seedScan(t, st)

	a := &models.Assessment{
		ScanID:    scanID,
		Framework: "wand",
		Dimensions: []models.DimensionScore{
			{
				Dimension:    models.DimensionJuridisch,
				Score:        models.ScoreVoldoende,
				Completeness: models.CompletenessComplete,
				Rationale: []models.Rationale{
					{CriteriumID: "wand.j.1", Verdict: "ok", Score: models.ScoreVoldoende, Evidence: []string{"f1"}},
				},
			},
		},
		Report: "# fake report",
	}
	if err := st.CreateAssessment(ctx, a); err != nil {
		t.Fatalf("create: %v", err)
	}
	if a.ID == "" {
		t.Fatalf("expected ID assigned")
	}
	if a.CreatedAt.IsZero() {
		t.Fatalf("expected CreatedAt set")
	}

	got, err := st.GetAssessment(ctx, a.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ScanID != scanID || got.Framework != "wand" || len(got.Dimensions) != 1 {
		t.Errorf("round trip lost fields: %+v", got)
	}
	if got.Dimensions[0].Rationale[0].Evidence[0] != "f1" {
		t.Errorf("rationale evidence lost")
	}
	if got.Report != "# fake report" {
		t.Errorf("report lost: %q", got.Report)
	}
}

func TestListAssessmentsForScan(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()
	scanID := seedScan(t, st)

	mk := func(when time.Time) *models.Assessment {
		return &models.Assessment{
			ScanID:    scanID,
			Framework: "wand",
			CreatedAt: when,
			Dimensions: []models.DimensionScore{
				{Dimension: models.DimensionJuridisch, Score: models.ScoreVoldoende, Completeness: models.CompletenessComplete},
			},
		}
	}
	a1 := mk(time.Date(2026, 4, 24, 9, 0, 0, 0, time.UTC))
	a2 := mk(time.Date(2026, 4, 24, 11, 0, 0, 0, time.UTC))
	if err := st.CreateAssessment(ctx, a1); err != nil {
		t.Fatalf("create a1: %v", err)
	}
	if err := st.CreateAssessment(ctx, a2); err != nil {
		t.Fatalf("create a2: %v", err)
	}

	list, err := st.ListAssessmentsForScan(ctx, scanID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("want 2, got %d", len(list))
	}
	if list[0].ID != a2.ID {
		t.Errorf("want newest first; got %s then %s", list[0].ID, list[1].ID)
	}
}

func TestGetAssessment_NotFound(t *testing.T) {
	st := openTestStore(t)
	_, err := st.GetAssessment(context.Background(), "does_not_exist")
	if err == nil {
		t.Fatalf("expected error")
	}
	if err != ErrNotFound {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}
