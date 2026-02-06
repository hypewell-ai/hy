package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

// TestServer wraps httptest.Server with helper methods
type TestServer struct {
	*httptest.Server
	Handlers map[string]http.HandlerFunc
}

// NewTestServer creates a mock API server
func NewTestServer() *TestServer {
	ts := &TestServer{
		Handlers: make(map[string]http.HandlerFunc),
	}

	ts.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Route to handler first (some paths like /upload/* don't need auth)
		key := r.Method + " " + r.URL.Path
		if handler, ok := ts.Handlers[key]; ok {
			handler(w, r)
			return
		}

		// Check auth for API endpoints
		auth := r.Header.Get("Authorization")
		if auth == "" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
			return
		}

		// Default 404
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Not found: " + key})
	}))

	return ts
}

// Handle registers a handler for a method + path
func (ts *TestServer) Handle(method, path string, handler http.HandlerFunc) {
	ts.Handlers[method+" "+path] = handler
}

// HandleJSON registers a handler that returns JSON
func (ts *TestServer) HandleJSON(method, path string, status int, response interface{}) {
	ts.Handle(method, path, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(response)
	})
}

// TestConfig holds test configuration
type TestConfig struct {
	Server    *TestServer
	ConfigDir string
}

// SetupTest creates a test environment
func SetupTest(t *testing.T) *TestConfig {
	t.Helper()

	// Reset viper for each test
	viper.Reset()

	// Create test server
	server := NewTestServer()

	// Configure viper directly (simpler than env vars)
	viper.Set("api_url", server.URL)
	viper.Set("workspace_id", "ws_test123")

	// Create temp config directory for any file operations
	configDir, err := os.MkdirTemp("", "hy-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	return &TestConfig{
		Server:    server,
		ConfigDir: configDir,
	}
}

// Cleanup restores the original environment
func (tc *TestConfig) Cleanup() {
	tc.Server.Close()
	os.RemoveAll(tc.ConfigDir)
	viper.Reset()
}

// ExecuteCommand runs a CLI command and returns output
func ExecuteCommand(args ...string) (string, error) {
	// Also set API key in viper since keyring won't work in tests
	viper.Set("api_key", "sk_live_test123")

	// Capture stdout since commands use fmt.Print directly
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	rootCmd.SetArgs(args)
	err := rootCmd.Execute()

	// Restore stdout and read captured output
	w.Close()
	os.Stdout = oldStdout
	out, _ := io.ReadAll(r)

	// Reset args for next test
	rootCmd.SetArgs([]string{})

	return string(out), err
}

// ExecuteCommandWithStdin runs a command with simulated stdin
func ExecuteCommandWithStdin(stdin string, args ...string) (string, error) {
	viper.Set("api_key", "sk_live_test123")

	// Save and replace stdin
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	w.WriteString(stdin)
	w.Close()
	os.Stdin = r

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(args)

	err := rootCmd.Execute()

	// Restore stdin
	os.Stdin = oldStdin

	// Reset args for next test
	rootCmd.SetArgs([]string{})

	return buf.String(), err
}

// CaptureOutput captures stdout during function execution
func CaptureOutput(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	out, _ := io.ReadAll(r)
	return string(out)
}

// AssertContains checks if output contains expected string
func AssertContains(t *testing.T, output, expected string) {
	t.Helper()
	if !strings.Contains(output, expected) {
		t.Errorf("Output missing expected string %q\nGot: %s", expected, output)
	}
}

// AssertNotContains checks if output does NOT contain unexpected string
func AssertNotContains(t *testing.T, output, unexpected string) {
	t.Helper()
	if strings.Contains(output, unexpected) {
		t.Errorf("Output unexpectedly contains %q\nGot: %s", unexpected, output)
	}
}

// JSON helpers for test responses

type ProductionResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Topic     string `json:"topic"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type ProductionsListResponse struct {
	Productions []ProductionResponse `json:"productions"`
	NextCursor  string               `json:"nextCursor,omitempty"`
	HasMore     bool                 `json:"hasMore"`
}

type AssetResponse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	MimeType   string `json:"mimeType"`
	SizeBytes  int64  `json:"sizeBytes"`
	UploadedAt string `json:"uploadedAt"`
}

type AssetsListResponse struct {
	Assets     []AssetResponse `json:"assets"`
	NextCursor string          `json:"nextCursor,omitempty"`
	HasMore    bool            `json:"hasMore"`
}

type KeyResponse struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	KeyPrefix string   `json:"keyPrefix"`
	Scopes    []string `json:"scopes"`
	CreatedAt string   `json:"createdAt"`
}

type KeysListResponse struct {
	Keys []KeyResponse `json:"keys"`
}

type ThreadMessageResponse struct {
	ID        string `json:"id"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
}

type ThreadResponse struct {
	UserMessage      ThreadMessageResponse `json:"userMessage"`
	AssistantMessage ThreadMessageResponse `json:"assistantMessage"`
}

type ThreadHistoryResponse struct {
	Messages []ThreadMessageResponse `json:"messages"`
}
