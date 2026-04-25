package export

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"

	"github.com/MWest2020/wanderer/internal/store"
)

// AssessmentsCSVHeader names the columns emitted by WriteAssessmentsCSV.
// One row is written per (assessment × dimension) pair; the rationale
// list is not flattened — consumers who need it use JSONL.
var AssessmentsCSVHeader = []string{
	"id", "scan_id", "framework", "created_at",
	"dimension", "score", "completeness",
}

// WriteAssessmentsCSV streams one row per dimension per assessment.
func WriteAssessmentsCSV(ctx context.Context, w io.Writer, st *store.Store, sel store.Selectors) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()
	if err := cw.Write(AssessmentsCSVHeader); err != nil {
		return fmt.Errorf("export: write header: %w", err)
	}
	list, err := st.ListAssessments(ctx, sel)
	if err != nil {
		return err
	}
	for _, a := range list {
		for _, d := range a.Dimensions {
			if sel.Dimension != "" && string(d.Dimension) != sel.Dimension {
				continue
			}
			if err := cw.Write([]string{
				a.ID, a.ScanID, a.Framework, formatTime(a.CreatedAt),
				string(d.Dimension), string(d.Score), string(d.Completeness),
			}); err != nil {
				return fmt.Errorf("export: write row: %w", err)
			}
		}
	}
	return nil
}
