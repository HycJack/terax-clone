// Package mcp implements a lightweight MCP (Model Context Protocol) server
// that exposes IDE tools to AI agents via stdio or HTTP transport.
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	internalfs "terax/internal/fs"
	"terax/internal/types"
	"terax/internal/workspace"
)

// Server is a minimal MCP server exposing IDE tools.
type Server struct {
	mu      sync.Mutex
	tools   []Tool
	handler func(ctx context.Context, tool string, args json.RawMessage) (any, error)
}

// Tool describes an MCP tool exposed to agents.
type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

// NewServer creates an MCP server with built-in IDE tools.
func NewServer() *Server {
	s := &Server{}
	s.registerBuiltins()
	return s
}

// SetHandler overrides the default tool execution handler.
func (s *Server) SetHandler(h func(ctx context.Context, tool string, args json.RawMessage) (any, error)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handler = h
}

// Tools returns the list of registered tools.
func (s *Server) Tools() []Tool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.tools
}

// CallTool executes a tool by name.
func (s *Server) CallTool(ctx context.Context, name string, args json.RawMessage) (any, error) {
	s.mu.Lock()
	handler := s.handler
	s.mu.Unlock()
	return handler(ctx, name, args)
}

// ServeStdio reads JSON-RPC messages from stdin and writes responses to stdout.
func (s *Server) ServeStdio(ctx context.Context) error {
	return s.serve(ctx, os.Stdin, os.Stdout)
}

// Serve reads from r and writes to w.
func (s *Server) Serve(ctx context.Context, r io.Reader, w io.Writer) error {
	return s.serve(ctx, r, w)
}

func (s *Server) serve(ctx context.Context, r io.Reader, w io.Writer) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	encoder := json.NewEncoder(w)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var msg jsonrpcMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}

		resp := s.handleMessage(ctx, &msg)
		if resp != nil {
			if err := encoder.Encode(resp); err != nil {
				return fmt.Errorf("write response: %w", err)
			}
		}
	}

	return scanner.Err()
}

type jsonrpcMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonrpcResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id"`
	Result  any    `json:"result,omitempty"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (s *Server) handleMessage(ctx context.Context, msg *jsonrpcMessage) *jsonrpcResponse {
	resp := &jsonrpcResponse{
		JSONRPC: "2.0",
		ID:      msg.ID,
	}

	switch msg.Method {
	case "initialize":
		resp.Result = map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
			"serverInfo": map[string]any{
				"name":    "terax-mcp",
				"version": "0.1.0",
			},
		}

	case "tools/list":
		s.mu.Lock()
		tools := s.tools
		s.mu.Unlock()
		resp.Result = map[string]any{"tools": tools}

	case "tools/call":
		var params struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(msg.Params, &params); err != nil {
			resp.Error = &struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			}{Code: -32602, Message: "invalid params"}
			return resp
		}

		s.mu.Lock()
		handler := s.handler
		s.mu.Unlock()

		result, err := handler(ctx, params.Name, params.Arguments)
		if err != nil {
			resp.Error = &struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			}{Code: -32000, Message: err.Error()}
			return resp
		}

		resp.Result = map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": formatResult(result)},
			},
		}

	case "notifications/initialized":
		return nil

	default:
		if msg.ID != nil {
			resp.Error = &struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			}{Code: -32601, Message: "method not found"}
			return resp
		}
		return nil
	}

	return resp
}

func formatResult(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}

func (s *Server) registerBuiltins() {
	s.tools = []Tool{
		{
			Name:        "read_file",
			Description: "Read the contents of a file. Returns the file content as text.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Absolute file path to read",
					},
				},
				"required": []string{"path"},
			},
		},
		{
			Name:        "write_file",
			Description: "Write content to a file. Creates the file if it doesn't exist.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Absolute file path to write",
					},
					"content": map[string]any{
						"type":        "string",
						"description": "Content to write",
					},
				},
				"required": []string{"path", "content"},
			},
		},
		{
			Name:        "list_directory",
			Description: "List files and directories at the given path.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Directory path to list",
					},
				},
				"required": []string{"path"},
			},
		},
		{
			Name:        "search_files",
			Description: "Search for files matching a glob pattern.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"pattern": map[string]any{
						"type":        "string",
						"description": "Glob pattern (e.g. **/*.go)",
					},
					"root": map[string]any{
						"type":        "string",
						"description": "Root directory to search from",
					},
				},
				"required": []string{"pattern"},
			},
		},
		{
			Name:        "grep",
			Description: "Search file contents using regex patterns.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"pattern": map[string]any{
						"type":        "string",
						"description": "Regex pattern to search for",
					},
					"path": map[string]any{
						"type":        "string",
						"description": "Directory to search in",
					},
					"include": map[string]any{
						"type":        "string",
						"description": "File glob to include (e.g. *.go)",
					},
				},
				"required": []string{"pattern"},
			},
		},
		{
			Name:        "workspace_info",
			Description: "Get information about the current workspace.",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
	}

	s.SetHandler(s.defaultHandler)
}

func (s *Server) defaultHandler(ctx context.Context, tool string, args json.RawMessage) (any, error) {
	switch tool {
	case "read_file":
		var p struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal(args, &p); err != nil {
			return nil, err
		}
		if !workspace.IsAuthorized(p.Path) {
			return nil, fmt.Errorf("path not in authorized workspace")
		}
		result, err := internalfs.ReadFile(p.Path)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"content": result.Content,
			"kind":    result.Kind,
			"size":    result.Size,
		}, nil

	case "write_file":
		var p struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		}
		if err := json.Unmarshal(args, &p); err != nil {
			return nil, err
		}
		if !workspace.IsAuthorized(p.Path) {
			return nil, fmt.Errorf("path not in authorized workspace")
		}
		if err := internalfs.WriteFile(types.FsWriteArgs{Path: p.Path, Content: p.Content}); err != nil {
			return nil, err
		}
		return map[string]any{"success": true}, nil

	case "list_directory":
		var p struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal(args, &p); err != nil {
			return nil, err
		}
		if !workspace.IsAuthorized(p.Path) {
			return nil, fmt.Errorf("path not in authorized workspace")
		}
		entries, err := internalfs.ReadDir(types.FsReadDirArgs{
			Path:       p.Path,
			ShowHidden: false,
		})
		if err != nil {
			return nil, err
		}
		return map[string]any{"entries": entries}, nil

	case "search_files":
		var p struct {
			Pattern string `json:"pattern"`
			Root    string `json:"root"`
		}
		if err := json.Unmarshal(args, &p); err != nil {
			return nil, err
		}
		root := p.Root
		if root == "" {
			root = workspace.CurrentDir()
		}
		if !workspace.IsAuthorized(root) {
			return nil, fmt.Errorf("path not in authorized workspace")
		}
		result, err := internalfs.Glob(types.FsGlobArgs{
			Pattern:   p.Pattern,
			Root:      root,
			MaxResults: 500,
		})
		if err != nil {
			return nil, err
		}
		return map[string]any{"matches": result.Hits, "truncated": result.Truncated}, nil

	case "grep":
		var p struct {
			Pattern string `json:"pattern"`
			Path    string `json:"path"`
			Include string `json:"include"`
		}
		if err := json.Unmarshal(args, &p); err != nil {
			return nil, err
		}
		root := p.Path
		if root == "" {
			root = workspace.CurrentDir()
		}
		if !workspace.IsAuthorized(root) {
			return nil, fmt.Errorf("path not in authorized workspace")
		}
		includeGlob := []string{}
		if p.Include != "" {
			includeGlob = []string{p.Include}
		}
		result, err := internalfs.Grep(types.FsGrepArgs{
			Root:        root,
			Pattern:     p.Pattern,
			Glob:        includeGlob,
			MaxResults:  500,
		})
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"hits":          result.Hits,
			"filesScanned":  result.FilesScanned,
			"truncated":     result.Truncated,
		}, nil

	case "workspace_info":
		cwd := workspace.CurrentDir()
		return map[string]any{
			"cwd": cwd,
		}, nil

	default:
		return nil, fmt.Errorf("unknown tool: %s", tool)
	}
}
