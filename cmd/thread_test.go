package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestThreadChat(t *testing.T) {
	tc := SetupTest(t)
	defer tc.Cleanup()

	var receivedBody map[string]interface{}

	tc.Server.Handle("POST", "/workspaces/ws_test123/thread", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ThreadResponse{
			UserMessage: ThreadMessageResponse{
				ID:        "msg_user123",
				Role:      "user",
				Content:   receivedBody["message"].(string),
				CreatedAt: "2026-02-06T12:00:00Z",
			},
			AssistantMessage: ThreadMessageResponse{
				ID:        "msg_asst123",
				Role:      "assistant",
				Content:   "Here's my response to your question about video structure...",
				CreatedAt: "2026-02-06T12:00:01Z",
			},
		})
	})

	output, err := ExecuteCommand("thread", "chat", "How should I structure my video?")
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	if receivedBody["message"] != "How should I structure my video?" {
		t.Errorf("Expected message not sent: %v", receivedBody["message"])
	}

	AssertContains(t, output, "Here's my response")
}

func TestThreadChatWithProduction(t *testing.T) {
	tc := SetupTest(t)
	defer tc.Cleanup()

	var requestPath string

	tc.Server.Handle("POST", "/workspaces/ws_test123/productions/prod_abc123/thread", func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ThreadResponse{
			UserMessage: ThreadMessageResponse{
				ID:        "msg_user123",
				Role:      "user",
				Content:   "Make the hook better",
				CreatedAt: "2026-02-06T12:00:00Z",
			},
			AssistantMessage: ThreadMessageResponse{
				ID:        "msg_asst123",
				Role:      "assistant",
				Content:   "I've improved the hook. Here's the updated version...",
				CreatedAt: "2026-02-06T12:00:01Z",
			},
		})
	})

	output, err := ExecuteCommand("thread", "chat", "-p", "prod_abc123", "Make the hook better")
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	if !strings.Contains(requestPath, "prod_abc123") {
		t.Errorf("Expected production-specific endpoint, got: %s", requestPath)
	}

	AssertContains(t, output, "improved the hook")
}

func TestThreadChatWithSuggestedChanges(t *testing.T) {
	tc := SetupTest(t)
	defer tc.Cleanup()

	tc.Server.Handle("POST", "/workspaces/ws_test123/productions/prod_abc123/thread", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"userMessage": ThreadMessageResponse{
				ID:        "msg_user123",
				Role:      "user",
				Content:   "Make the hook better",
				CreatedAt: "2026-02-06T12:00:00Z",
			},
			"assistantMessage": ThreadMessageResponse{
				ID:        "msg_asst123",
				Role:      "assistant",
				Content:   "I've updated the hook to be more engaging.",
				CreatedAt: "2026-02-06T12:00:01Z",
			},
			"suggestedChanges": []map[string]string{
				{
					"type":        "update_section",
					"description": "Updated hook script with stronger opening",
				},
			},
		})
	})

	output, err := ExecuteCommand("thread", "chat", "-p", "prod_abc123", "Make the hook better")
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	AssertContains(t, output, "Suggested changes")
	AssertContains(t, output, "Updated hook script")
}

func TestThreadHistory(t *testing.T) {
	tc := SetupTest(t)
	defer tc.Cleanup()

	tc.Server.HandleJSON("GET", "/workspaces/ws_test123/thread", http.StatusOK, ThreadHistoryResponse{
		Messages: []ThreadMessageResponse{
			{
				ID:        "msg_1",
				Role:      "user",
				Content:   "Hello",
				CreatedAt: "2026-02-06T12:00:00Z",
			},
			{
				ID:        "msg_2",
				Role:      "assistant",
				Content:   "Hi! How can I help you today?",
				CreatedAt: "2026-02-06T12:00:01Z",
			},
			{
				ID:        "msg_3",
				Role:      "user",
				Content:   "What's the best video length?",
				CreatedAt: "2026-02-06T12:01:00Z",
			},
			{
				ID:        "msg_4",
				Role:      "assistant",
				Content:   "For short-form content, 30-60 seconds is ideal.",
				CreatedAt: "2026-02-06T12:01:01Z",
			},
		},
	})

	output, err := ExecuteCommand("thread", "history")
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	AssertContains(t, output, "You:")
	AssertContains(t, output, "AI:")
	AssertContains(t, output, "30-60 seconds")
}

func TestThreadHistoryEmpty(t *testing.T) {
	tc := SetupTest(t)
	defer tc.Cleanup()

	tc.Server.HandleJSON("GET", "/workspaces/ws_test123/thread", http.StatusOK, ThreadHistoryResponse{
		Messages: []ThreadMessageResponse{},
	})

	output, err := ExecuteCommand("thread", "history")
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	AssertContains(t, output, "No messages")
}

func TestThreadHistoryWithLimit(t *testing.T) {
	tc := SetupTest(t)
	defer tc.Cleanup()

	var requestedURL string

	tc.Server.Handle("GET", "/workspaces/ws_test123/thread", func(w http.ResponseWriter, r *http.Request) {
		requestedURL = r.URL.String()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ThreadHistoryResponse{
			Messages: []ThreadMessageResponse{},
		})
	})

	_, err := ExecuteCommand("thread", "history", "--limit", "10")
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	if !strings.Contains(requestedURL, "limit=10") {
		t.Errorf("Expected limit in URL, got: %s", requestedURL)
	}
}
