package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// MCPServer implements Model Context Protocol server
type MCPServer struct {
	router      *mux.Router
	upgrader    websocket.Upgrader
	clients     map[string]*Client
	clientsMu   sync.RWMutex
	tools       map[string]Tool
	resources   map[string]Resource
	port        int
	authToken   string
	rateLimiter *RateLimiter
}

// Client represents a connected MCP client
type Client struct {
	ID         string
	Conn       *websocket.Conn
	Send       chan Message
	Tools      []string
	Authorized bool
}

// Message represents an MCP message
type Message struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Method    string                 `json:"method,omitempty"`
	Params    map[string]interface{} `json:"params,omitempty"`
	Result    interface{}            `json:"result,omitempty"`
	Error     *Error                 `json:"error,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// Error represents an MCP error
type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Tool represents an MCP tool
type Tool struct {
	Name         string               `json:"name"`
	Description  string               `json:"description"`
	Parameters   map[string]Parameter `json:"parameters"`
	Handler      func(params map[string]interface{}) (interface{}, error)
	RequiresAuth bool `json:"requires_auth"`
}

// Parameter represents a tool parameter
type Parameter struct {
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default,omitempty"`
	Enum        []string    `json:"enum,omitempty"`
}

// Resource represents an MCP resource
type Resource struct {
	URI         string    `json:"uri"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	MimeType    string    `json:"mime_type"`
	Content     []byte    `json:"-"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// RateLimiter implements rate limiting
type RateLimiter struct {
	requests map[string][]time.Time
	mu       sync.Mutex
	limit    int
	window   time.Duration
}

// NewMCPServer creates a new MCP server
func NewMCPServer(port int, authToken string) *MCPServer {
	s := &MCPServer{
		router:    mux.NewRouter(),
		upgrader:  websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
		clients:   make(map[string]*Client),
		tools:     make(map[string]Tool),
		resources: make(map[string]Resource),
		port:      port,
		authToken: authToken,
		rateLimiter: &RateLimiter{
			requests: make(map[string][]time.Time),
			limit:    100,
			window:   time.Minute,
		},
	}

	s.registerDefaultTools()
	s.setupRoutes()
	return s
}

// Start starts the MCP server
func (s *MCPServer) Start(ctx context.Context) error {
    server := &http.Server{
        Addr:    fmt.Sprintf("127.0.0.1:%d", s.port),
        Handler: s.router,
    }

	go func() {
		<-ctx.Done()
		server.Shutdown(context.Background())
	}()

	fmt.Printf("🔧 MCP Server starting on port %d\n", s.port)
	return server.ListenAndServe()
}

// setupRoutes sets up HTTP routes
func (s *MCPServer) setupRoutes() {
	s.router.HandleFunc("/mcp", s.handleWebSocket)
	s.router.HandleFunc("/tools", s.handleTools).Methods("GET")
	s.router.HandleFunc("/resources", s.handleResources).Methods("GET")
	s.router.HandleFunc("/execute", s.handleExecute).Methods("POST")
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")
}

// handleWebSocket handles WebSocket connections
func (s *MCPServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	client := &Client{
		ID:   generateID(),
		Conn: conn,
		Send: make(chan Message, 256),
	}

	s.clientsMu.Lock()
	s.clients[client.ID] = client
	s.clientsMu.Unlock()

	go client.writePump()
	go client.readPump(s)

	// Send welcome message
	client.Send <- Message{
		Type: "welcome",
		Result: map[string]interface{}{
			"version":      "1.0.0",
			"capabilities": []string{"tools", "resources", "streaming"},
		},
		Timestamp: time.Now(),
	}
}

// handleTools handles tool listing
func (s *MCPServer) handleTools(w http.ResponseWriter, r *http.Request) {
	if !s.authenticate(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	tools := make([]map[string]interface{}, 0)
	for name, tool := range s.tools {
		tools = append(tools, map[string]interface{}{
			"name":        name,
			"description": tool.Description,
			"parameters":  tool.Parameters,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tools)
}

// handleExecute handles tool execution
func (s *MCPServer) handleExecute(w http.ResponseWriter, r *http.Request) {
	if !s.authenticate(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Tool   string                 `json:"tool"`
		Params map[string]interface{} `json:"params"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tool, exists := s.tools[req.Tool]
	if !exists {
		http.Error(w, "Tool not found", http.StatusNotFound)
		return
	}

	// Rate limiting
	if !s.rateLimiter.Allow(r.RemoteAddr) {
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	result, err := tool.Handler(req.Params)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"result": result,
	})
}

// registerDefaultTools registers default MCP tools
func (s *MCPServer) registerDefaultTools() {
	// File operations
	s.RegisterTool(Tool{
		Name:        "read_file",
		Description: "Read contents of a file",
		Parameters: map[string]Parameter{
			"path": {Type: "string", Description: "File path", Required: true},
		},
		Handler: func(params map[string]interface{}) (interface{}, error) {
			path, ok := params["path"].(string)
			if !ok {
				return nil, fmt.Errorf("invalid path parameter")
			}

			// Security check
			if strings.Contains(path, "..") {
				return nil, fmt.Errorf("path traversal not allowed")
			}

			content, err := os.ReadFile(path)
			if err != nil {
				return nil, err
			}

			return map[string]interface{}{
				"content": string(content),
				"size":    len(content),
			}, nil
		},
		RequiresAuth: true,
	})

	s.RegisterTool(Tool{
		Name:        "write_file",
		Description: "Write content to a file",
		Parameters: map[string]Parameter{
			"path":    {Type: "string", Description: "File path", Required: true},
			"content": {Type: "string", Description: "File content", Required: true},
		},
		Handler: func(params map[string]interface{}) (interface{}, error) {
			path, _ := params["path"].(string)
			content, _ := params["content"].(string)

			if strings.Contains(path, "..") {
				return nil, fmt.Errorf("path traversal not allowed")
			}

			err := os.WriteFile(path, []byte(content), 0644)
			if err != nil {
				return nil, err
			}

			return map[string]interface{}{
				"success": true,
				"size":    len(content),
			}, nil
		},
		RequiresAuth: true,
	})

	// Command execution
	s.RegisterTool(Tool{
		Name:        "execute_command",
		Description: "Execute a system command",
		Parameters: map[string]Parameter{
			"command": {Type: "string", Description: "Command to execute", Required: true},
			"args":    {Type: "array", Description: "Command arguments", Required: false},
		},
		Handler: func(params map[string]interface{}) (interface{}, error) {
			command, _ := params["command"].(string)
			args, _ := params["args"].([]interface{})

			// Security check - whitelist safe commands
			safeCommands := []string{"ls", "pwd", "date", "echo", "cat", "grep", "find"}
			cmdBase := strings.Fields(command)[0]

			isSafe := false
			for _, safe := range safeCommands {
				if cmdBase == safe {
					isSafe = true
					break
				}
			}

			if !isSafe {
				return nil, fmt.Errorf("command not in whitelist: %s", cmdBase)
			}

			var argStrings []string
			for _, arg := range args {
				if s, ok := arg.(string); ok {
					argStrings = append(argStrings, s)
				}
			}

			cmd := exec.Command(command, argStrings...)
			output, err := cmd.CombinedOutput()

			return map[string]interface{}{
				"output": string(output),
				"error":  err != nil,
			}, nil
		},
		RequiresAuth: true,
	})

    // Web requests
    s.RegisterTool(Tool{
        Name:        "http_request",
        Description: "Make an HTTP request",
		Parameters: map[string]Parameter{
			"url":    {Type: "string", Description: "URL to request", Required: true},
			"method": {Type: "string", Description: "HTTP method", Default: "GET"},
		},
		Handler: func(params map[string]interface{}) (interface{}, error) {
			url, _ := params["url"].(string)
			method, _ := params["method"].(string)
			if method == "" {
				method = "GET"
			}

			client := &http.Client{Timeout: 10 * time.Second}
			req, err := http.NewRequest(method, url, nil)
			if err != nil {
				return nil, err
			}

			resp, err := client.Do(req)
			if err != nil {
				return nil, err
			}
			defer resp.Body.Close()

			body, _ := io.ReadAll(resp.Body)

			return map[string]interface{}{
				"status":  resp.StatusCode,
				"body":    string(body),
				"headers": resp.Header,
			}, nil
		},
        // Require auth to avoid abuse via unauthenticated websocket clients
        RequiresAuth: true,
    })

	// Code analysis
	s.RegisterTool(Tool{
		Name:        "analyze_code",
		Description: "Analyze code for issues and suggestions",
		Parameters: map[string]Parameter{
			"code":     {Type: "string", Description: "Code to analyze", Required: true},
			"language": {Type: "string", Description: "Programming language", Required: true},
		},
		Handler: func(params map[string]interface{}) (interface{}, error) {
			code, _ := params["code"].(string)
			language, _ := params["language"].(string)

			// Simplified analysis (in production, use real tools)
			analysis := map[string]interface{}{
				"language": language,
				"lines":    len(strings.Split(code, "\n")),
				"issues":   []string{},
				"suggestions": []string{
					"Consider adding comments",
					"Check error handling",
				},
			}

			// Basic checks
			if !strings.Contains(code, "error") && language == "go" {
				analysis["issues"] = append(analysis["issues"].([]string), "Missing error handling")
			}

			return analysis, nil
		},
		RequiresAuth: false,
	})

	// Data processing
	s.RegisterTool(Tool{
		Name:        "json_parse",
		Description: "Parse and query JSON data",
		Parameters: map[string]Parameter{
			"data":  {Type: "string", Description: "JSON data", Required: true},
			"query": {Type: "string", Description: "JQ-like query", Required: false},
		},
		Handler: func(params map[string]interface{}) (interface{}, error) {
			data, _ := params["data"].(string)

			var parsed interface{}
			if err := json.Unmarshal([]byte(data), &parsed); err != nil {
				return nil, err
			}

			return parsed, nil
		},
		RequiresAuth: false,
	})
}

// RegisterTool registers a new tool
func (s *MCPServer) RegisterTool(tool Tool) {
	s.tools[tool.Name] = tool
}

// RegisterResource registers a new resource
func (s *MCPServer) RegisterResource(resource Resource) {
	s.resources[resource.URI] = resource
}

// authenticate checks authentication
func (s *MCPServer) authenticate(r *http.Request) bool {
	if s.authToken == "" {
		return true
	}

	token := r.Header.Get("Authorization")
	return token == "Bearer "+s.authToken
}

// Client methods

func (c *Client) readPump(server *MCPServer) {
	defer func() {
		server.clientsMu.Lock()
		delete(server.clients, c.ID)
		server.clientsMu.Unlock()
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var msg Message
		err := c.Conn.ReadJSON(&msg)
		if err != nil {
			break
		}

		// Handle message
		response := server.handleMessage(c, msg)
		if response != nil {
			c.Send <- *response
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteJSON(message); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage handles incoming messages
func (s *MCPServer) handleMessage(client *Client, msg Message) *Message {
	switch msg.Method {
	case "tools.list":
		tools := make([]map[string]interface{}, 0)
		for name, tool := range s.tools {
			if !tool.RequiresAuth || client.Authorized {
				tools = append(tools, map[string]interface{}{
					"name":        name,
					"description": tool.Description,
				})
			}
		}
		return &Message{
			ID:        msg.ID,
			Type:      "response",
			Result:    tools,
			Timestamp: time.Now(),
		}

	case "tools.execute":
		toolName, _ := msg.Params["tool"].(string)
		toolParams, _ := msg.Params["params"].(map[string]interface{})

		tool, exists := s.tools[toolName]
		if !exists {
			return &Message{
				ID:   msg.ID,
				Type: "error",
				Error: &Error{
					Code:    404,
					Message: "Tool not found",
				},
				Timestamp: time.Now(),
			}
		}

		if tool.RequiresAuth && !client.Authorized {
			return &Message{
				ID:   msg.ID,
				Type: "error",
				Error: &Error{
					Code:    403,
					Message: "Unauthorized",
				},
				Timestamp: time.Now(),
			}
		}

		result, err := tool.Handler(toolParams)
		if err != nil {
			return &Message{
				ID:   msg.ID,
				Type: "error",
				Error: &Error{
					Code:    500,
					Message: err.Error(),
				},
				Timestamp: time.Now(),
			}
		}

		return &Message{
			ID:        msg.ID,
			Type:      "response",
			Result:    result,
			Timestamp: time.Now(),
		}

	case "auth":
		token, _ := msg.Params["token"].(string)
		client.Authorized = (token == s.authToken)

		return &Message{
			ID:        msg.ID,
			Type:      "response",
			Result:    map[string]bool{"authorized": client.Authorized},
			Timestamp: time.Now(),
		}

	default:
		return &Message{
			ID:   msg.ID,
			Type: "error",
			Error: &Error{
				Code:    400,
				Message: "Unknown method",
			},
			Timestamp: time.Now(),
		}
	}
}

// RateLimiter methods

func (r *RateLimiter) Allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	requests := r.requests[key]

	// Remove old requests
	var validRequests []time.Time
	for _, t := range requests {
		if now.Sub(t) < r.window {
			validRequests = append(validRequests, t)
		}
	}

	if len(validRequests) >= r.limit {
		return false
	}

	r.requests[key] = append(validRequests, now)
	return true
}

// Utility functions

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func (s *MCPServer) handleResources(w http.ResponseWriter, r *http.Request) {
	resources := make([]map[string]interface{}, 0)
	for uri, res := range s.resources {
		resources = append(resources, map[string]interface{}{
			"uri":         uri,
			"name":        res.Name,
			"description": res.Description,
			"mime_type":   res.MimeType,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resources)
}

func (s *MCPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.clientsMu.RLock()
	clientCount := len(s.clients)
	s.clientsMu.RUnlock()

	health := map[string]interface{}{
		"status":  "healthy",
		"clients": clientCount,
		"tools":   len(s.tools),
		"uptime":  time.Since(time.Now()).String(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}
