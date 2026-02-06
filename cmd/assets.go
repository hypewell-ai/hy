package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var assetsCmd = &cobra.Command{
	Use:     "assets",
	Aliases: []string{"asset", "a"},
	Short:   "Manage assets",
}

var assetsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all assets",
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey := GetAPIKey()
		if apiKey == "" {
			return fmt.Errorf("not authenticated. Run 'hy auth login' first")
		}

		workspaceID := GetWorkspaceID()
		url := fmt.Sprintf("%s/workspaces/%s/assets", GetAPIURL(), workspaceID)

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
			Assets []struct {
				ID       string `json:"id"`
				Name     string `json:"name"`
				Type     string `json:"type"`
				MimeType string `json:"mimeType"`
				Size     int64  `json:"sizeBytes"`
			} `json:"assets"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if len(result.Assets) == 0 {
			fmt.Println("No assets found")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tTYPE\tSIZE")
		for _, a := range result.Assets {
			size := formatBytes(a.Size)
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", a.ID, a.Name, a.Type, size)
		}
		w.Flush()

		return nil
	},
}

var assetsUploadCmd = &cobra.Command{
	Use:   "upload [file]",
	Short: "Upload an asset",
	Long: `Upload a file as an asset.

Supported types: video, image, audio, font

Examples:
  hy assets upload video.mp4
  hy assets upload --name "Hero Image" banner.png`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]

		apiKey := GetAPIKey()
		if apiKey == "" {
			return fmt.Errorf("not authenticated. Run 'hy auth login' first")
		}

		workspaceID := GetWorkspaceID()

		// Read the file
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}

		// Get file info
		fileName := filepath.Base(filePath)
		name, _ := cmd.Flags().GetString("name")
		if name == "" {
			name = fileName
		}

		// Detect MIME type
		ext := filepath.Ext(filePath)
		mimeType := mime.TypeByExtension(ext)
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}

		// Determine asset type from MIME
		assetType := "video"
		if strings.HasPrefix(mimeType, "image/") {
			assetType = "image"
		} else if strings.HasPrefix(mimeType, "audio/") {
			assetType = "audio"
		} else if strings.Contains(mimeType, "font") {
			assetType = "font"
		}

		// Create multipart request (simplified - actual impl would use multipart/form-data)
		url := fmt.Sprintf("%s/workspaces/%s/assets", GetAPIURL(), workspaceID)

		// For now, send as JSON with base64 (real impl would use signed URLs)
		payload := map[string]interface{}{
			"name":     name,
			"type":     assetType,
			"mimeType": mimeType,
			"fileName": fileName,
			"size":     len(data),
		}

		body, _ := json.Marshal(payload)
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

		var result struct {
			ID        string `json:"id"`
			UploadURL string `json:"uploadUrl"`
		}
		json.Unmarshal(respBody, &result)

		// If we got an upload URL, upload the file there
		if result.UploadURL != "" {
			uploadReq, _ := http.NewRequest("PUT", result.UploadURL, bytes.NewReader(data))
			uploadReq.Header.Set("Content-Type", mimeType)

			uploadResp, err := http.DefaultClient.Do(uploadReq)
			if err != nil {
				return fmt.Errorf("upload failed: %w", err)
			}
			uploadResp.Body.Close()

			if uploadResp.StatusCode != http.StatusOK {
				return fmt.Errorf("upload failed with status %d", uploadResp.StatusCode)
			}
		}

		fmt.Println("✓ Asset uploaded")
		fmt.Printf("ID: %s\n", result.ID)
		fmt.Printf("Name: %s\n", name)
		fmt.Printf("Size: %s\n", formatBytes(int64(len(data))))

		return nil
	},
}

var assetsDeleteCmd = &cobra.Command{
	Use:   "delete [asset-id]",
	Short: "Delete an asset",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		assetID := args[0]

		apiKey := GetAPIKey()
		if apiKey == "" {
			return fmt.Errorf("not authenticated. Run 'hy auth login' first")
		}

		workspaceID := GetWorkspaceID()

		force, _ := cmd.Flags().GetBool("force")
		if !force {
			fmt.Printf("Are you sure you want to delete asset %s? [y/N] ", assetID)
			reader := bufio.NewReader(os.Stdin)
			confirm, _ := reader.ReadString('\n')
			confirm = strings.TrimSpace(strings.ToLower(confirm))
			if confirm != "y" && confirm != "yes" {
				fmt.Println("Cancelled")
				return nil
			}
		}

		url := fmt.Sprintf("%s/workspaces/%s/assets/%s", GetAPIURL(), workspaceID, assetID)

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

		fmt.Printf("✓ Asset %s deleted\n", assetID)
		return nil
	},
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func init() {
	rootCmd.AddCommand(assetsCmd)
	assetsCmd.AddCommand(assetsListCmd)
	assetsCmd.AddCommand(assetsUploadCmd)
	assetsCmd.AddCommand(assetsDeleteCmd)

	assetsUploadCmd.Flags().String("name", "", "Asset name (defaults to filename)")
	assetsDeleteCmd.Flags().Bool("force", false, "Skip confirmation prompt")
}
