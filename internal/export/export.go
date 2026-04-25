// Package export writes findings, scans, and assessments from the
// store to CSV or JSONL. All writers stream row-by-row so large
// exports stay memory-bounded. Column order in CSV and field order in
// JSONL are deterministic: two runs on the same data produce byte-
// identical output.
package export

import "time"

// formatTime returns the RFC 3339 UTC representation of t, or an
// empty string for the zero value. Exporters use this for every
// timestamp so readers can rely on one format.
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
