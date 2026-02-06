package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	version = "dev"
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "hy",
	Short: "Hypewell Studio CLI",
	Long: `hy is the command-line interface for Hypewell Studio.

Create, manage, and build video productions from your terminal.`,
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

// SetVersion sets the version string
func SetVersion(v string) {
	version = v
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/hy/config.yaml)")
	rootCmd.PersistentFlags().String("api-url", "", "API base URL")
	rootCmd.PersistentFlags().String("workspace", "", "Workspace ID")

	viper.BindPFlag("api_url", rootCmd.PersistentFlags().Lookup("api-url"))
	viper.BindPFlag("workspace_id", rootCmd.PersistentFlags().Lookup("workspace"))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		configDir := home + "/.config/hy"
		os.MkdirAll(configDir, 0755)

		viper.AddConfigPath(configDir)
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	// Environment variables
	viper.SetEnvPrefix("HY")
	viper.AutomaticEnv()

	// Defaults
	viper.SetDefault("api_url", "https://studio.hypewell.ai/api")

	if err := viper.ReadInConfig(); err == nil {
		// Config file found and loaded
	}
}

// GetAPIURL returns the configured API URL
func GetAPIURL() string {
	return viper.GetString("api_url")
}

// GetWorkspaceID returns the configured workspace ID
func GetWorkspaceID() string {
	return viper.GetString("workspace_id")
}
