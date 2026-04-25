package export

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"

	"github.com/MWest2020/wanderer/internal/store"
)

// FindingsCSVHeader is exported so CLI help and tests can reference it.
var FindingsCSVHeader = []string{
	"id", "scan_id", "probe_id", "dimension_hint", "criterium_hint",
	"subject", "severity", "created_at", "attributes_json",
}

// WriteFindingsCSV streams findings matching sel to w as CSV. The
// header is always written, even for an empty result set.
func WriteFindingsCSV(ctx context.Context, w io.Writer, st *store.Store, sel store.Selectors) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()

	if err := cw.Write(FindingsCSVHeader); err != nil {
		return fmt.Errorf("export: write header: %w", err)
	}

	it, err := st.ListFindings(ctx, sel)
	if err != nil {
		return err
	}
	defer it.Close()

	for it.Next() {
		f, err := it.Finding()
		if err != nil {
			return fmt.Errorf("export: scan finding: %w", err)
		}
		attrs, err := json.Marshal(f.Attributes)
		if err != nil {
			return fmt.Errorf("export: marshal attrs: %w", err)
		}
		row := []string{
			f.ID,
			f.ScanID,
			f.ProbeID,
			string(f.DimensionHint),
			f.CriteriumHint,
			f.Subject,
			string(f.Severity),
			formatTime(f.CreatedAt),
			string(attrs),
		}
		if err := cw.Write(row); err != nil {
			return fmt.Errorf("export: write row: %w", err)
		}
	}
	if err := it.Err(); err != nil {
		return err
	}
	return nil
}
