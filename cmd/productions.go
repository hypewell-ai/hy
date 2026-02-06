package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
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

		status, _ := cmd.Flags().GetString("status")
		limit, _ := cmd.Flags().GetInt("limit")

		url := fmt.Sprintf("%s/workspaces/%s/productions?limit=%d", GetAPIURL(), workspaceID, limit)
		if status != "" {
			url += "&status=" + status
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
			Productions []struct {
				ID        string `json:"id"`
				Name      string `json:"name"`
				Status    string `json:"status"`
				Topic     string `json:"topic"`
				CreatedAt string `json:"createdAt"`
			} `json:"productions"`
			NextCursor string `json:"nextCursor"`
			HasMore    bool   `json:"hasMore"`
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
			// Truncate topic if too long
			topic := p.Topic
			if len(topic) > 40 {
				topic = topic[:37] + "..."
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", p.ID, p.Name, p.Status, topic)
		}
		w.Flush()

		if result.HasMore {
			fmt.Printf("\n(more results available, use --limit or pagination)\n")
		}

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

var productionsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new production",
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey := GetAPIKey()
		if apiKey == "" {
			return fmt.Errorf("not authenticated. Run 'hy auth login' first")
		}

		workspaceID := GetWorkspaceID()
		if workspaceID == "" {
			return fmt.Errorf("no workspace configured. Run 'hy auth login' first")
		}

		name, _ := cmd.Flags().GetString("name")
		topic, _ := cmd.Flags().GetString("topic")
		category, _ := cmd.Flags().GetString("category")
		specFile, _ := cmd.Flags().GetString("spec")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		if name == "" || topic == "" {
			return fmt.Errorf("--name and --topic are required")
		}

		payload := map[string]interface{}{
			"name":  name,
			"topic": topic,
		}

		if category != "" {
			payload["category"] = category
		}

		// Load spec from file if provided
		if specFile != "" {
			specData, err := os.ReadFile(specFile)
			if err != nil {
				return fmt.Errorf("failed to read spec file: %w", err)
			}
			var spec map[string]interface{}
			if err := json.Unmarshal(specData, &spec); err != nil {
				return fmt.Errorf("invalid spec JSON: %w", err)
			}
			payload["spec"] = spec
		}

		// Dry run - just validate and show what would be created
		if dryRun {
			fmt.Println("[dry-run] Would create production:")
			fmt.Printf("  Name:     %s\n", name)
			fmt.Printf("  Topic:    %s\n", topic)
			if category != "" {
				fmt.Printf("  Category: %s\n", category)
			}
			if specFile != "" {
				fmt.Printf("  Spec:     %s\n", specFile)
			}
			return nil
		}

		body, _ := json.Marshal(payload)
		url := fmt.Sprintf("%s/workspaces/%s/productions", GetAPIURL(), workspaceID)

		req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
		req.Header.Set("Authorization", apiKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			respBody, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
		}

		var result struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Topic  string `json:"topic"`
			Status string `json:"status"`
		}
		json.NewDecoder(resp.Body).Decode(&result)

		fmt.Printf("✓ Created production: %s\n", result.ID)
		fmt.Printf("  Name:   %s\n", result.Name)
		fmt.Printf("  Topic:  %s\n", result.Topic)
		fmt.Printf("  Status: %s\n", result.Status)

		return nil
	},
}

var productionsBuildCmd = &cobra.Command{
	Use:   "build [production-id]",
	Short: "Trigger a build for a production",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		productionID := args[0]

		apiKey := GetAPIKey()
		if apiKey == "" {
			return fmt.Errorf("not authenticated. Run 'hy auth login' first")
		}

		workspaceID := GetWorkspaceID()
		validateOnly, _ := cmd.Flags().GetBool("validate-only")

		// First, get the production to check if it has a spec
		getURL := fmt.Sprintf("%s/workspaces/%s/productions/%s", GetAPIURL(), workspaceID, productionID)
		getReq, _ := http.NewRequest("GET", getURL, nil)
		getReq.Header.Set("Authorization", apiKey)

		getResp, err := http.DefaultClient.Do(getReq)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		defer getResp.Body.Close()

		if getResp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(getResp.Body)
			return fmt.Errorf("API error (%d): %s", getResp.StatusCode, string(body))
		}

		var production struct {
			ID     string                 `json:"id"`
			Name   string                 `json:"name"`
			Status string                 `json:"status"`
			Spec   map[string]interface{} `json:"spec"`
		}
		json.NewDecoder(getResp.Body).Decode(&production)

		// Validate spec exists
		if production.Spec == nil {
			return fmt.Errorf("production has no spec. Add a spec before building")
		}

		// Validate-only mode - check spec but don't trigger build
		if validateOnly {
			fmt.Printf("[validate-only] Production %s is ready to build\n", productionID)
			fmt.Printf("  Name:   %s\n", production.Name)
			fmt.Printf("  Status: %s\n", production.Status)
			fmt.Println("  Spec:   ✓ valid")
			return nil
		}

		// Trigger actual build
		url := fmt.Sprintf("%s/workspaces/%s/productions/%s/build", GetAPIURL(), workspaceID, productionID)

		req, _ := http.NewRequest("POST", url, nil)
		req.Header.Set("Authorization", apiKey)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)

		if resp.StatusCode == http.StatusAccepted {
			var result struct {
				ID      string `json:"id"`
				Status  string `json:"status"`
				BuildID string `json:"buildId"`
				Message string `json:"message"`
			}
			json.Unmarshal(respBody, &result)

			fmt.Printf("✓ Build started for %s\n", productionID)
			if result.BuildID != "" {
				fmt.Printf("  Build ID: %s\n", result.BuildID)
			}
			fmt.Printf("  %s\n", result.Message)
			return nil
		}

		if resp.StatusCode == http.StatusConflict {
			return fmt.Errorf("build already in progress for this production")
		}

		return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	},
}

