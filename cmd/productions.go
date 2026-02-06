package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var productionsCmd = &cobra.Command{
	Use:     "productions",
	Aliases: []string{"prod", "p"},
	Short:   "Manage productions",
}

var productionsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all productions",
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey := GetAPIKey()
		if apiKey == "" {
			return fmt.Errorf("not authenticated. Run 'hy auth login' first")
		}

		workspaceID := GetWorkspaceID()
		if workspaceID == "" {
			return fmt.Errorf("no workspace configured. Run 'hy auth login' first")
		}

		url := fmt.Sprintf("%s/workspaces/%s/productions", GetAPIURL(), workspaceID)

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
			Productions []struct {
				ID        string `json:"id"`
				Name      string `json:"name"`
				Status    string `json:"status"`
				Topic     string `json:"topic"`
				CreatedAt string `json:"createdAt"`
			} `json:"productions"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if len(result.Productions) == 0 {
			fmt.Println("No productions found")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tSTATUS\tTOPIC")
		for _, p := range result.Productions {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", p.ID, p.Name, p.Status, p.Topic)
		}
		w.Flush()

		return nil
	},
}

var productionsGetCmd = &cobra.Command{
	Use:   "get [production-id]",
	Short: "Get production details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		productionID := args[0]

		apiKey := GetAPIKey()
		if apiKey == "" {
			return fmt.Errorf("not authenticated. Run 'hy auth login' first")
		}

		workspaceID := GetWorkspaceID()
		url := fmt.Sprintf("%s/workspaces/%s/productions/%s", GetAPIURL(), workspaceID, productionID)

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

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		// Pretty print
		output, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(output))

		return nil
	},
}

func init() {
	rootCmd.AddCommand(productionsCmd)
	productionsCmd.AddCommand(productionsListCmd)
	productionsCmd.AddCommand(productionsGetCmd)

	// TODO: Add create, build, delete commands
}
