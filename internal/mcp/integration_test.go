package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/MWest2020/wanderer/internal/mcp"
	"github.com/MWest2020/wanderer/internal/probe"
	"github.com/MWest2020/wanderer/internal/scanner"
	"github.com/MWest2020/wanderer/internal/store"
	"github.com/MWest2020/wanderer/pkg/models"
)

// stubProbe always emits one info finding so an MCP scan exercises
// the full pipeline without external network.
type stubProbe struct{}

func (stubProbe) ID() string { return "stub" }
func (stubProbe) Run(_ context.Context, t models.Target, _ probe.Config) ([]models.Finding, error) {
	return []models.Finding{{
		ProbeID: "stub.hello", Subject: t.Domain, Severity: models.SeverityInfo,
		Attributes: map[string]any{"ok": true},
	}}, nil
}

func wireServer(t *testing.T) (*mcp.Server, *store.Store) {
	t.Helper()
	st, err := store.Open(context.Background(), "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	sc := scanner.New(st, []probe.Probe{stubProbe{}}, probe.Config{})
	deps := mcp.Deps{Store: st, Scanner: sc}
	static, patterns := mcp.BuildResources(deps)
	return &mcp.Server{
		Tools:    mcp.BuildTools(deps),
		Static:   static,
		Patterns: patterns,
	}, st
}

func runMessages(t *testing.T, srv *mcp.Server, msgs ...string) []map[string]any {
	t.Helper()
	in := strings.Join(msgs, "\n") + "\n"
	var out bytes.Buffer
	if err := srv.Run(context.Background(), strings.NewReader(in), &out); err != nil {
		t.Fatalf("run: %v", err)
	}
	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	parsed := make([]map[string]any, 0, len(lines))
	for _, l := range lines {
		if l == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(l), &m); err != nil {
			t.Fatalf("decode %q: %v", l, err)
		}
		parsed = append(parsed, m)
	}
	return parsed
}

func TestMCP_ToolsListIncludesCoreSet(t *testing.T) {
	srv, _ := wireServer(t)
	resps := runMessages(t, srv, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	if len(resps) != 1 {
		t.Fatalf("want 1 reply, got %d", len(resps))
	}
	tools := resps[0]["result"].(map[string]any)["tools"].([]any)
	names := map[string]bool{}
	for _, t := range tools {
		names[t.(map[string]any)["name"].(string)] = true
	}
	for _, want := range []string{"scan_domain", "get_scan", "list_scans", "assess_scan", "get_assessment"} {
		if !names[want] {
			t.Errorf("missing tool: %s", want)
		}
	}
}

func TestMCP_NoConfigOrResetTools(t *testing.T) {
	srv, _ := wireServer(t)
	resps := runMessages(t, srv, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	tools := resps[0]["result"].(map[string]any)["tools"].([]any)
	for _, raw := range tools {
		name := raw.(map[string]any)["name"].(string)
		for _, banned := range []string{"config", "reset", "reload"} {
			if strings.Contains(name, banned) {
				t.Errorf("disallowed tool name surfaced: %s", name)
			}
		}
	}
}

func TestMCP_ScanDomainInvalid(t *testing.T) {
	srv, _ := wireServer(t)
	resps := runMessages(
		t, srv,
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"scan_domain","arguments":{"domain":""}}}`,
	)
	res := resps[0]["result"].(map[string]any)
	if res["isError"] != true {
		t.Errorf("expected isError=true on empty domain; got %v", res)
	}
}

func TestMCP_ResourceReadMissing(t *testing.T) {
	srv, _ := wireServer(t)
	resps := runMessages(
		t, srv,
		`{"jsonrpc":"2.0","id":1,"method":"resources/read","params":{"uri":"wanderer://scans/s_missing"}}`,
	)
	if _, hasErr := resps[0]["error"].(map[string]any); !hasErr {
		// Fallback: the read may have surfaced via the not-found path.
		if _, hasResult := resps[0]["result"]; hasResult {
			t.Errorf("expected error on missing scan; got %v", resps[0])
		}
	}
}

func TestMCP_ScanWrittenViaHTTPVisibleViaMCP(t *testing.T) {
	srv, st := wireServer(t)

	// Persist a scan directly through the store to mimic "written via
	// the HTTP API" without spinning up the API.
	tgt := &models.Target{Domain: "example.nl"}
	if err := st.UpsertTarget(context.Background(), tgt); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	sc, err := st.CreateScan(context.Background(), tgt.ID)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	resps := runMessages(
		t, srv,
		`{"jsonrpc":"2.0","id":1,"method":"resources/read","params":{"uri":"wanderer://scans/`+sc.ID+`"}}`,
	)
	res := resps[0]["result"].(map[string]any)
	contents := res["contents"].([]any)
	if len(contents) == 0 {
		t.Fatalf("no contents")
	}
	body := contents[0].(map[string]any)["text"].(string)
	if !strings.Contains(body, sc.ID) {
		t.Errorf("resource body missing scan id: %s", body)
	}
}
