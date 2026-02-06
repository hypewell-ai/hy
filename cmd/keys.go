package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var keysCmd = &cobra.Command{
	Use:   "keys",
	Short: "Manage API keys",
}

var keysListCmd = &cobra.Command{
	Use:   "list",
	Short: "List API keys",
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey := GetAPIKey()
		if apiKey == "" {
			return fmt.Errorf("not authenticated. Run 'hy auth login' first")
		}

		workspaceID := GetWorkspaceID()
		url := fmt.Sprintf("%s/workspaces/%s/keys", GetAPIURL(), workspaceID)

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
			Keys []struct {
				ID         string   `json:"id"`
				Name       string   `json:"name"`
				KeyPrefix  string   `json:"keyPrefix"`
				Scopes     []string `json:"scopes"`
				LastUsedAt string   `json:"lastUsedAt"`
			} `json:"keys"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if len(result.Keys) == 0 {
			fmt.Println("No API keys found")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tPREFIX\tLAST USED")
		for _, k := range result.Keys {
			lastUsed := k.LastUsedAt
			if lastUsed == "" {
				lastUsed = "never"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", k.ID, k.Name, k.KeyPrefix, lastUsed)
		}
		w.Flush()

		return nil
	},
}

var keysCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new API key",
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey := GetAPIKey()
		if apiKey == "" {
			return fmt.Errorf("not authenticated. Run 'hy auth login' first")
		}

		name, _ := cmd.Flags().GetString("name")
		scopes, _ := cmd.Flags().GetStringSlice("scopes")

		if name == "" {
			return fmt.Errorf("--name is required")
		}

		workspaceID := GetWorkspaceID()
		url := fmt.Sprintf("%s/workspaces/%s/keys", GetAPIURL(), workspaceID)

		body, _ := json.Marshal(map[string]interface{}{
			"name":   name,
			"scopes": scopes,
		})

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
			ID      string `json:"id"`
			Key     string `json:"key"`
			Name    string `json:"name"`
			Warning string `json:"warning"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		fmt.Println("✓ API key created")
		fmt.Println()
		fmt.Printf("ID:   %s\n", result.ID)
		fmt.Printf("Name: %s\n", result.Name)
		fmt.Printf("Key:  %s\n", result.Key)
		fmt.Println()
		fmt.Printf("⚠️  %s\n", result.Warning)

		return nil
	},
}

var keysRevokeCmd = &cobra.Command{
	Use:   "revoke [key-id]",
	Short: "Revoke an API key",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		keyID := args[0]

		apiKey := GetAPIKey()
		if apiKey == "" {
			return fmt.Errorf("not authenticated. Run 'hy auth login' first")
		}

		workspaceID := GetWorkspaceID()
		url := fmt.Sprintf("%s/workspaces/%s/keys/%s", GetAPIURL(), workspaceID, keyID)

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

		fmt.Printf("✓ API key %s revoked\n", keyID)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(keysCmd)
	keysCmd.AddCommand(keysListCmd)
	keysCmd.AddCommand(keysCreateCmd)
	keysCmd.AddCommand(keysRevokeCmd)

	keysCreateCmd.Flags().String("name", "", "Key name (required)")
	keysCreateCmd.Flags().StringSlice("scopes", []string{"productions:read", "assets:read"}, "Key scopes")
}
