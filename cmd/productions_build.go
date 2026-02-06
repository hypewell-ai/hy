package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"
)

var productionsBuildCmd = &cobra.Command{
	Use:   "build [production-id]",
	Short: "Trigger a build for a production",
	Long: `Queue a production for building.

The build will run asynchronously. Use 'hy productions get' to check status.

Examples:
  hy productions build prod_abc123`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		productionID := args[0]

		apiKey := GetAPIKey()
		if apiKey == "" {
			return fmt.Errorf("not authenticated. Run 'hy auth login' first")
		}

		workspaceID := GetWorkspaceID()
		if workspaceID == "" {
			return fmt.Errorf("no workspace configured. Run 'hy auth login' first")
		}

		url := fmt.Sprintf("%s/workspaces/%s/productions/%s/build", GetAPIURL(), workspaceID, productionID)

		req, _ := http.NewRequest("POST", url, nil)
		req.Header.Set("Authorization", apiKey)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
			return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
		}

		var result map[string]interface{}
		json.Unmarshal(body, &result)

		fmt.Println("✓ Build queued")
		if buildID, ok := result["buildId"].(string); ok {
			fmt.Printf("Build ID: %s\n", buildID)
		}
		if status, ok := result["status"].(string); ok {
			fmt.Printf("Status: %s\n", status)
		}

		fmt.Println("\nUse 'hy productions get " + productionID + "' to check build status")

		return nil
	},
}

func init() {
	productionsCmd.AddCommand(productionsBuildCmd)
}
