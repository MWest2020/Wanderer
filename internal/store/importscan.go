package store

import "fmt"

// importScanExistsFmt is the store's single SQL definition of "this
// scan is an import scan" (habitat run 02: an import written by
// `wanderer import internetnl` or `POST /imports/internetnl` carries
// only source_modus='import' Findings and no assessment; a perimeter
// scan never carries an import Finding). Checking for at least one
// import Finding is therefore a sufficient test and needs no schema
// change. Every selection site that must skip import scans when
// picking a target's "latest"/"previous" scan goes through this —
// either via ScanRow.IsImport (ListScans) or inline in
// PreviousScanForTarget's WHERE clause — rather than re-deriving the
// rule itself.
//
// scanIDExpr must be a trusted, internal SQL expression (a column
// reference such as "scans.id"), never request input — it is
// interpolated directly, not bound as a parameter.
const importScanExistsFmt = `EXISTS (SELECT 1 FROM findings WHERE findings.scan_id = %s AND findings.source_modus = 'import')`

// importScanExistsSQL renders importScanExistsFmt for scanIDExpr.
func importScanExistsSQL(scanIDExpr string) string {
	return fmt.Sprintf(importScanExistsFmt, scanIDExpr)
}
