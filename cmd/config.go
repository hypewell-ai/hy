package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
}

var configGetCmd = &cobra.Command{
	Use:   "get [key]",
	Short: "Get a configuration value",
	Long: `Get a configuration value.

Available keys:
  api_url       API base URL
  workspace_id  Current workspace ID

Examples:
  hy config get api_url
  hy config get workspace_id`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		value := viper.GetString(key)

		if value == "" {
			fmt.Printf("%s is not set\n", key)
		} else {
			fmt.Println(value)
		}

		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set [key] [value]",
	Short: "Set a configuration value",
	Long: `Set a configuration value.

Available keys:
  api_url       API base URL
  workspace_id  Current workspace ID

Examples:
  hy config set api_url https://studio.hypewell.ai/api
  hy config set workspace_id ws_abc123`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		value := args[1]

		viper.Set(key, value)

		if err := viper.WriteConfig(); err != nil {
			// Try to create the config file if it doesn't exist
			if err := viper.SafeWriteConfig(); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}
		}

		fmt.Printf("✓ Set %s = %s\n", key, value)
		return nil
	},
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configuration values",
	RunE: func(cmd *cobra.Command, args []string) error {
		settings := viper.AllSettings()

		if len(settings) == 0 {
			fmt.Println("No configuration set")
			return nil
		}

		for key, value := range settings {
			// Redact sensitive values
			if key == "api_key" {
				if v, ok := value.(string); ok && len(v) > 12 {
					value = v[:12] + "..."
				}
			}
			fmt.Printf("%s = %v\n", key, value)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configListCmd)
}
