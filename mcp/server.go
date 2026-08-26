package mcp

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
)

const maxRequestBodySize = 1 << 20

// ToolHandler implements a tool's logic.
// It receives raw JSON arguments and returns the result or an error.
type ToolHandler func(args json.RawMessage) (CallToolResult, error)

// PromptHandler implements a prompt's logic.
// It receives string arguments and returns the prompt messages or an error.
type PromptHandler func(args map[string]string) (GetPromptResult, error)

// registeredTool groups a tool definition with its handler.
type registeredTool struct {
	def     Tool
	handler ToolHandler
}

// registeredPrompt groups a prompt definition with its handler.
type registeredPrompt struct {
	def     Prompt
	handler PromptHandler
}

// Server is the MCP HTTP server (simple Streamable HTTP mode without SSE).
type Server struct {
	name    string
	version string
	tools   map[string]registeredTool
	prompts map[string]registeredPrompt
}

// NewServer creates a new MCP server.
func NewServer(name, version string) *Server {
	return &Server{
		name:    name,
		version: version,
		tools:   make(map[string]registeredTool),
		prompts: make(map[string]registeredPrompt),
	}
}

// RegisterTool adds a tool to the server.
func (s *Server) RegisterTool(def Tool, handler ToolHandler) {
	s.tools[def.Name] = registeredTool{def: def, handler: handler}
}

// RegisterPrompt adds a prompt to the server.
func (s *Server) RegisterPrompt(def Prompt, handler PromptHandler) {
	s.prompts[def.Name] = registeredPrompt{def: def, handler: handler}
}

// Handler returns the http.Handler to mount on the mux (the /mcp endpoint).
func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(s.serveHTTP)
}

func (s *Server) serveHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		// Full "Streamable HTTP" transport supports GET for server-initiated
		// streams (SSE). This simple implementation only needs request/response,
		// so it returns 405 for other methods.
		w.Header().Set("Allow", "POST")
		http.Error(w, "method not supported, use POST", http.StatusMethodNotAllowed)
		return
	}

	var req Request
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestBodySize))
	if err := decoder.Decode(&req); err != nil {
		writeJSON(w, Response{
			JSONRPC: "2.0",
			Error:   &RPCError{Code: CodeParseError, Message: "invalid JSON: " + err.Error()},
		})
		return
	}
	var extra interface{}
	if err := decoder.Decode(&extra); err != io.EOF {
		writeJSON(w, Response{
			JSONRPC: "2.0",
			Error:   &RPCError{Code: CodeParseError, Message: "request must contain exactly one JSON value"},
		})
		return
	}

	resp := s.dispatch(req)

	// Notifications (without an "id") do not receive a JSON-RPC response.
	if req.ID == nil {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	writeJSON(w, resp)
}

func (s *Server) dispatch(req Request) Response {
	base := Response{JSONRPC: "2.0", ID: req.ID}
	if err := validateRequest(req); err != nil {
		base.Error = &RPCError{Code: CodeInvalidRequest, Message: err.Error()}
		return base
	}

	switch req.Method {
	case "initialize":
		base.Result = InitializeResult{
			ProtocolVersion: "2025-03-26",
			Capabilities: Capabilities{
				Tools:   &ToolsCapability{ListChanged: false},
				Prompts: &PromptsCapability{ListChanged: false},
			},
			ServerInfo: ServerInfo{Name: s.name, Version: s.version},
		}

	case "notifications/initialized", "notifications/cancelled":
		// Client notifications; they do not require a content response.
		return base

	case "tools/list":
		names := sortedNames(s.tools)
		defs := make([]Tool, 0, len(names))
		for _, name := range names {
			defs = append(defs, s.tools[name].def)
		}
		base.Result = ListToolsResult{Tools: defs}

	case "tools/call":
		var params CallToolParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			base.Error = &RPCError{Code: CodeInvalidParams, Message: "invalid parameters: " + err.Error()}
			return base
		}
		if params.Name == "" {
			base.Error = &RPCError{Code: CodeInvalidParams, Message: "tool name cannot be empty"}
			return base
		}
		tool, ok := s.tools[params.Name]
		if !ok {
			base.Error = &RPCError{Code: CodeMethodNotFound, Message: "unknown tool: " + params.Name}
			return base
		}
		result, err := tool.handler(params.Arguments)
		if err != nil {
			base.Result = CallToolResult{
				Content: []ContentBlock{{Type: "text", Text: err.Error()}},
				IsError: true,
			}
			return base
		}
		base.Result = result

	case "prompts/list":
		names := sortedNames(s.prompts)
		defs := make([]Prompt, 0, len(names))
		for _, name := range names {
			defs = append(defs, s.prompts[name].def)
		}
		base.Result = ListPromptsResult{Prompts: defs}

	case "prompts/get":
		var params GetPromptParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			base.Error = &RPCError{Code: CodeInvalidParams, Message: "invalid parameters: " + err.Error()}
			return base
		}
		if params.Name == "" {
			base.Error = &RPCError{Code: CodeInvalidParams, Message: "prompt name cannot be empty"}
			return base
		}
		prompt, ok := s.prompts[params.Name]
		if !ok {
			base.Error = &RPCError{Code: CodeMethodNotFound, Message: "unknown prompt: " + params.Name}
			return base
		}
		result, err := prompt.handler(params.Arguments)
		if err != nil {
			base.Error = &RPCError{Code: CodeInvalidParams, Message: err.Error()}
			return base
		}
		base.Result = result

	default:
		base.Error = &RPCError{Code: CodeMethodNotFound, Message: "unknown method: " + req.Method}
	}

	return base
}

func validateRequest(req Request) error {
	if req.JSONRPC != "2.0" {
		return fmt.Errorf("jsonrpc must be \"2.0\"")
	}
	if req.Method == "" {
		return fmt.Errorf("method cannot be empty")
	}
	if req.ID != nil {
		var id interface{}
		if err := json.Unmarshal(req.ID, &id); err != nil {
			return fmt.Errorf("invalid id: %w", err)
		}
		switch id.(type) {
		case nil, string, float64:
		default:
			return fmt.Errorf("id must be a string, number, or null")
		}
	}
	return nil
}

func sortedNames[T any](registered map[string]T) []string {
	names := make([]string, 0, len(registered))
	for name := range registered {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("error writing JSON response: %v", err)
	}
}
