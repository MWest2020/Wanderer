package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/MWest2020/wanderer/internal/assessor"
	"github.com/MWest2020/wanderer/internal/assessor/dictu"
	"github.com/MWest2020/wanderer/internal/scanner"
	"github.com/MWest2020/wanderer/internal/store"
	"github.com/MWest2020/wanderer/pkg/models"
)

// Deps groups the runtime objects the tool handlers depend on.
type Deps struct {
	Store   *store.Store
	Scanner *scanner.Scanner
}

// BuildTools returns the standard Wanderer MCP tool set.
func BuildTools(d Deps) []Tool {
	return []Tool{
		scanDomainTool(d),
		getScanTool(d),
		listScansTool(d),
		assessScanTool(d),
		getAssessmentTool(d),
	}
}

func scanDomainTool(d Deps) Tool {
	return Tool{
		Name:        "scan_domain",
		Description: "Run a Wanderer scan against a domain and return the resulting Scan with findings.",
		InputSchema: SchemaScanDomain,
		Handler: func(ctx context.Context, params json.RawMessage) (ToolResult, error) {
			var p struct {
				Domain  string   `json:"domain"`
				Related []string `json:"related"`
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return ToolResult{}, fmt.Errorf("invalid params: %w", err)
			}
			if p.Domain == "" {
				return ToolResult{}, errors.New("domain is required")
			}
			scan, err := d.Scanner.Scan(ctx, models.Target{Domain: p.Domain, Related: p.Related})
			if err != nil {
				return ToolResult{}, err
			}
			return jsonContent(scan)
		},
	}
}

func getScanTool(d Deps) Tool {
	return Tool{
		Name:        "get_scan",
		Description: "Return a stored Scan and its findings by ID.",
		InputSchema: SchemaIDOnly,
		Handler: func(ctx context.Context, params json.RawMessage) (ToolResult, error) {
			var p struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return ToolResult{}, fmt.Errorf("invalid params: %w", err)
			}
			if p.ID == "" {
				return ToolResult{}, errors.New("id is required")
			}
			scan, err := d.Store.GetScan(ctx, p.ID)
			if err != nil {
				return ToolResult{}, err
			}
			return jsonContent(scan)
		},
	}
}

func listScansTool(d Deps) Tool {
	return Tool{
		Name:        "list_scans",
		Description: "List recent scans with summary fields (id, domain, status, started_at, finding_count).",
		InputSchema: SchemaListScans,
		Handler: func(ctx context.Context, params json.RawMessage) (ToolResult, error) {
			rows, err := d.Store.ListScans(ctx, store.Selectors{})
			if err != nil {
				return ToolResult{}, err
			}
			// Truncate by --limit if supplied. Default cap is 100.
			limit := 100
			if len(params) > 0 {
				var p struct {
					Limit int `json:"limit"`
				}
				_ = json.Unmarshal(params, &p)
				if p.Limit > 0 {
					limit = p.Limit
				}
			}
			// Newest first.
			out := rows
			if len(out) > limit {
				out = out[:limit]
			}
			return jsonContent(map[string]any{"scans": out})
		},
	}
}

func assessScanTool(d Deps) Tool {
	return Tool{
		Name:        "assess_scan",
		Description: "Score a stored scan against the DICTU rule set and return the persisted Assessment.",
		InputSchema: SchemaAssessScan,
		Handler: func(ctx context.Context, params json.RawMessage) (ToolResult, error) {
			var p struct {
				ScanID string `json:"scan_id"`
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return ToolResult{}, fmt.Errorf("invalid params: %w", err)
			}
			if p.ScanID == "" {
				return ToolResult{}, errors.New("scan_id is required")
			}
			scan, err := d.Store.GetScan(ctx, p.ScanID)
			if err != nil {
				return ToolResult{}, err
			}
			rules := dictu.DefaultRules()
			a := &models.Assessment{
				ScanID:     scan.ID,
				Framework:  "dictu",
				Dimensions: assessor.Assess(scan.Findings, rules),
			}
			if err := d.Store.CreateAssessment(ctx, a); err != nil {
				return ToolResult{}, err
			}
			return jsonContent(a)
		},
	}
}

func getAssessmentTool(d Deps) Tool {
	return Tool{
		Name:        "get_assessment",
		Description: "Return a persisted Assessment by ID.",
		InputSchema: SchemaIDOnly,
		Handler: func(ctx context.Context, params json.RawMessage) (ToolResult, error) {
			var p struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return ToolResult{}, fmt.Errorf("invalid params: %w", err)
			}
			if p.ID == "" {
				return ToolResult{}, errors.New("id is required")
			}
			a, err := d.Store.GetAssessment(ctx, p.ID)
			if err != nil {
				return ToolResult{}, err
			}
			return jsonContent(a)
		},
	}
}

// jsonContent JSON-encodes v and wraps it in a single text content
// block — the MCP tool result shape.
func jsonContent(v any) (ToolResult, error) {
	buf, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return ToolResult{}, err
	}
	return ToolResult{Content: []ContentBlock{{Type: "text", Text: string(buf)}}}, nil
}
