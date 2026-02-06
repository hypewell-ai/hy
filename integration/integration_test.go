// +build integration

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

var (
	apiKey      string
	workspaceID string
	apiURL      string
)

func init() {
	apiKey = os.Getenv("HY_TEST_API_KEY")
	workspaceID = os.Getenv("HY_TEST_WORKSPACE_ID")
	apiURL = os.Getenv("HY_TEST_API_URL")

	if apiURL == "" {
		apiURL = "https://studio.hypewell.ai/api"
	}
	if workspaceID == "" {
		workspaceID = "ws_integration_test"
	}
}

func skipIfNoCredentials(t *testing.T) {
	if apiKey == "" {
		t.Skip("HY_TEST_API_KEY not set, skipping integration test")
	}
}

func apiRequest(method, path string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		reqBody = bytes.NewReader(data)
	}

	url := fmt.Sprintf("%s%s", apiURL, path)
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return http.DefaultClient.Do(req)
}

// TestHealthCheck verifies the API is reachable
func TestHealthCheck(t *testing.T) {
	resp, err := http.Get(apiURL + "/health")
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
}

// TestProductionsLifecycle tests create, get, list, delete
func TestProductionsLifecycle(t *testing.T) {
	skipIfNoCredentials(t)

	// Generate unique name for this test run
	testName := fmt.Sprintf("Integration Test %d", time.Now().Unix())
	var productionID string

	// Create
	t.Run("Create", func(t *testing.T) {
		resp, err := apiRequest("POST", fmt.Sprintf("/workspaces/%s/productions", workspaceID), map[string]interface{}{
			"name":  testName,
			"topic": "Integration test topic",
		})
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("Expected 201, got %d: %s", resp.StatusCode, string(body))
		}

		var result struct {
			ID string `json:"id"`
		}
		json.NewDecoder(resp.Body).Decode(&result)
		productionID = result.ID

		if !strings.HasPrefix(productionID, "prod_") {
			t.Errorf("Expected prefixed ID, got %s", productionID)
		}
	})

	// Get
	t.Run("Get", func(t *testing.T) {
		if productionID == "" {
			t.Skip("No production ID from create")
		}

		resp, err := apiRequest("GET", fmt.Sprintf("/workspaces/%s/productions/%s", workspaceID, productionID), nil)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200, got %d", resp.StatusCode)
		}

		var result struct {
			Name string `json:"name"`
		}
		json.NewDecoder(resp.Body).Decode(&result)

		if result.Name != testName {
			t.Errorf("Expected name %q, got %q", testName, result.Name)
		}
	})

	// List
	t.Run("List", func(t *testing.T) {
		resp, err := apiRequest("GET", fmt.Sprintf("/workspaces/%s/productions", workspaceID), nil)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200, got %d", resp.StatusCode)
		}

		var result struct {
			Productions []struct {
				ID string `json:"id"`
			} `json:"productions"`
		}
		json.NewDecoder(resp.Body).Decode(&result)

		found := false
		for _, p := range result.Productions {
			if p.ID == productionID {
				found = true
				break
			}
		}
		if !found && productionID != "" {
			t.Error("Created production not found in list")
		}
	})

	// Delete (cleanup)
	t.Run("Delete", func(t *testing.T) {
		if productionID == "" {
			t.Skip("No production ID to delete")
		}

		resp, err := apiRequest("DELETE", fmt.Sprintf("/workspaces/%s/productions/%s", workspaceID, productionID), nil)
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected 200, got %d: %s", resp.StatusCode, string(body))
		}
	})

	// Verify deleted
	t.Run("VerifyDeleted", func(t *testing.T) {
		if productionID == "" {
			t.Skip("No production ID to verify")
		}

		resp, err := apiRequest("GET", fmt.Sprintf("/workspaces/%s/productions/%s", workspaceID, productionID), nil)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected 404 after delete, got %d", resp.StatusCode)
		}
	})
}

// TestKeysLifecycle tests create, list, revoke
func TestKeysLifecycle(t *testing.T) {
	skipIfNoCredentials(t)

	testName := fmt.Sprintf("Test Key %d", time.Now().Unix())
	var keyID string

	// Create - API keys cannot create other API keys (requires Firebase auth)
	t.Run("Create", func(t *testing.T) {
		resp, err := apiRequest("POST", fmt.Sprintf("/workspaces/%s/keys", workspaceID), map[string]interface{}{
			"name":   testName,
			"scopes": []string{"productions:read"},
		})
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		defer resp.Body.Close()

		// API keys cannot create other API keys - this is by design
		if resp.StatusCode != http.StatusForbidden {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("Expected 403 (API keys can't create keys), got %d: %s", resp.StatusCode, string(body))
		}

		// Skip the key validation since we didn't create one
		t.Log("Confirmed: API keys cannot create other API keys (security by design)")
	})

	// List
	t.Run("List", func(t *testing.T) {
		resp, err := apiRequest("GET", fmt.Sprintf("/workspaces/%s/keys", workspaceID), nil)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200, got %d", resp.StatusCode)
		}
	})

	// Revoke (cleanup)
	t.Run("Revoke", func(t *testing.T) {
		if keyID == "" {
			t.Skip("No key ID to revoke")
		}

		resp, err := apiRequest("DELETE", fmt.Sprintf("/workspaces/%s/keys/%s", workspaceID, keyID), nil)
		if err != nil {
			t.Fatalf("Revoke failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("Expected 200, got %d: %s", resp.StatusCode, string(body))
		}
	})
}

// TestThreadChat tests the AI chat endpoint
func TestThreadChat(t *testing.T) {
	skipIfNoCredentials(t)

	resp, err := apiRequest("POST", fmt.Sprintf("/workspaces/%s/thread", workspaceID), map[string]interface{}{
		"message": "Hello, this is an integration test. Reply with 'OK'.",
	})
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		AssistantMessage struct {
			Content string `json:"content"`
		} `json:"assistantMessage"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if result.AssistantMessage.Content == "" {
		t.Error("Expected non-empty assistant response")
	}
}

// TestRateLimiting verifies rate limit headers are present
func TestRateLimiting(t *testing.T) {
	skipIfNoCredentials(t)

	resp, err := apiRequest("GET", fmt.Sprintf("/workspaces/%s/productions", workspaceID), nil)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	// Check for rate limit headers
	headers := []string{"X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset"}
	for _, h := range headers {
		if resp.Header.Get(h) == "" {
			t.Errorf("Missing rate limit header: %s", h)
		}
	}
}
