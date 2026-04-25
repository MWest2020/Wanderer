package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func runOnce(t *testing.T, srv *Server, in string) string {
	t.Helper()
	var out bytes.Buffer
	if err := srv.Run(context.Background(), strings.NewReader(in), &out); err != nil {
		t.Fatalf("run: %v", err)
	}
	return out.String()
}

func decodeOne(t *testing.T, line string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(line), &m); err != nil {
		t.Fatalf("decode %q: %v", line, err)
	}
	return m
}

func TestInitialize(t *testing.T) {
	srv := &Server{}
	out := runOnce(t, srv, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`+"\n")
	got := decodeOne(t, strings.TrimSpace(out))
	if got["jsonrpc"] != "2.0" {
		t.Errorf("jsonrpc = %v", got["jsonrpc"])
	}
	res, ok := got["result"].(map[string]any)
	if !ok {
		t.Fatalf("missing result: %v", got)
	}
	if res["protocolVersion"] == "" {
		t.Errorf("protocolVersion empty")
	}
}

func TestParseError(t *testing.T) {
	srv := &Server{}
	out := runOnce(t, srv, "this is not json\n")
	got := decodeOne(t, strings.TrimSpace(out))
	rerr, ok := got["error"].(map[string]any)
	if !ok {
		t.Fatalf("missing error envelope: %v", got)
	}
	if rerr["code"].(float64) != codeParseError {
		t.Errorf("code = %v, want %d", rerr["code"], codeParseError)
	}
}

func TestParseErrorThenValidRequestStaysAlive(t *testing.T) {
	srv := &Server{}
	in := "garbage\n" +
		`{"jsonrpc":"2.0","id":2,"method":"initialize"}` + "\n"
	out := runOnce(t, srv, in)
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 reply lines, got %d:\n%s", len(lines), out)
	}
	first := decodeOne(t, lines[0])
	second := decodeOne(t, lines[1])
	if _, hasErr := first["error"]; !hasErr {
		t.Errorf("first reply should be error: %v", first)
	}
	if _, hasResult := second["result"]; !hasResult {
		t.Errorf("second reply should be initialize result: %v", second)
	}
}

func TestUnknownMethod(t *testing.T) {
	srv := &Server{}
	out := runOnce(t, srv, `{"jsonrpc":"2.0","id":7,"method":"frobnicate"}`+"\n")
	got := decodeOne(t, strings.TrimSpace(out))
	if rerr, _ := got["error"].(map[string]any); rerr == nil || rerr["code"].(float64) != codeMethodNotFound {
		t.Errorf("expected method-not-found error, got %v", got)
	}
}

func TestToolsListAndCall(t *testing.T) {
	srv := &Server{
		Tools: []Tool{{
			Name:        "echo",
			Description: "Echo back its input.",
			InputSchema: map[string]any{"type": "object"},
			Handler: func(ctx context.Context, params json.RawMessage) (ToolResult, error) {
				return ToolResult{Content: []ContentBlock{{Type: "text", Text: string(params)}}}, nil
			},
		}},
	}
	in := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}` + "\n" +
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"echo","arguments":{"hello":"world"}}}` + "\n"
	out := runOnce(t, srv, in)
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 lines, got %d", len(lines))
	}
	listResp := decodeOne(t, lines[0])
	tools := listResp["result"].(map[string]any)["tools"].([]any)
	if len(tools) != 1 {
		t.Fatalf("want 1 tool, got %d", len(tools))
	}
	callResp := decodeOne(t, lines[1])
	res := callResp["result"].(map[string]any)
	content := res["content"].([]any)
	if len(content) != 1 {
		t.Fatalf("want 1 content block, got %d", len(content))
	}
	text := content[0].(map[string]any)["text"].(string)
	if !strings.Contains(text, `"hello":"world"`) {
		t.Errorf("echo lost params: %q", text)
	}
}

func TestNotificationHasNoReply(t *testing.T) {
	srv := &Server{}
	out := runOnce(t, srv, `{"jsonrpc":"2.0","method":"notifications/initialized"}`+"\n")
	if strings.TrimSpace(out) != "" {
		t.Errorf("notification produced reply: %q", out)
	}
}

func TestToolErrorReturnedAsContent(t *testing.T) {
	srv := &Server{
		Tools: []Tool{{
			Name: "fail",
			Handler: func(ctx context.Context, _ json.RawMessage) (ToolResult, error) {
				return ToolResult{}, errFail
			},
		}},
	}
	out := runOnce(t, srv, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"fail"}}`+"\n")
	got := decodeOne(t, strings.TrimSpace(out))
	res := got["result"].(map[string]any)
	if res["isError"] != true {
		t.Errorf("isError = %v, want true", res["isError"])
	}
}

var errFail = errInline("rule failed")

type errInline string

func (e errInline) Error() string { return string(e) }
