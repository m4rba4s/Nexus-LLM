package mcp

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewMCPServer(t *testing.T) {
	t.Parallel()
	srv := NewMCPServer(9090, "test-token")
	if srv == nil {
		t.Fatal("NewMCPServer returned nil")
	}
}

func TestMCPServer_RegisterTool(t *testing.T) {
	t.Parallel()
	srv := NewMCPServer(0, "")
	srv.RegisterTool(Tool{
		Name:        "test_tool",
		Description: "A test tool",
		Parameters: map[string]Parameter{
			"input": {Type: "string", Required: true},
		},
		Handler: func(params map[string]interface{}) (interface{}, error) {
			return "ok", nil
		},
	})

	if _, exists := srv.tools["test_tool"]; !exists {
		t.Error("registered tool not found in server.tools")
	}
}

func TestMCPServer_RegisterResource(t *testing.T) {
	t.Parallel()
	srv := NewMCPServer(0, "")
	srv.RegisterResource(Resource{
		URI:         "file:///test.txt",
		Name:        "test",
		Description: "test resource",
		MimeType:    "text/plain",
		Content:     []byte("hello"),
		UpdatedAt:   time.Now(),
	})

	if len(srv.resources) != 1 {
		t.Errorf("expected 1 resource, got %d", len(srv.resources))
	}
}

func TestMessage_JSON(t *testing.T) {
	t.Parallel()

	msg := Message{
		ID:        "msg-1",
		Type:      "request",
		Method:    "tools/call",
		Params:    map[string]interface{}{"name": "test"},
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal Message: %v", err)
	}

	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal Message: %v", err)
	}

	if decoded.ID != msg.ID {
		t.Errorf("ID = %q, want %q", decoded.ID, msg.ID)
	}
	if decoded.Method != msg.Method {
		t.Errorf("Method = %q, want %q", decoded.Method, msg.Method)
	}
}

func TestError_JSON(t *testing.T) {
	t.Parallel()

	e := Error{Code: -32600, Message: "Invalid Request"}
	data, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}

	var decoded Error
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Code != -32600 {
		t.Errorf("Code = %d, want -32600", decoded.Code)
	}
}

func TestRateLimiter_Allow(t *testing.T) {
	t.Parallel()

	rl := &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    3,
		window:   time.Second,
	}

	key := "test-client"
	for i := 0; i < 3; i++ {
		if !rl.Allow(key) {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}

	// 4th request should be denied
	if rl.Allow(key) {
		t.Error("4th request should be rate-limited")
	}
}

func TestTool_Struct(t *testing.T) {
	t.Parallel()

	tool := Tool{
		Name:         "execute_code",
		Description:  "Executes code safely",
		RequiresAuth: true,
		Parameters: map[string]Parameter{
			"language": {Type: "string", Required: true, Enum: []string{"python", "go"}},
			"code":     {Type: "string", Required: true},
		},
	}

	if tool.Name != "execute_code" {
		t.Error("wrong tool name")
	}
	if !tool.RequiresAuth {
		t.Error("should require auth")
	}
	if len(tool.Parameters) != 2 {
		t.Errorf("expected 2 parameters, got %d", len(tool.Parameters))
	}
}
