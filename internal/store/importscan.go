package store

// importScanExistsSQL is the store's single SQL definition of "this
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
// It correlates on `scans.id`, so it only fits a query over the
// scans table. A constant, not a format string: the query text stays
// fixed at compile time.
const importScanExistsSQL = `EXISTS (SELECT 1 FROM findings WHERE findings.scan_id = scans.id AND findings.source_modus = 'import')`
