package cmd

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var productionsDeleteCmd = &cobra.Command{
	Use:   "delete [production-id]",
	Short: "Delete a production",
	Long: `Delete a production (soft delete).

The production will be moved to trash and permanently deleted after 30 days.

Examples:
  hy productions delete prod_abc123
  hy productions delete prod_abc123 --force`,
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

		force, _ := cmd.Flags().GetBool("force")

		if !force {
			fmt.Printf("Are you sure you want to delete production %s? [y/N] ", productionID)
			reader := bufio.NewReader(os.Stdin)
			confirm, _ := reader.ReadString('\n')
			confirm = strings.TrimSpace(strings.ToLower(confirm))
			if confirm != "y" && confirm != "yes" {
				fmt.Println("Cancelled")
				return nil
			}
		}

		url := fmt.Sprintf("%s/workspaces/%s/productions/%s", GetAPIURL(), workspaceID, productionID)

		req, _ := http.NewRequest("DELETE", url, nil)
		req.Header.Set("Authorization", apiKey)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
		}

		fmt.Printf("✓ Production %s deleted\n", productionID)
		fmt.Println("It will be permanently removed after 30 days.")

		return nil
	},
}

func init() {
	productionsCmd.AddCommand(productionsDeleteCmd)
	productionsDeleteCmd.Flags().Bool("force", false, "Skip confirmation prompt")
}
