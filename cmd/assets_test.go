package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAssetsList(t *testing.T) {
	tc := SetupTest(t)
	defer tc.Cleanup()

	tc.Server.HandleJSON("GET", "/workspaces/ws_test123/assets", http.StatusOK, AssetsListResponse{
		Assets: []AssetResponse{
			{
				ID:         "asset_abc123",
				Name:       "intro.mp4",
				Type:       "video",
				MimeType:   "video/mp4",
				SizeBytes:  10485760,
				UploadedAt: "2026-02-06T12:00:00Z",
			},
			{
				ID:         "asset_def456",
				Name:       "logo.png",
				Type:       "image",
				MimeType:   "image/png",
				SizeBytes:  102400,
				UploadedAt: "2026-02-06T11:00:00Z",
			},
		},
		HasMore: false,
	})

	output, err := ExecuteCommand("assets", "list")
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	AssertContains(t, output, "asset_abc123")
	AssertContains(t, output, "intro.mp4")
	AssertContains(t, output, "video")
}

func TestAssetsListWithTypeFilter(t *testing.T) {
	tc := SetupTest(t)
	defer tc.Cleanup()

	var requestedURL string

	tc.Server.Handle("GET", "/workspaces/ws_test123/assets", func(w http.ResponseWriter, r *http.Request) {
		requestedURL = r.URL.String()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(AssetsListResponse{
			Assets:  []AssetResponse{},
			HasMore: false,
		})
	})

	_, err := ExecuteCommand("assets", "list", "--type", "video")
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	if !strings.Contains(requestedURL, "type=video") {
		t.Errorf("Expected type filter in URL, got: %s", requestedURL)
	}
}

func TestAssetsGet(t *testing.T) {
	tc := SetupTest(t)
	defer tc.Cleanup()

	downloadURL := "https://storage.googleapis.com/signed-url..."
	tc.Server.HandleJSON("GET", "/workspaces/ws_test123/assets/asset_abc123", http.StatusOK, map[string]interface{}{
		"id":          "asset_abc123",
		"name":        "intro.mp4",
		"type":        "video",
		"mimeType":    "video/mp4",
		"sizeBytes":   10485760,
		"downloadUrl": downloadURL,
		"uploadedAt":  "2026-02-06T12:00:00Z",
	})

	output, err := ExecuteCommand("assets", "get", "asset_abc123")
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	AssertContains(t, output, "asset_abc123")
	AssertContains(t, output, "intro.mp4")
	AssertContains(t, output, downloadURL)
}

func TestAssetsUpload(t *testing.T) {
	tc := SetupTest(t)
	defer tc.Cleanup()

	// Create a test file
	testFile := filepath.Join(tc.ConfigDir, "test-video.mp4")
	testContent := []byte("fake video content for testing")
	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Track requests
	var createRequest map[string]interface{}
	uploadCalled := false

	// Mock create asset endpoint
	tc.Server.Handle("POST", "/workspaces/ws_test123/assets", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &createRequest)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":        "asset_new123",
			"uploadUrl": tc.Server.URL + "/upload/test",
		})
	})

	// Mock upload endpoint
	tc.Server.Handle("PUT", "/upload/test", func(w http.ResponseWriter, r *http.Request) {
		uploadCalled = true
		if r.Header.Get("Content-Type") != "video/mp4" {
			t.Errorf("Expected Content-Type video/mp4, got %s", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusOK)
	})

	output, err := ExecuteCommand("assets", "upload", testFile)
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	// Verify create request
	if createRequest["name"] != "test-video.mp4" {
		t.Errorf("Expected name 'test-video.mp4', got '%v'", createRequest["name"])
	}
	if createRequest["type"] != "video" {
		t.Errorf("Expected type 'video', got '%v'", createRequest["type"])
	}

	if !uploadCalled {
		t.Error("Upload endpoint was not called")
	}

	AssertContains(t, output, "Uploaded")
	AssertContains(t, output, "asset_new123")
}

func TestAssetsDelete(t *testing.T) {
	tc := SetupTest(t)
	defer tc.Cleanup()

	tc.Server.HandleJSON("DELETE", "/workspaces/ws_test123/assets/asset_abc123", http.StatusOK, map[string]interface{}{
		"id":        "asset_abc123",
		"deleted":   true,
		"deletedAt": "2026-02-06T12:00:00Z",
	})

	output, err := ExecuteCommand("assets", "delete", "asset_abc123", "--force")
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	AssertContains(t, output, "Deleted asset")
}

func TestAssetsListEmpty(t *testing.T) {
	tc := SetupTest(t)
	defer tc.Cleanup()

	tc.Server.HandleJSON("GET", "/workspaces/ws_test123/assets", http.StatusOK, AssetsListResponse{
		Assets:  []AssetResponse{},
		HasMore: false,
	})

	output, err := ExecuteCommand("assets", "list")
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	AssertContains(t, output, "No assets found")
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{10485760, "10.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		result := formatBytes(tt.bytes)
		if result != tt.expected {
			t.Errorf("formatBytes(%d) = %s, want %s", tt.bytes, result, tt.expected)
		}
	}
}

func TestDetectMimeType(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"video.mp4", "video/mp4"},
		{"video.MP4", "video/mp4"},
		{"image.jpg", "image/jpeg"},
		{"image.jpeg", "image/jpeg"},
		{"image.png", "image/png"},
		{"audio.mp3", "audio/mpeg"},
		{"font.ttf", "font/ttf"},
		{"unknown.xyz", "application/octet-stream"},
	}

	for _, tt := range tests {
		result := detectMimeType(tt.path)
		if result != tt.expected {
			t.Errorf("detectMimeType(%s) = %s, want %s", tt.path, result, tt.expected)
		}
	}
}

func TestDetectAssetType(t *testing.T) {
	tests := []struct {
		mimeType string
		expected string
	}{
		{"video/mp4", "video"},
		{"video/quicktime", "video"},
		{"image/jpeg", "image"},
		{"image/png", "image"},
		{"audio/mpeg", "audio"},
		{"audio/wav", "audio"},
		{"font/ttf", "font"},
		{"application/octet-stream", "video"}, // default
	}

	for _, tt := range tests {
		result := detectAssetType(tt.mimeType)
		if result != tt.expected {
			t.Errorf("detectAssetType(%s) = %s, want %s", tt.mimeType, result, tt.expected)
		}
	}
}
