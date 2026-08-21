package mcp

import (
	"encoding/json"
	"log"
	"net/http"
)

// ToolHandler implements a tool's logic.
// It receives raw JSON arguments and returns the result or an error.
type ToolHandler func(args json.RawMessage) (CallToolResult, error)

// registeredTool groups a tool definition with its handler.
type registeredTool struct {
	def     Tool
	handler ToolHandler
}

// Server is the MCP HTTP server (simple Streamable HTTP mode without SSE).
type Server struct {
	name    string
	version string
	tools   map[string]registeredTool
}

// NewServer creates a new MCP server.
func NewServer(name, version string) *Server {
	return &Server{
		name:    name,
		version: version,
		tools:   make(map[string]registeredTool),
	}
}

// RegisterTool adds a tool to the server.
func (s *Server) RegisterTool(def Tool, handler ToolHandler) {
	s.tools[def.Name] = registeredTool{def: def, handler: handler}
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, Response{
			JSONRPC: "2.0",
			Error:   &RPCError{Code: CodeParseError, Message: "invalid JSON: " + err.Error()},
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

	switch req.Method {
	case "initialize":
		base.Result = InitializeResult{
			ProtocolVersion: "2025-03-26",
			Capabilities: Capabilities{
				Tools: &ToolsCapability{ListChanged: false},
			},
			ServerInfo: ServerInfo{Name: s.name, Version: s.version},
		}

	case "notifications/initialized", "notifications/cancelled":
		// Client notifications; they do not require a content response.
		return base

	case "tools/list":
		defs := make([]Tool, 0, len(s.tools))
		for _, t := range s.tools {
			defs = append(defs, t.def)
		}
		base.Result = ListToolsResult{Tools: defs}

	case "tools/call":
		var params CallToolParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			base.Error = &RPCError{Code: CodeInvalidParams, Message: "invalid parameters: " + err.Error()}
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

	default:
		base.Error = &RPCError{Code: CodeMethodNotFound, Message: "unknown method: " + req.Method}
	}

	return base
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("error writing JSON response: %v", err)
	}
}
