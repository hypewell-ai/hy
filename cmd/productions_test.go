package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestProductionsList(t *testing.T) {
	tc := SetupTest(t)
	defer tc.Cleanup()

	tc.Server.HandleJSON("GET", "/workspaces/ws_test123/productions", http.StatusOK, ProductionsListResponse{
		Productions: []ProductionResponse{
			{
				ID:        "prod_abc123",
				Name:      "Test Production",
				Topic:     "Test topic",
				Status:    "draft",
				CreatedAt: "2026-02-06T12:00:00Z",
			},
			{
				ID:        "prod_def456",
				Name:      "Another Production",
				Topic:     "Another topic",
				Status:    "building",
				CreatedAt: "2026-02-06T11:00:00Z",
			},
		},
		HasMore: false,
	})

	output, err := ExecuteCommand("productions", "list")
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	AssertContains(t, output, "prod_abc123")
	AssertContains(t, output, "Test Production")
	AssertContains(t, output, "draft")
}

func TestProductionsGet(t *testing.T) {
	tc := SetupTest(t)
	defer tc.Cleanup()

	tc.Server.HandleJSON("GET", "/workspaces/ws_test123/productions/prod_abc123", http.StatusOK, ProductionResponse{
		ID:        "prod_abc123",
		Name:      "Test Production",
		Topic:     "Test topic for video",
		Status:    "draft",
		CreatedAt: "2026-02-06T12:00:00Z",
		UpdatedAt: "2026-02-06T12:00:00Z",
	})

	output, err := ExecuteCommand("productions", "get", "prod_abc123")
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	AssertContains(t, output, "prod_abc123")
	AssertContains(t, output, "Test topic for video")
}

func TestProductionsCreate(t *testing.T) {
	tc := SetupTest(t)
	defer tc.Cleanup()

	var receivedBody map[string]interface{}

	tc.Server.Handle("POST", "/workspaces/ws_test123/productions", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(ProductionResponse{
			ID:        "prod_new123",
			Name:      "New Video",
			Topic:     "Product launch",
			Status:    "draft",
			CreatedAt: "2026-02-06T12:00:00Z",
		})
	})

	output, err := ExecuteCommand("productions", "create", "--name", "New Video", "--topic", "Product launch")
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	// Verify request body
	if receivedBody["name"] != "New Video" {
		t.Errorf("Expected name 'New Video', got '%v'", receivedBody["name"])
	}
	if receivedBody["topic"] != "Product launch" {
		t.Errorf("Expected topic 'Product launch', got '%v'", receivedBody["topic"])
	}

	AssertContains(t, output, "prod_new123")
	AssertContains(t, output, "✓")
}

func TestProductionsBuild(t *testing.T) {
	tc := SetupTest(t)
	defer tc.Cleanup()

	// Build now does a GET first to validate spec exists
	tc.Server.HandleJSON("GET", "/workspaces/ws_test123/productions/prod_abc123", http.StatusOK, map[string]interface{}{
		"id":     "prod_abc123",
		"name":   "Test Production",
		"status": "draft",
		"spec":   map[string]interface{}{"version": "2.0"},
	})

	tc.Server.HandleJSON("POST", "/workspaces/ws_test123/productions/prod_abc123/build", http.StatusAccepted, map[string]interface{}{
		"id":      "prod_abc123",
		"status":  "building",
		"buildId": "build_xyz789",
		"message": "Build started. You will be notified when complete.",
	})

	output, err := ExecuteCommand("productions", "build", "prod_abc123")
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	AssertContains(t, output, "Build started")
	AssertContains(t, output, "build_xyz789")
}

func TestProductionsDeleteForce(t *testing.T) {
	tc := SetupTest(t)
	defer tc.Cleanup()

	tc.Server.HandleJSON("DELETE", "/workspaces/ws_test123/productions/prod_abc123", http.StatusOK, map[string]interface{}{
		"id":        "prod_abc123",
		"deleted":   true,
		"deletedAt": "2026-02-06T12:00:00Z",
	})

	output, err := ExecuteCommand("productions", "delete", "prod_abc123", "--force")
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	AssertContains(t, output, "Deleted production")
}

func TestProductionsStatus(t *testing.T) {
	tc := SetupTest(t)
	defer tc.Cleanup()

	tc.Server.HandleJSON("GET", "/workspaces/ws_test123/productions/prod_abc123/build", http.StatusOK, map[string]interface{}{
		"id":              "prod_abc123",
		"status":          "building",
		"buildId":         "build_xyz789",
		"buildLogUrl":     "https://console.cloud.google.com/build/...",
		"buildFinishedAt": nil,
		"outputUrl":       nil,
	})

	output, err := ExecuteCommand("productions", "status", "prod_abc123")
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	AssertContains(t, output, "building")
	AssertContains(t, output, "build_xyz789")
}

