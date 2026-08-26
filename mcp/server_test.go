package mcp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestDispatchRejectsInvalidJSONRPCRequest(t *testing.T) {
	server := NewServer("test", "1.0.0")

	response := server.dispatch(Request{JSONRPC: "1.0", ID: json.RawMessage(`1`), Method: "tools/list"})

	if response.Error == nil || response.Error.Code != CodeInvalidRequest {
		t.Fatalf("expected invalid request error, got %#v", response.Error)
	}
}

func TestHandlerRejectsMultipleJSONValues(t *testing.T) {
	server := NewServer("test", "1.0.0")
	request := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"} {}`))
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	var response Response
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Error == nil || response.Error.Code != CodeParseError {
		t.Fatalf("expected parse error, got %#v", response.Error)
	}
}

func TestDispatchSortsToolsAndPrompts(t *testing.T) {
	server := NewServer("test", "1.0.0")
	server.RegisterTool(Tool{Name: "zulu"}, func(json.RawMessage) (CallToolResult, error) {
		return CallToolResult{}, nil
	})
	server.RegisterTool(Tool{Name: "alpha"}, func(json.RawMessage) (CallToolResult, error) {
		return CallToolResult{}, nil
	})
	server.RegisterPrompt(Prompt{Name: "zulu"}, func(map[string]string) (GetPromptResult, error) {
		return GetPromptResult{}, nil
	})
	server.RegisterPrompt(Prompt{Name: "alpha"}, func(map[string]string) (GetPromptResult, error) {
		return GetPromptResult{}, nil
	})

	toolResponse := server.dispatch(Request{JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: "tools/list"})
	promptResponse := server.dispatch(Request{JSONRPC: "2.0", ID: json.RawMessage(`2`), Method: "prompts/list"})

	tools := toolResponse.Result.(ListToolsResult).Tools
	prompts := promptResponse.Result.(ListPromptsResult).Prompts
	if !reflect.DeepEqual([]string{tools[0].Name, tools[1].Name}, []string{"alpha", "zulu"}) {
		t.Fatalf("tools are not sorted: %#v", tools)
	}
	if !reflect.DeepEqual([]string{prompts[0].Name, prompts[1].Name}, []string{"alpha", "zulu"}) {
		t.Fatalf("prompts are not sorted: %#v", prompts)
	}
}
