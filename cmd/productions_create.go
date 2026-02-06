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

var productionsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new production",
	Long: `Create a new production interactively or from a JSON file.

Examples:
  hy productions create
  hy productions create --name "My Video" --topic "Product Launch"
  hy productions create --from spec.json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey := GetAPIKey()
		if apiKey == "" {
			return fmt.Errorf("not authenticated. Run 'hy auth login' first")
		}

		workspaceID := GetWorkspaceID()
		if workspaceID == "" {
			return fmt.Errorf("no workspace configured. Run 'hy auth login' first")
		}

		var name, topic, category string
		var err error

		// Check for --from file
		fromFile, _ := cmd.Flags().GetString("from")
		if fromFile != "" {
			return createFromFile(apiKey, workspaceID, fromFile)
		}

		// Get from flags or interactive
		name, _ = cmd.Flags().GetString("name")
		topic, _ = cmd.Flags().GetString("topic")
		category, _ = cmd.Flags().GetString("category")

		reader := bufio.NewReader(os.Stdin)

		if name == "" {
			fmt.Print("Production name: ")
			name, err = reader.ReadString('\n')
			if err != nil {
				return err
			}
			name = strings.TrimSpace(name)
		}

		if topic == "" {
			fmt.Print("Topic: ")
			topic, err = reader.ReadString('\n')
			if err != nil {
				return err
			}
			topic = strings.TrimSpace(topic)
		}

		if category == "" {
			fmt.Print("Category (optional): ")
			category, _ = reader.ReadString('\n')
			category = strings.TrimSpace(category)
		}

		// Create the production
		return createProduction(apiKey, workspaceID, map[string]interface{}{
			"name":     name,
			"topic":    topic,
			"category": category,
		})
	},
}

func createFromFile(apiKey, workspaceID, filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var spec map[string]interface{}
	if err := json.Unmarshal(data, &spec); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	return createProduction(apiKey, workspaceID, spec)
}

func createProduction(apiKey, workspaceID string, data map[string]interface{}) error {
	url := fmt.Sprintf("%s/workspaces/%s/productions", GetAPIURL(), workspaceID)

	body, _ := json.Marshal(data)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
	req.Header.Set("Authorization", apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	json.Unmarshal(respBody, &result)

	fmt.Println("✓ Production created")
	if id, ok := result["id"].(string); ok {
		fmt.Printf("ID: %s\n", id)
	}
	if name, ok := result["name"].(string); ok {
		fmt.Printf("Name: %s\n", name)
	}

	return nil
}

func init() {
	productionsCmd.AddCommand(productionsCreateCmd)

	productionsCreateCmd.Flags().String("name", "", "Production name")
	productionsCreateCmd.Flags().String("topic", "", "Production topic")
	productionsCreateCmd.Flags().String("category", "", "Production category")
	productionsCreateCmd.Flags().String("from", "", "Create from JSON file")
}
