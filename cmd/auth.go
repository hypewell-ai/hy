package cmd

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

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
	Long: `Authenticate via browser login.

This will open your browser to sign in with Hypewell Studio.
An API key will be created automatically and stored securely.`,
	RunE: runAuthLogin,
}

func runAuthLogin(cmd *cobra.Command, args []string) error {
	// Find an available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("failed to start local server: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	// Channel to receive the result
	resultCh := make(chan authResult, 1)

	// Start local HTTP server
	server := &http.Server{
		Addr: fmt.Sprintf("127.0.0.1:%d", port),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/callback" {
				http.NotFound(w, r)
				return
			}

			apiKey := r.URL.Query().Get("key")
			workspaceID := r.URL.Query().Get("workspace")
			errorMsg := r.URL.Query().Get("error")

			if errorMsg != "" {
				resultCh <- authResult{err: fmt.Errorf(errorMsg)}
				w.Header().Set("Content-Type", "text/html")
				fmt.Fprintf(w, `<html><body><h1>❌ Authentication failed</h1><p>%s</p><p>You can close this window.</p></body></html>`, errorMsg)
				return
			}

			if apiKey == "" || workspaceID == "" {
				resultCh <- authResult{err: fmt.Errorf("missing credentials in callback")}
				w.Header().Set("Content-Type", "text/html")
				fmt.Fprint(w, `<html><body><h1>❌ Authentication failed</h1><p>Missing credentials.</p><p>You can close this window.</p></body></html>`)
				return
			}

			resultCh <- authResult{apiKey: apiKey, workspaceID: workspaceID}
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<html><body><h1>✅ Authenticated!</h1><p>You can close this window and return to the terminal.</p><script>window.close()</script></body></html>`)
		}),
	}

	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			resultCh <- authResult{err: err}
		}
	}()

	// Build the auth URL
	apiURL := GetAPIURL()
	authURL := fmt.Sprintf("%s/../cli/auth?port=%d", strings.TrimSuffix(apiURL, "/api"), port)

	fmt.Println("Opening browser to authenticate...")
	fmt.Printf("If browser doesn't open, visit: %s\n\n", authURL)

	// Open browser
	if err := openBrowser(authURL); err != nil {
		fmt.Printf("Could not open browser: %v\n", err)
	}

	fmt.Println("Waiting for authentication...")

	// Wait for result with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	select {
	case result := <-resultCh:
		server.Shutdown(context.Background())

		if result.err != nil {
			return fmt.Errorf("authentication failed: %w", result.err)
		}

		// Store API key in keyring
		if err := keyring.Set(serviceName, keyringUser, result.apiKey); err != nil {
			// Fallback: store in config file (less secure)
			fmt.Println("Warning: Could not store in system keyring, storing in config file")
			viper.Set("api_key", result.apiKey)
		}

		// Store workspace ID in config
		viper.Set("workspace_id", result.workspaceID)
		if err := viper.WriteConfig(); err != nil {
			if err := viper.SafeWriteConfig(); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}
		}

		fmt.Println("\n✓ Authenticated successfully")
		fmt.Printf("  Workspace: %s\n", result.workspaceID)
		return nil

	case <-ctx.Done():
		server.Shutdown(context.Background())
		return fmt.Errorf("authentication timed out")
	}
}

type authResult struct {
	apiKey      string
	workspaceID string
	err         error
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform")
	}
	return cmd.Start()
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
