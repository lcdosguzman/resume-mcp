package mcp

import "encoding/json"

// ---------- JSON-RPC 2.0 ----------

// Request represents an incoming JSON-RPC 2.0 message.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response represents an outgoing JSON-RPC 2.0 message.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC 2.0 error.
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Standard JSON-RPC error codes.
const (
	CodeParseError     = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternalError  = -32603
)

// ---------- MCP: initialize ----------

// InitializeResult is the response to the "initialize" method.
type InitializeResult struct {
	ProtocolVersion string       `json:"protocolVersion"`
	Capabilities    Capabilities `json:"capabilities"`
	ServerInfo      ServerInfo   `json:"serverInfo"`
}

// Capabilities describes the features supported by the server.
type Capabilities struct {
	Tools   *ToolsCapability   `json:"tools,omitempty"`
	Prompts *PromptsCapability `json:"prompts,omitempty"`
}

// ToolsCapability indicates tool support and whether list changes are notified.
type ToolsCapability struct {
	ListChanged bool `json:"listChanged"`
}

// PromptsCapability indicates prompt support and whether list changes are notified.
type PromptsCapability struct {
	ListChanged bool `json:"listChanged"`
}

// ServerInfo identifies the MCP server.
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ---------- MCP: prompts ----------

// Prompt describes a reusable prompt exposed by the server.
type Prompt struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Arguments   []PromptArgument `json:"arguments,omitempty"`
}

// PromptArgument describes an argument accepted by a prompt.
type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// ListPromptsResult is the response to the "prompts/list" method.
type ListPromptsResult struct {
	Prompts []Prompt `json:"prompts"`
}

// GetPromptParams are the parameters for the "prompts/get" method.
type GetPromptParams struct {
	Name      string            `json:"name"`
	Arguments map[string]string `json:"arguments,omitempty"`
}

// PromptMessage is a message returned by a prompt.
type PromptMessage struct {
	Role    string       `json:"role"`
	Content ContentBlock `json:"content"`
}

// GetPromptResult is the response to the "prompts/get" method.
type GetPromptResult struct {
	Description string          `json:"description,omitempty"`
	Messages    []PromptMessage `json:"messages"`
}

// ---------- MCP: tools/list ----------

// Tool describes a tool exposed by the server.
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

// InputSchema is a simplified JSON Schema for a tool's arguments.
type InputSchema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties,omitempty"`
	Required   []string            `json:"required,omitempty"`
}

// Property describes a field within the InputSchema.
type Property struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

// ListToolsResult is the response to the "tools/list" method.
type ListToolsResult struct {
	Tools []Tool `json:"tools"`
}

// ---------- MCP: tools/call ----------

// CallToolParams are the parameters for the "tools/call" method.
type CallToolParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// ContentBlock is a content block returned by a tool (text for now).
type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// CallToolResult is the response to the "tools/call" method.
type CallToolResult struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}