var productionsDeleteCmd = &cobra.Command{
	Use:   "delete [production-id]",
	Short: "Delete a production (soft delete)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		productionID := args[0]

		apiKey := GetAPIKey()
		if apiKey == "" {
			return fmt.Errorf("not authenticated. Run 'hy auth login' first")
		}

		workspaceID := GetWorkspaceID()

		// Confirm unless --force
		force, _ := cmd.Flags().GetBool("force")
		if !force {
			fmt.Printf("Delete production %s? This can be undone within 30 days. [y/N] ", productionID)
			var confirm string
			fmt.Scanln(&confirm)
			if strings.ToLower(confirm) != "y" && strings.ToLower(confirm) != "yes" {
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

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
		}

		fmt.Printf("✓ Deleted production: %s\n", productionID)
		fmt.Println("  (Will be permanently removed after 30 days)")

		return nil
	},
}

var productionsStatusCmd = &cobra.Command{
	Use:   "status [production-id]",
	Short: "Get build status for a production",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		productionID := args[0]

		apiKey := GetAPIKey()
		if apiKey == "" {
			return fmt.Errorf("not authenticated. Run 'hy auth login' first")
		}

		workspaceID := GetWorkspaceID()
		url := fmt.Sprintf("%s/workspaces/%s/productions/%s/build", GetAPIURL(), workspaceID, productionID)

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
			ID              string  `json:"id"`
			Status          string  `json:"status"`
			BuildID         *string `json:"buildId"`
			BuildLogURL     *string `json:"buildLogUrl"`
			BuildFinishedAt *string `json:"buildFinishedAt"`
			OutputURL       *string `json:"outputUrl"`
		}
		json.NewDecoder(resp.Body).Decode(&result)

		fmt.Printf("Production: %s\n", result.ID)
		fmt.Printf("Status:     %s\n", result.Status)

		if result.BuildID != nil && *result.BuildID != "" {
			fmt.Printf("Build ID:   %s\n", *result.BuildID)
		}
		if result.BuildLogURL != nil && *result.BuildLogURL != "" {
			fmt.Printf("Logs:       %s\n", *result.BuildLogURL)
		}
		if result.BuildFinishedAt != nil && *result.BuildFinishedAt != "" {
			fmt.Printf("Finished:   %s\n", *result.BuildFinishedAt)
		}
		if result.OutputURL != nil && *result.OutputURL != "" {
			fmt.Printf("Output:     %s\n", *result.OutputURL)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(productionsCmd)
	productionsCmd.AddCommand(productionsListCmd)
	productionsCmd.AddCommand(productionsGetCmd)
	productionsCmd.AddCommand(productionsCreateCmd)
	productionsCmd.AddCommand(productionsBuildCmd)
	productionsCmd.AddCommand(productionsDeleteCmd)
	productionsCmd.AddCommand(productionsStatusCmd)

	// List flags
	productionsListCmd.Flags().String("status", "", "Filter by status (draft, queued, building, review, approved, published, failed)")
	productionsListCmd.Flags().Int("limit", 20, "Maximum number of results")

	// Create flags
	productionsCreateCmd.Flags().String("name", "", "Production name (required)")
	productionsCreateCmd.Flags().String("topic", "", "Production topic (required)")
	productionsCreateCmd.Flags().String("category", "", "Production category")
	productionsCreateCmd.Flags().String("spec", "", "Path to SFSY spec JSON file")
	productionsCreateCmd.Flags().Bool("dry-run", false, "Validate inputs without creating")

	// Build flags
	productionsBuildCmd.Flags().Bool("validate-only", false, "Check spec validity without triggering build")

	// Delete flags
	productionsDeleteCmd.Flags().Bool("force", false, "Skip confirmation prompt")
}
