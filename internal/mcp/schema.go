package mcp

// JSON Schema fragments for the MCP tools. Kept centralised so the
// shapes are reviewable in one place rather than scattered through
// the handlers in tools.go. Each map is the value of a tool's
// inputSchema field per the MCP tools/list response.

// SchemaScanDomain describes the input to scan_domain.
var SchemaScanDomain = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"domain":  map[string]any{"type": "string", "description": "Apex domain to scan"},
		"related": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
	},
	"required": []string{"domain"},
}

// SchemaIDOnly is the canonical {"id": string} input shape used by
// get_scan and get_assessment.
var SchemaIDOnly = map[string]any{
	"type":       "object",
	"properties": map[string]any{"id": map[string]any{"type": "string"}},
	"required":   []string{"id"},
}

// SchemaListScans describes the input to list_scans.
var SchemaListScans = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"limit": map[string]any{"type": "integer", "minimum": 1, "maximum": 200},
	},
}

// SchemaAssessScan describes the input to assess_scan.
var SchemaAssessScan = map[string]any{
	"type":       "object",
	"properties": map[string]any{"scan_id": map[string]any{"type": "string"}},
	"required":   []string{"scan_id"},
}
