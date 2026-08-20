package mcp

import (
	"encoding/json"
	"log"
	"net/http"
)

// ToolHandler es la función que implementa la lógica de una tool.
// Recibe los argumentos crudos (JSON) y devuelve el resultado o un error.
type ToolHandler func(args json.RawMessage) (CallToolResult, error)

// registeredTool agrupa la definición de una tool con su handler.
type registeredTool struct {
	def     Tool
	handler ToolHandler
}

// Server es el servidor MCP HTTP (Streamable HTTP, modo simple sin SSE).
type Server struct {
	name    string
	version string
	tools   map[string]registeredTool
}

// NewServer crea un servidor MCP nuevo.
func NewServer(name, version string) *Server {
	return &Server{
		name:    name,
		version: version,
		tools:   make(map[string]registeredTool),
	}
}

// RegisterTool agrega una tool al servidor.
func (s *Server) RegisterTool(def Tool, handler ToolHandler) {
	s.tools[def.Name] = registeredTool{def: def, handler: handler}
}

// Handler devuelve el http.Handler a montar en el mux (endpoint /mcp).
func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(s.serveHTTP)
}

func (s *Server) serveHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		// El transporte "Streamable HTTP" completo soporta GET para streams
		// server-initiated (SSE). Esta implementación simple solo necesita
		// request/response, así que devolvemos 405 para otros métodos.
		w.Header().Set("Allow", "POST")
		http.Error(w, "método no soportado, usar POST", http.StatusMethodNotAllowed)
		return
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, Response{
			JSONRPC: "2.0",
			Error:   &RPCError{Code: CodeParseError, Message: "JSON inválido: " + err.Error()},
		})
		return
	}

	resp := s.dispatch(req)

	// Las notificaciones (sin "id") no llevan respuesta en JSON-RPC.
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
		// Notificaciones del cliente; no requieren respuesta con contenido.
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
			base.Error = &RPCError{Code: CodeInvalidParams, Message: "parámetros inválidos: " + err.Error()}
			return base
		}
		tool, ok := s.tools[params.Name]
		if !ok {
			base.Error = &RPCError{Code: CodeMethodNotFound, Message: "tool desconocida: " + params.Name}
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
		base.Error = &RPCError{Code: CodeMethodNotFound, Message: "método desconocido: " + req.Method}
	}

	return base
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("error escribiendo respuesta JSON: %v", err)
	}
}
