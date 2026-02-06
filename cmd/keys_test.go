package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestKeysList(t *testing.T) {
	tc := SetupTest(t)
	defer tc.Cleanup()

	tc.Server.HandleJSON("GET", "/workspaces/ws_test123/keys", http.StatusOK, KeysListResponse{
		Keys: []KeyResponse{
			{
				ID:        "key_abc123",
				Name:      "Production Key",
				KeyPrefix: "sk_live_xxxx",
				Scopes:    []string{"productions:read", "productions:write"},
				CreatedAt: "2026-02-06T12:00:00Z",
			},
			{
				ID:        "key_def456",
				Name:      "Read Only",
				KeyPrefix: "sk_live_yyyy",
				Scopes:    []string{"productions:read"},
				CreatedAt: "2026-02-06T11:00:00Z",
			},
		},
	})

	output, err := ExecuteCommand("keys", "list")
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	AssertContains(t, output, "key_abc123")
	AssertContains(t, output, "Production Key")
	AssertContains(t, output, "sk_live_xxxx")
}

func TestKeysCreate(t *testing.T) {
	tc := SetupTest(t)
	defer tc.Cleanup()

	var receivedBody map[string]interface{}

	tc.Server.Handle("POST", "/workspaces/ws_test123/keys", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":        "key_new123",
			"key":       "sk_live_full_key_value_here",
			"name":      receivedBody["name"],
			"keyPrefix": "sk_live_full",
			"scopes":    receivedBody["scopes"],
			"createdAt": "2026-02-06T12:00:00Z",
			"warning":   "Save this key now. You won't be able to see it again.",
		})
	})

	output, err := ExecuteCommand("keys", "create", "--name", "CI Key", "--scopes", "productions:read,productions:write")
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	if receivedBody["name"] != "CI Key" {
		t.Errorf("Expected name 'CI Key', got '%v'", receivedBody["name"])
	}

	AssertContains(t, output, "sk_live_full_key_value_here")
	AssertContains(t, output, "Save this key")
}

func TestKeysRevokeForce(t *testing.T) {
	tc := SetupTest(t)
	defer tc.Cleanup()

	tc.Server.HandleJSON("DELETE", "/workspaces/ws_test123/keys/key_abc123", http.StatusOK, map[string]interface{}{
		"id":        "key_abc123",
		"revoked":   true,
		"revokedAt": "2026-02-06T12:00:00Z",
	})

	output, err := ExecuteCommand("keys", "revoke", "key_abc123", "--force")
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	AssertContains(t, output, "Revoked")
}

func TestKeysListEmpty(t *testing.T) {
	tc := SetupTest(t)
	defer tc.Cleanup()

	tc.Server.HandleJSON("GET", "/workspaces/ws_test123/keys", http.StatusOK, KeysListResponse{
		Keys: []KeyResponse{},
	})

	output, err := ExecuteCommand("keys", "list")
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	AssertContains(t, output, "No API keys found")
}

func TestKeysCreateMissingName(t *testing.T) {
	tc := SetupTest(t)
	defer tc.Cleanup()

	_, err := ExecuteCommand("keys", "create")
	if err == nil {
		t.Error("Expected error for missing name")
	}
}