func TestProductionsListWithStatusFilter(t *testing.T) {
	tc := SetupTest(t)
	defer tc.Cleanup()

	var requestedURL string

	tc.Server.Handle("GET", "/workspaces/ws_test123/productions", func(w http.ResponseWriter, r *http.Request) {
		requestedURL = r.URL.String()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ProductionsListResponse{
			Productions: []ProductionResponse{},
			HasMore:     false,
		})
	})

	_, err := ExecuteCommand("productions", "list", "--status", "draft")
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	if !strings.Contains(requestedURL, "status=draft") {
		t.Errorf("Expected status filter in URL, got: %s", requestedURL)
	}
}

func TestProductionsCreateMissingRequired(t *testing.T) {
	tc := SetupTest(t)
	defer tc.Cleanup()

	_, err := ExecuteCommand("productions", "create", "--name", "Test")

	if err == nil {
		t.Error("Expected error for missing required field")
	}
}

func TestProductionsListEmpty(t *testing.T) {
	tc := SetupTest(t)
	defer tc.Cleanup()

	tc.Server.HandleJSON("GET", "/workspaces/ws_test123/productions", http.StatusOK, ProductionsListResponse{
		Productions: []ProductionResponse{},
		HasMore:     false,
	})

	output, err := ExecuteCommand("productions", "list")
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	AssertContains(t, output, "No productions found")
}

func TestProductionsBuildConflict(t *testing.T) {
	tc := SetupTest(t)
	defer tc.Cleanup()

	// Build now does a GET first
	tc.Server.HandleJSON("GET", "/workspaces/ws_test123/productions/prod_abc123", http.StatusOK, map[string]interface{}{
		"id":     "prod_abc123",
		"name":   "Test Production",
		"status": "building",
		"spec":   map[string]interface{}{"version": "2.0"},
	})

	tc.Server.HandleJSON("POST", "/workspaces/ws_test123/productions/prod_abc123/build", http.StatusConflict, map[string]interface{}{
		"error":  "Build already in progress",
		"status": "building",
	})

	_, err := ExecuteCommand("productions", "build", "prod_abc123")

	if err == nil {
		t.Error("Expected error for conflict")
	}
	if !strings.Contains(err.Error(), "already in progress") {
		t.Errorf("Error should mention conflict: %v", err)
	}
}

func TestProductionsGetNotFound(t *testing.T) {
	tc := SetupTest(t)
	defer tc.Cleanup()

	tc.Server.HandleJSON("GET", "/workspaces/ws_test123/productions/prod_nonexistent", http.StatusNotFound, map[string]string{
		"error": "Production not found",
	})

	_, err := ExecuteCommand("productions", "get", "prod_nonexistent")

	if err == nil {
		t.Error("Expected error for not found")
	}
}

func TestProductionsCreateDryRun(t *testing.T) {
	tc := SetupTest(t)
	defer tc.Cleanup()

	// No server handler needed - dry-run doesn't make API calls

	output, err := ExecuteCommand("productions", "create", "--name", "Test", "--topic", "Test topic", "--dry-run")
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	AssertContains(t, output, "[dry-run]")
	AssertContains(t, output, "Test")
	AssertContains(t, output, "Test topic")
}

func TestProductionsBuildValidateOnly(t *testing.T) {
	tc := SetupTest(t)
	defer tc.Cleanup()

	tc.Server.HandleJSON("GET", "/workspaces/ws_test123/productions/prod_abc123", http.StatusOK, map[string]interface{}{
		"id":     "prod_abc123",
		"name":   "Test Production",
		"status": "draft",
		"spec":   map[string]interface{}{"version": "2.0"},
	})

	// No POST handler needed - validate-only doesn't trigger build

	output, err := ExecuteCommand("productions", "build", "prod_abc123", "--validate-only")
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	AssertContains(t, output, "[validate-only]")
	AssertContains(t, output, "ready to build")
}

func TestProductionsBuildNoSpec(t *testing.T) {
	tc := SetupTest(t)
	defer tc.Cleanup()

	tc.Server.HandleJSON("GET", "/workspaces/ws_test123/productions/prod_abc123", http.StatusOK, map[string]interface{}{
		"id":     "prod_abc123",
		"name":   "Test Production",
		"status": "draft",
		"spec":   nil,
	})

	_, err := ExecuteCommand("productions", "build", "prod_abc123")

	if err == nil {
		t.Error("Expected error for missing spec")
	}
	if !strings.Contains(err.Error(), "no spec") {
		t.Errorf("Error should mention missing spec: %v", err)
	}
}
