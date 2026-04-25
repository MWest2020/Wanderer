package export

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"

	"github.com/MWest2020/wanderer/internal/store"
)

// ScansCSVHeader enumerates the scans CSV columns in the order they
// appear in every export.
var ScansCSVHeader = []string{
	"id", "target_id", "domain",
	"started_at", "ended_at",
	"status", "error", "finding_count",
}

// WriteScansCSV writes one row per scan matching sel.
func WriteScansCSV(ctx context.Context, w io.Writer, st *store.Store, sel store.Selectors) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()
	if err := cw.Write(ScansCSVHeader); err != nil {
		return fmt.Errorf("export: write header: %w", err)
	}
	rows, err := st.ListScans(ctx, sel)
	if err != nil {
		return err
	}
	for _, r := range rows {
		ended := ""
		if r.EndedAt.Valid {
			ended = formatTime(r.EndedAt.Time)
		}
		if err := cw.Write([]string{
			r.ID, r.TargetID, r.Domain,
			formatTime(r.StartedAt), ended,
			r.Status, r.Error, strconv.Itoa(r.FindingCount),
		}); err != nil {
			return fmt.Errorf("export: write row: %w", err)
		}
	}
	return nil
}
