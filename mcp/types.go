package mcp

import "encoding/json"

// ---------- JSON-RPC 2.0 ----------

// Request representa un mensaje JSON-RPC 2.0 entrante.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response representa un mensaje JSON-RPC 2.0 saliente.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCError representa un error JSON-RPC 2.0.
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Códigos de error estándar JSON-RPC.
const (
	CodeParseError     = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternalError  = -32603
)

// ---------- MCP: initialize ----------

// InitializeResult es la respuesta al método "initialize".
type InitializeResult struct {
	ProtocolVersion string       `json:"protocolVersion"`
	Capabilities    Capabilities `json:"capabilities"`
	ServerInfo      ServerInfo   `json:"serverInfo"`
}

// Capabilities describe qué funcionalidades soporta el servidor.
type Capabilities struct {
	Tools *ToolsCapability `json:"tools,omitempty"`
}

// ToolsCapability indica soporte de tools (y si notifica cambios en la lista).
type ToolsCapability struct {
	ListChanged bool `json:"listChanged"`
}

// ServerInfo identifica al servidor MCP.
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ---------- MCP: tools/list ----------

// Tool describe una herramienta expuesta por el servidor.
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

// InputSchema es un JSON Schema simplificado para los argumentos de una tool.
type InputSchema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties,omitempty"`
	Required   []string            `json:"required,omitempty"`
}

// Property describe un campo dentro del InputSchema.
type Property struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

// ListToolsResult es la respuesta al método "tools/list".
type ListToolsResult struct {
	Tools []Tool `json:"tools"`
}

// ---------- MCP: tools/call ----------

// CallToolParams son los parámetros del método "tools/call".
type CallToolParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// ContentBlock es un bloque de contenido devuelto por una tool (texto por ahora).
type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// CallToolResult es la respuesta al método "tools/call".
type CallToolResult struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}
