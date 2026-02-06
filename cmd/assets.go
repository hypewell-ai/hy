package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
	Short:   "Manage assets (video, image, audio, font)",
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
		if workspaceID == "" {
			return fmt.Errorf("no workspace configured. Run 'hy auth login' first")
		}

		assetType, _ := cmd.Flags().GetString("type")
		limit, _ := cmd.Flags().GetInt("limit")

		url := fmt.Sprintf("%s/workspaces/%s/assets?limit=%d", GetAPIURL(), workspaceID, limit)
		if assetType != "" {
			url += "&type=" + assetType
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
			Assets []struct {
				ID         string `json:"id"`
				Name       string `json:"name"`
				Type       string `json:"type"`
				MimeType   string `json:"mimeType"`
				SizeBytes  int64  `json:"sizeBytes"`
				UploadedAt string `json:"uploadedAt"`
			} `json:"assets"`
			NextCursor string `json:"nextCursor"`
			HasMore    bool   `json:"hasMore"`
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
			size := formatBytes(a.SizeBytes)
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", a.ID, a.Name, a.Type, size)
		}
		w.Flush()

		if result.HasMore {
			fmt.Printf("\n(more results available)\n")
		}

		return nil
	},
}

var assetsUploadCmd = &cobra.Command{
	Use:   "upload [file-path]",
	Short: "Upload an asset",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]

		apiKey := GetAPIKey()
		if apiKey == "" {
			return fmt.Errorf("not authenticated. Run 'hy auth login' first")
		}

		workspaceID := GetWorkspaceID()
		if workspaceID == "" {
			return fmt.Errorf("no workspace configured. Run 'hy auth login' first")
		}

		// Read file info
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			return fmt.Errorf("cannot access file: %w", err)
		}

		fileName := filepath.Base(filePath)
		mimeType := detectMimeType(filePath)
		assetType := detectAssetType(mimeType)

		// Allow override
		if t, _ := cmd.Flags().GetString("type"); t != "" {
			assetType = t
		}
		if n, _ := cmd.Flags().GetString("name"); n != "" {
			fileName = n
		}

		fmt.Printf("Uploading %s (%s, %s)...\n", fileName, assetType, formatBytes(fileInfo.Size()))

		// Step 1: Create asset and get upload URL
		payload := map[string]interface{}{
			"name":      fileName,
			"type":      assetType,
			"mimeType":  mimeType,
			"sizeBytes": fileInfo.Size(),
		}
		body, _ := json.Marshal(payload)

		url := fmt.Sprintf("%s/workspaces/%s/assets", GetAPIURL(), workspaceID)
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

		var createResult struct {
			ID        string `json:"id"`
			UploadURL string `json:"uploadUrl"`
		}
		json.NewDecoder(resp.Body).Decode(&createResult)

		// Step 2: Upload file to signed URL
		file, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("cannot open file: %w", err)
		}
		defer file.Close()

		uploadReq, _ := http.NewRequest("PUT", createResult.UploadURL, file)
		uploadReq.Header.Set("Content-Type", mimeType)
		uploadReq.ContentLength = fileInfo.Size()

		uploadResp, err := http.DefaultClient.Do(uploadReq)
		if err != nil {
			return fmt.Errorf("upload failed: %w", err)
		}
		defer uploadResp.Body.Close()

		if uploadResp.StatusCode != http.StatusOK && uploadResp.StatusCode != http.StatusCreated {
			respBody, _ := io.ReadAll(uploadResp.Body)
			return fmt.Errorf("upload failed (%d): %s", uploadResp.StatusCode, string(respBody))
		}

		fmt.Printf("✓ Uploaded: %s\n", createResult.ID)
		return nil
	},
}

var assetsGetCmd = &cobra.Command{
	Use:   "get [asset-id]",
	Short: "Get asset details and download URL",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		assetID := args[0]

		apiKey := GetAPIKey()
		if apiKey == "" {
			return fmt.Errorf("not authenticated. Run 'hy auth login' first")
		}

		workspaceID := GetWorkspaceID()
		url := fmt.Sprintf("%s/workspaces/%s/assets/%s", GetAPIURL(), workspaceID, assetID)

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
			ID          string  `json:"id"`
			Name        string  `json:"name"`
			Type        string  `json:"type"`
			MimeType    string  `json:"mimeType"`
			SizeBytes   int64   `json:"sizeBytes"`
			DownloadURL *string `json:"downloadUrl"`
			UploadedAt  string  `json:"uploadedAt"`
		}
		json.NewDecoder(resp.Body).Decode(&result)

		fmt.Printf("ID:       %s\n", result.ID)
		fmt.Printf("Name:     %s\n", result.Name)
		fmt.Printf("Type:     %s\n", result.Type)
		fmt.Printf("MIME:     %s\n", result.MimeType)
		fmt.Printf("Size:     %s\n", formatBytes(result.SizeBytes))
		fmt.Printf("Uploaded: %s\n", result.UploadedAt)
		if result.DownloadURL != nil && *result.DownloadURL != "" {
			fmt.Printf("\nDownload URL (expires in 1 hour):\n%s\n", *result.DownloadURL)
		}

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
			fmt.Printf("Delete asset %s? [y/N] ", assetID)
			var confirm string
			fmt.Scanln(&confirm)
			if strings.ToLower(confirm) != "y" && strings.ToLower(confirm) != "yes" {
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

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
		}

		fmt.Printf("✓ Deleted asset: %s\n", assetID)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(assetsCmd)
	assetsCmd.AddCommand(assetsListCmd)
	assetsCmd.AddCommand(assetsUploadCmd)
	assetsCmd.AddCommand(assetsGetCmd)
	assetsCmd.AddCommand(assetsDeleteCmd)

	// List flags
	assetsListCmd.Flags().String("type", "", "Filter by type (video, image, audio, font)")
	assetsListCmd.Flags().Int("limit", 20, "Maximum number of results")

	// Upload flags
	assetsUploadCmd.Flags().String("type", "", "Override asset type")
	assetsUploadCmd.Flags().String("name", "", "Override file name")

	// Delete flags
	assetsDeleteCmd.Flags().Bool("force", false, "Skip confirmation prompt")
}

// Helper functions

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

func detectMimeType(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	mimeTypes := map[string]string{
		".mp4":   "video/mp4",
		".mov":   "video/quicktime",
		".avi":   "video/x-msvideo",
		".webm":  "video/webm",
		".jpg":   "image/jpeg",
		".jpeg":  "image/jpeg",
		".png":   "image/png",
		".gif":   "image/gif",
		".webp":  "image/webp",
		".mp3":   "audio/mpeg",
		".wav":   "audio/wav",
		".m4a":   "audio/mp4",
		".aac":   "audio/aac",
		".ttf":   "font/ttf",
		".otf":   "font/otf",
		".woff":  "font/woff",
		".woff2": "font/woff2",
	}
	if mime, ok := mimeTypes[ext]; ok {
		return mime
	}
	return "application/octet-stream"
}

func detectAssetType(mimeType string) string {
	if strings.HasPrefix(mimeType, "video/") {
		return "video"
	}
	if strings.HasPrefix(mimeType, "image/") {
		return "image"
	}
	if strings.HasPrefix(mimeType, "audio/") {
		return "audio"
	}
	if strings.HasPrefix(mimeType, "font/") {
		return "font"
	}
	return "video" // default
}
