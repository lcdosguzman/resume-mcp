package mcp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDispatch(t *testing.T) {
	t.Run("when request is invalid then dispatch returns invalid request error", func(t *testing.T) {
		server := NewServer("test", "1.0.0")

		response := server.dispatch(Request{JSONRPC: "1.0", ID: json.RawMessage(`1`), Method: "tools/list"})

		require.NotNil(t, response.Error)
		assert.Equal(t, CodeInvalidRequest, response.Error.Code)
	})

	t.Run("when tools and prompts are registered then dispatch sorts them alphabetically", func(t *testing.T) {
		server := NewServer("test", "1.0.0")
		require.NoError(t, server.RegisterTool(Tool{Name: "zulu"}, func(json.RawMessage) (CallToolResult, error) {
			return CallToolResult{}, nil
		}))
		require.NoError(t, server.RegisterTool(Tool{Name: "alpha"}, func(json.RawMessage) (CallToolResult, error) {
			return CallToolResult{}, nil
		}))
		require.NoError(t, server.RegisterPrompt(Prompt{Name: "zulu"}, func(map[string]string) (GetPromptResult, error) {
			return GetPromptResult{}, nil
		}))
		require.NoError(t, server.RegisterPrompt(Prompt{Name: "alpha"}, func(map[string]string) (GetPromptResult, error) {
			return GetPromptResult{}, nil
		}))

		toolResponse := server.dispatch(Request{JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: "tools/list"})
		promptResponse := server.dispatch(Request{JSONRPC: "2.0", ID: json.RawMessage(`2`), Method: "prompts/list"})

		tools := toolResponse.Result.(ListToolsResult).Tools
		prompts := promptResponse.Result.(ListPromptsResult).Prompts
		assert.True(t, reflect.DeepEqual([]string{tools[0].Name, tools[1].Name}, []string{"alpha", "zulu"}))
		assert.True(t, reflect.DeepEqual([]string{prompts[0].Name, prompts[1].Name}, []string{"alpha", "zulu"}))
	})
}

func TestHandler(t *testing.T) {
	t.Run("when a request contains multiple JSON values then Handler returns parse error", func(t *testing.T) {
		server := NewServer("test", "1.0.0")
		request := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"} {}`))
		recorder := httptest.NewRecorder()

		server.Handler().ServeHTTP(recorder, request)

		var response Response
		require.NoError(t, json.NewDecoder(recorder.Body).Decode(&response))
		require.NotNil(t, response.Error)
		assert.Equal(t, CodeParseError, response.Error.Code)
	})

	t.Run("when an RPC request is missing params for tools/call then Handler returns invalid params", func(t *testing.T) {
		server := NewServer("test", "1.0.0")
		request := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/call"}`))
		recorder := httptest.NewRecorder()

		server.Handler().ServeHTTP(recorder, request)

		var response Response
		require.NoError(t, json.NewDecoder(recorder.Body).Decode(&response))
		require.NotNil(t, response.Error)
		assert.Equal(t, CodeInvalidParams, response.Error.Code)
	})
}

func TestRegisterToolAndPromptRejectInvalidAndDuplicateDefinitions(t *testing.T) {
	t.Run("when tool definition is invalid then RegisterTool returns an error", func(t *testing.T) {
		server := NewServer("test", "1.0.0")

		err := server.RegisterTool(Tool{Name: ""}, func(json.RawMessage) (CallToolResult, error) { return CallToolResult{}, nil })
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tool name")
	})

	t.Run("when a tool name is duplicated then RegisterTool returns an error and keeps the first version", func(t *testing.T) {
		server := NewServer("test", "1.0.0")
		require.NoError(t, server.RegisterTool(Tool{Name: "alpha"}, func(json.RawMessage) (CallToolResult, error) { return CallToolResult{}, nil }))

		err := server.RegisterTool(Tool{Name: "alpha"}, func(json.RawMessage) (CallToolResult, error) { return CallToolResult{IsError: true}, nil })
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already registered")
		assert.Len(t, server.tools, 1)
	})

	t.Run("when prompt definition is invalid then RegisterPrompt returns an error", func(t *testing.T) {
		server := NewServer("test", "1.0.0")

		err := server.RegisterPrompt(Prompt{Name: ""}, func(map[string]string) (GetPromptResult, error) { return GetPromptResult{}, nil })
		require.Error(t, err)
		assert.Contains(t, err.Error(), "prompt name")
	})

	t.Run("when a prompt name is duplicated then RegisterPrompt returns an error and keeps the first version", func(t *testing.T) {
		server := NewServer("test", "1.0.0")
		require.NoError(t, server.RegisterPrompt(Prompt{Name: "alpha"}, func(map[string]string) (GetPromptResult, error) { return GetPromptResult{}, nil }))

		err := server.RegisterPrompt(Prompt{Name: "alpha"}, func(map[string]string) (GetPromptResult, error) { return GetPromptResult{}, nil })
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already registered")
		assert.Len(t, server.prompts, 1)
	})
}
