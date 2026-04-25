package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/MWest2020/wanderer/internal/store"
)

// BuildResources returns the standard Wanderer MCP resource set:
// static directory listings plus URI patterns for individual records.
func BuildResources(d Deps) ([]Resource, []ResourcePattern) {
	static := []Resource{
		{
			URI:         "wanderer://scans",
			Name:        "Wanderer scans",
			Description: "Summary of every stored scan.",
			MimeType:    "application/json",
			Read: func(ctx context.Context) (string, error) {
				rows, err := d.Store.ListScans(ctx, store.Selectors{})
				if err != nil {
					return "", err
				}
				return jsonString(map[string]any{"scans": rows})
			},
		},
		{
			URI:         "wanderer://assessments",
			Name:        "Wanderer assessments",
			Description: "Every persisted Assessment.",
			MimeType:    "application/json",
			Read: func(ctx context.Context) (string, error) {
				list, err := d.Store.ListAssessments(ctx, store.Selectors{})
				if err != nil {
					return "", err
				}
				return jsonString(map[string]any{"assessments": list})
			},
		},
	}
	patterns := []ResourcePattern{
		{
			Match: func(uri string) bool { return strings.HasPrefix(uri, "wanderer://scans/") },
			Read: func(ctx context.Context, uri string) (string, error) {
				id := strings.TrimPrefix(uri, "wanderer://scans/")
				findingsOnly := false
				if strings.HasSuffix(id, "/findings") {
					id = strings.TrimSuffix(id, "/findings")
					findingsOnly = true
				}
				if id == "" {
					return "", errors.New("empty scan id")
				}
				scan, err := d.Store.GetScan(ctx, id)
				if err != nil {
					return "", err
				}
				if findingsOnly {
					return jsonString(map[string]any{"scan_id": scan.ID, "findings": scan.Findings})
				}
				return jsonString(scan)
			},
		},
		{
			Match: func(uri string) bool { return strings.HasPrefix(uri, "wanderer://assessments/") },
			Read: func(ctx context.Context, uri string) (string, error) {
				id := strings.TrimPrefix(uri, "wanderer://assessments/")
				if id == "" {
					return "", errors.New("empty assessment id")
				}
				a, err := d.Store.GetAssessment(ctx, id)
				if err != nil {
					return "", err
				}
				return jsonString(a)
			},
		},
	}
	return static, patterns
}

func jsonString(v any) (string, error) {
	buf, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", fmt.Errorf("mcp: encode resource: %w", err)
	}
	return string(buf), nil
}
