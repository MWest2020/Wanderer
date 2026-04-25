package export

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"

	"github.com/MWest2020/wanderer/internal/store"
	"github.com/MWest2020/wanderer/pkg/models"
)

// findingJSONL mirrors models.Finding with a base64 string in place
// of the binary Evidence slice so JSONL consumers can read the field
// without double-decoding.
type findingJSONL struct {
	ID            string         `json:"id"`
	ScanID        string         `json:"scan_id"`
	ProbeID       string         `json:"probe_id"`
	DimensionHint string         `json:"dimension_hint,omitempty"`
	CriteriumHint string         `json:"criterium_hint,omitempty"`
	Subject       string         `json:"subject"`
	Severity      string         `json:"severity"`
	Attributes    map[string]any `json:"attributes"`
	Evidence      string         `json:"evidence,omitempty"`
	CreatedAt     string         `json:"created_at"`
}

// WriteFindingsJSONL streams findings as one JSON object per line.
// When includeEvidence is false the evidence field is dropped; when
// true it is base64-encoded.
func WriteFindingsJSONL(ctx context.Context, w io.Writer, st *store.Store, sel store.Selectors, includeEvidence bool) error {
	enc := json.NewEncoder(w)
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
		row := findingJSONL{
			ID:            f.ID,
			ScanID:        f.ScanID,
			ProbeID:       f.ProbeID,
			DimensionHint: string(f.DimensionHint),
			CriteriumHint: f.CriteriumHint,
			Subject:       f.Subject,
			Severity:      string(f.Severity),
			Attributes:    f.Attributes,
			CreatedAt:     formatTime(f.CreatedAt),
		}
		if includeEvidence && len(f.Evidence) > 0 {
			row.Evidence = base64.StdEncoding.EncodeToString(f.Evidence)
		}
		if err := enc.Encode(row); err != nil {
			return fmt.Errorf("export: encode: %w", err)
		}
	}
	return it.Err()
}

// WriteScansJSONL streams scans as one JSON object per line.
func WriteScansJSONL(ctx context.Context, w io.Writer, st *store.Store, sel store.Selectors) error {
	enc := json.NewEncoder(w)
	rows, err := st.ListScans(ctx, sel)
	if err != nil {
		return err
	}
	for _, r := range rows {
		ended := ""
		if r.EndedAt.Valid {
			ended = formatTime(r.EndedAt.Time)
		}
		row := map[string]any{
			"id":            r.ID,
			"target_id":     r.TargetID,
			"domain":        r.Domain,
			"started_at":    formatTime(r.StartedAt),
			"ended_at":      ended,
			"status":        r.Status,
			"error":         r.Error,
			"finding_count": r.FindingCount,
		}
		if err := enc.Encode(row); err != nil {
			return fmt.Errorf("export: encode: %w", err)
		}
	}
	return nil
}

// WriteAssessmentsJSONL streams one full Assessment object per line.
func WriteAssessmentsJSONL(ctx context.Context, w io.Writer, st *store.Store, sel store.Selectors) error {
	enc := json.NewEncoder(w)
	list, err := st.ListAssessments(ctx, sel)
	if err != nil {
		return err
	}
	for _, a := range list {
		// Copy to a local so we can suppress Report when the CLI ever
		// wants to do so; today JSONL always includes it.
		row := a
		if err := enc.Encode(row); err != nil {
			return fmt.Errorf("export: encode: %w", err)
		}
	}
	return nil
}

// ensureModelsImported is a linker-visible reference to keep the
// models import in use when build configurations vary; the findings
// JSONL path above already uses models.Finding indirectly via the
// store, but leaving this untouched is insurance against a future
// refactor that trims the findings path.
var _ = models.Finding{}
