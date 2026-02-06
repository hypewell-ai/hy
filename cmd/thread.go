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
	Short: "Interactive AI chat",
	Long: `Start an interactive chat session with the Hypewell AI assistant.

For production-specific context, use --production flag.

Examples:
  hy thread
  hy thread --production prod_abc123
  hy thread send "How do I improve my hook?"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey := GetAPIKey()
		if apiKey == "" {
			return fmt.Errorf("not authenticated. Run 'hy auth login' first")
		}

		workspaceID := GetWorkspaceID()
		productionID, _ := cmd.Flags().GetString("production")

		fmt.Println("Hypewell AI Assistant")
		fmt.Println("Type 'exit' or Ctrl+C to quit")
		fmt.Println()

		reader := bufio.NewReader(os.Stdin)

		for {
			fmt.Print("You: ")
			input, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					fmt.Println()
					return nil
				}
				return err
			}

			input = strings.TrimSpace(input)
			if input == "" {
				continue
			}
			if input == "exit" || input == "quit" {
				return nil
			}

			// Send message
			response, err := sendThreadMessage(apiKey, workspaceID, productionID, input)
			if err != nil {
				fmt.Printf("Error: %v\n\n", err)
				continue
			}

			fmt.Printf("\nAssistant: %s\n\n", response)
		}
	},
}

var threadSendCmd = &cobra.Command{
	Use:   "send [message]",
	Short: "Send a single message",
	Long: `Send a single message and get a response.

Examples:
  hy thread send "What makes a good hook?"
  hy thread send --production prod_abc123 "Improve my script"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		message := args[0]

		apiKey := GetAPIKey()
		if apiKey == "" {
			return fmt.Errorf("not authenticated. Run 'hy auth login' first")
		}

		workspaceID := GetWorkspaceID()
		productionID, _ := cmd.Flags().GetString("production")

		response, err := sendThreadMessage(apiKey, workspaceID, productionID, message)
		if err != nil {
			return err
		}

		fmt.Println(response)
		return nil
	},
}

func sendThreadMessage(apiKey, workspaceID, productionID, message string) (string, error) {
	var url string
	if productionID != "" {
		url = fmt.Sprintf("%s/workspaces/%s/productions/%s/thread", GetAPIURL(), workspaceID, productionID)
	} else {
		url = fmt.Sprintf("%s/workspaces/%s/thread", GetAPIURL(), workspaceID)
	}

	payload := map[string]string{"message": message}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
	req.Header.Set("Authorization", apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		AssistantMessage struct {
			Content string `json:"content"`
		} `json:"assistantMessage"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return result.AssistantMessage.Content, nil
}

func init() {
	rootCmd.AddCommand(threadCmd)
	threadCmd.AddCommand(threadSendCmd)

	threadCmd.Flags().String("production", "", "Production ID for context")
	threadSendCmd.Flags().String("production", "", "Production ID for context")
}
