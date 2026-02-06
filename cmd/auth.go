package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/zalando/go-keyring"
)

const (
	serviceName = "hypewell-studio"
	keyringUser = "api-key"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with Hypewell Studio",
	Long: `Authenticate with an API key.

You can create an API key at https://studio.hypewell.ai/settings/api-keys
or using 'hy keys create' if you're already authenticated.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		reader := bufio.NewReader(os.Stdin)

		// Get API key
		fmt.Print("Enter API key (sk_live_...): ")
		apiKey, _ := reader.ReadString('\n')
		apiKey = strings.TrimSpace(apiKey)

		if !strings.HasPrefix(apiKey, "sk_live_") {
			return fmt.Errorf("invalid API key format (should start with sk_live_)")
		}

		// Get workspace ID
		fmt.Print("Enter workspace ID (ws_...): ")
		workspaceID, _ := reader.ReadString('\n')
		workspaceID = strings.TrimSpace(workspaceID)

		if !strings.HasPrefix(workspaceID, "ws_") {
			return fmt.Errorf("invalid workspace ID format (should start with ws_)")
		}

		// Store API key in keyring
		if err := keyring.Set(serviceName, keyringUser, apiKey); err != nil {
			// Fallback: store in config file (less secure)
			fmt.Println("Warning: Could not store in system keyring, storing in config file")
			viper.Set("api_key", apiKey)
		}

		// Store workspace ID in config
		viper.Set("workspace_id", workspaceID)
		if err := viper.WriteConfig(); err != nil {
			// Create config file if it doesn't exist
			if err := viper.SafeWriteConfig(); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}
		}

		fmt.Println("✓ Authenticated successfully")
		return nil
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear stored credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Clear from keyring
		keyring.Delete(serviceName, keyringUser)

		// Clear from config
		viper.Set("api_key", "")
		viper.Set("workspace_id", "")
		viper.WriteConfig()

		fmt.Println("✓ Logged out")
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current auth status",
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey := GetAPIKey()
		workspaceID := GetWorkspaceID()

		if apiKey == "" {
			fmt.Println("Not authenticated")
			fmt.Println("Run 'hy auth login' to authenticate")
			return nil
		}

		// Show redacted key
		redacted := apiKey[:12] + "..." + apiKey[len(apiKey)-4:]
		fmt.Printf("API Key: %s\n", redacted)
		fmt.Printf("Workspace: %s\n", workspaceID)
		fmt.Printf("API URL: %s\n", GetAPIURL())

		return nil
	},
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authStatusCmd)
}

// GetAPIKey returns the stored API key
func GetAPIKey() string {
	// Check environment first
	if key := os.Getenv("HY_API_KEY"); key != "" {
		return key
	}

	// Check keyring
	if key, err := keyring.Get(serviceName, keyringUser); err == nil {
		return key
	}

	// Fallback to config
	return viper.GetString("api_key")
}
