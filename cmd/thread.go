package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var threadCmd = &cobra.Command{
	Use:   "thread",
	Short: "Chat with AI assistant",
	Long: `Start a conversation with the AI assistant.

Use workspace-level thread for general questions, or specify a production
for context-aware assistance with your video script.`,
}

var threadChatCmd = &cobra.Command{
	Use:   "chat [message]",
	Short: "Send a message to the thread",
	Long: `Send a message and get a response from the AI assistant.

Examples:
  hy thread chat "How should I structure my video?"
  hy thread chat --production prod_xxx "Make the hook more engaging"
  hy thread chat  # Interactive mode`,
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey := GetAPIKey()
		if apiKey == "" {
			return fmt.Errorf("not authenticated. Run 'hy auth login' first")
		}

		workspaceID := GetWorkspaceID()
		if workspaceID == "" {
			return fmt.Errorf("no workspace configured. Run 'hy auth login' first")
		}

		productionID, _ := cmd.Flags().GetString("production")

		// Build URL
		var url string
		if productionID != "" {
			url = fmt.Sprintf("%s/workspaces/%s/productions/%s/thread", GetAPIURL(), workspaceID, productionID)
		} else {
			url = fmt.Sprintf("%s/workspaces/%s/thread", GetAPIURL(), workspaceID)
		}

		// Interactive mode if no message provided
		if len(args) == 0 {
			return interactiveChat(url, apiKey, productionID)
		}

		message := strings.Join(args, " ")
		return sendChatMessage(url, apiKey, message)
	},
}

var threadHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "View chat history",
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey := GetAPIKey()
		if apiKey == "" {
			return fmt.Errorf("not authenticated. Run 'hy auth login' first")
		}

		workspaceID := GetWorkspaceID()
		if workspaceID == "" {
			return fmt.Errorf("no workspace configured. Run 'hy auth login' first")
		}

		productionID, _ := cmd.Flags().GetString("production")
		limit, _ := cmd.Flags().GetInt("limit")

		var url string
		if productionID != "" {
			url = fmt.Sprintf("%s/workspaces/%s/productions/%s/thread?limit=%d", GetAPIURL(), workspaceID, productionID, limit)
		} else {
			url = fmt.Sprintf("%s/workspaces/%s/thread?limit=%d", GetAPIURL(), workspaceID, limit)
		}

		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("Authorization", apiKey)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
		}

		var result struct {
			Messages []struct {
				ID        string `json:"id"`
				Role      string `json:"role"`
				Content   string `json:"content"`
				CreatedAt string `json:"createdAt"`
			} `json:"messages"`
		}
		json.NewDecoder(resp.Body).Decode(&result)

		if len(result.Messages) == 0 {
			fmt.Println("No messages in thread")
			return nil
		}

		for _, msg := range result.Messages {
			prefix := "You:"
			if msg.Role == "assistant" {
				prefix = "AI:"
			} else if msg.Role == "system" {
				prefix = "System:"
			}
			fmt.Printf("\n%s\n%s\n", prefix, msg.Content)
		}

		return nil
	},
}

func sendChatMessage(url, apiKey, message string) error {
	payload := map[string]string{"message": message}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
	req.Header.Set("Authorization", apiKey)
	req.Header.Set("Content-Type", "application/json")

	fmt.Println("Thinking...")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		AssistantMessage struct {
			Content string `json:"content"`
		} `json:"assistantMessage"`
		SuggestedChanges []struct {
			Type        string `json:"type"`
			Description string `json:"description"`
		} `json:"suggestedChanges"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	fmt.Printf("\n%s\n", result.AssistantMessage.Content)

	if len(result.SuggestedChanges) > 0 {
		fmt.Println("\n📝 Suggested changes:")
		for _, change := range result.SuggestedChanges {
			fmt.Printf("  • %s\n", change.Description)
		}
	}

	return nil
}

func interactiveChat(url, apiKey, productionID string) error {
	if productionID != "" {
		fmt.Printf("Chatting with production: %s\n", productionID)
	} else {
		fmt.Println("Chatting with workspace assistant")
	}
	fmt.Println("Type 'exit' or 'quit' to end the conversation.")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("You: ")
		if !scanner.Scan() {
			break
		}

		message := strings.TrimSpace(scanner.Text())
		if message == "" {
			continue
		}

		if message == "exit" || message == "quit" {
			fmt.Println("Goodbye!")
			break
		}

		if err := sendChatMessage(url, apiKey, message); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
		fmt.Println()
	}

	return nil
}

func init() {
	rootCmd.AddCommand(threadCmd)
	threadCmd.AddCommand(threadChatCmd)
	threadCmd.AddCommand(threadHistoryCmd)

	// Chat flags
	threadChatCmd.Flags().StringP("production", "p", "", "Production ID for context-aware chat")

	// History flags
	threadHistoryCmd.Flags().StringP("production", "p", "", "Production ID")
	threadHistoryCmd.Flags().Int("limit", 20, "Number of messages to fetch")
}
