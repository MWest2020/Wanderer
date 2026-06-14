// Package mcp implements a minimal Model Context Protocol server over
// stdio. The protocol is line-delimited JSON-RPC 2.0; messages are one
// JSON object per line on stdin/stdout. The server dispatches the small
// subset of methods Wanderer needs: initialize, tools/list, tools/call,
// resources/list, resources/read, ping. ADR-0005 records why this is
// hand-rolled rather than built on a third-party SDK.
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"
)

// JSON-RPC error codes per the spec.
const (
	codeParseError     = -32700
	codeInvalidRequest = -32600
	codeMethodNotFound = -32601
	codeInvalidParams  = -32602
	codeInternalError  = -32603
)

// ProtocolVersion is the MCP protocol version Wanderer advertises.
// Stable subset of the spec — initialize / tools / resources only.
const ProtocolVersion = "2024-11-05"

// rpcRequest is the JSON-RPC 2.0 request envelope. ID is intentionally
// json.RawMessage so notifications (no id) and id-bearing requests
// share one shape.
type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// rpcResponse is the JSON-RPC 2.0 response envelope.
type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Tool is a single MCP tool. Handlers receive raw JSON params and
// return a tool result; returning an error converts to an MCP
// tool-level error response (not a JSON-RPC transport error).
type Tool struct {
	Name        string
	Description string
	InputSchema map[string]any
	Handler     func(ctx context.Context, params json.RawMessage) (ToolResult, error)
}

// ToolResult is the body of a successful tools/call response. Content
// is a list of text blocks per the MCP spec.
type ToolResult struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

// ContentBlock is one chunk of a tool result. Wanderer tools always
// emit type=text with a JSON-stringified payload.
type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Resource is one named resource the server can serve.
type Resource struct {
	URI         string
	Name        string
	Description string
	MimeType    string
	Read        func(ctx context.Context) (string, error)
}

// ResourcePattern handles dynamic URIs (wanderer://scans/{id}).
type ResourcePattern struct {
	Match func(uri string) bool
	Read  func(ctx context.Context, uri string) (string, error)
}

// Server is the MCP dispatcher. Tools and ResourcePatterns are
// registered before Run.
type Server struct {
	Tools    []Tool
	Static   []Resource
	Patterns []ResourcePattern
	Logger   *slog.Logger
	writeMu  sync.Mutex
	enc      *json.Encoder
}

// Run consumes JSON-RPC messages from in until EOF, writing replies
// to out. Errors from individual handlers do not terminate the loop;
// only stdin closure does. Logs go to logger (typically wired to
// stderr by the caller).
func (s *Server) Run(ctx context.Context, in io.Reader, out io.Writer) error {
	s.enc = json.NewEncoder(out)
	if s.Logger == nil {
		s.Logger = slog.Default()
	}
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var req rpcRequest
		if err := json.Unmarshal(line, &req); err != nil {
			s.writeError(nil, codeParseError, "parse error")
			continue
		}
		s.dispatch(ctx, req)
	}
	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("mcp: read: %w", err)
	}
	return nil
}

func (s *Server) dispatch(ctx context.Context, req rpcRequest) {
	switch req.Method {
	case "initialize":
		s.handleInitialize(req)
	case "ping":
		s.writeResult(req.ID, map[string]any{})
	case "tools/list":
		s.handleToolsList(req)
	case "tools/call":
		s.handleToolsCall(ctx, req)
	case "resources/list":
		s.handleResourcesList(req)
	case "resources/read":
		s.handleResourcesRead(ctx, req)
	case "notifications/initialized", "notifications/cancelled":
		// Notifications carry no ID and need no reply.
	default:
		if len(req.ID) == 0 {
			return // notifications: ignore unknown methods silently
		}
		s.writeError(req.ID, codeMethodNotFound, "method not found: "+req.Method)
	}
}

func (s *Server) handleInitialize(req rpcRequest) {
	s.writeResult(req.ID, map[string]any{
		"protocolVersion": ProtocolVersion,
		"capabilities": map[string]any{
			"tools":     map[string]any{},
			"resources": map[string]any{},
		},
		"serverInfo": map[string]any{
			"name":    "wanderer",
			"version": "0.x",
		},
	})
}

func (s *Server) handleToolsList(req rpcRequest) {
	tools := make([]map[string]any, 0, len(s.Tools))
	for _, t := range s.Tools {
		tools = append(tools, map[string]any{
			"name":        t.Name,
			"description": t.Description,
			"inputSchema": t.InputSchema,
		})
	}
	s.writeResult(req.ID, map[string]any{"tools": tools})
}

func (s *Server) handleToolsCall(ctx context.Context, req rpcRequest) {
	var p struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &p); err != nil {
		s.writeError(req.ID, codeInvalidParams, "invalid params: "+err.Error())
		return
	}
	for _, t := range s.Tools {
		if t.Name != p.Name {
			continue
		}
		res, err := t.Handler(ctx, p.Arguments)
		if err != nil {
			// MCP tool-level error: still a successful JSON-RPC reply
			// with isError=true so clients can surface the message.
			s.writeResult(req.ID, ToolResult{
				IsError: true,
				Content: []ContentBlock{{Type: "text", Text: err.Error()}},
			})
			return
		}
		s.writeResult(req.ID, res)
		return
	}
	s.writeError(req.ID, codeMethodNotFound, "no such tool: "+p.Name)
}

func (s *Server) handleResourcesList(req rpcRequest) {
	resources := make([]map[string]any, 0, len(s.Static))
	for _, r := range s.Static {
		resources = append(resources, map[string]any{
			"uri":         r.URI,
			"name":        r.Name,
			"description": r.Description,
			"mimeType":    r.MimeType,
		})
	}
	s.writeResult(req.ID, map[string]any{"resources": resources})
}

func (s *Server) handleResourcesRead(ctx context.Context, req rpcRequest) {
	var p struct {
		URI string `json:"uri"`
	}
	if err := json.Unmarshal(req.Params, &p); err != nil {
		s.writeError(req.ID, codeInvalidParams, "invalid params: "+err.Error())
		return
	}
	for _, r := range s.Static {
		if r.URI == p.URI {
			body, err := r.Read(ctx)
			if err != nil {
				s.writeError(req.ID, codeInternalError, err.Error())
				return
			}
			s.writeResourceContents(req.ID, p.URI, r.MimeType, body)
			return
		}
	}
	for _, pat := range s.Patterns {
		if pat.Match(p.URI) {
			body, err := pat.Read(ctx, p.URI)
			if err != nil {
				s.writeError(req.ID, codeInternalError, err.Error())
				return
			}
			s.writeResourceContents(req.ID, p.URI, "application/json", body)
			return
		}
	}
	s.writeError(req.ID, codeInvalidParams, "resource not found: "+p.URI)
}

func (s *Server) writeResourceContents(id json.RawMessage, uri, mime, text string) {
	s.writeResult(id, map[string]any{
		"contents": []map[string]any{
			{"uri": uri, "mimeType": mime, "text": text},
		},
	})
}

func (s *Server) writeResult(id json.RawMessage, result any) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	resp := rpcResponse{JSONRPC: "2.0", ID: id, Result: result}
	if err := s.enc.Encode(resp); err != nil {
		s.Logger.Error("mcp: write", "err", err)
	}
}

func (s *Server) writeError(id json.RawMessage, code int, msg string) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if id == nil {
		id = json.RawMessage("null")
	}
	resp := rpcResponse{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: msg}}
	if err := s.enc.Encode(resp); err != nil {
		s.Logger.Error("mcp: write", "err", err)
	}
}
