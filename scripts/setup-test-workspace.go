//go:build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/firestore"
)

func main() {
	ctx := context.Background()

	client, err := firestore.NewClient(ctx, "hypewell-prod")
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	workspaceID := "ws_integration_test"
	now := time.Now()

	// Create test workspace
	_, err = client.Collection("workspaces").Doc(workspaceID).Set(ctx, map[string]interface{}{
		"id":        workspaceID,
		"name":      "Integration Tests",
		"slug":      "integration-test",
		"plan":      "free",
		"createdAt": now,
		"updatedAt": now,
	})
	if err != nil {
		log.Fatalf("Failed to create workspace: %v", err)
	}

	fmt.Printf("✓ Created test workspace: %s\n", workspaceID)
	fmt.Println("\nNext steps:")
	fmt.Println("1. Create a test API key:")
	fmt.Println("   hy keys create --name 'Integration Tests' --workspace " + workspaceID)
	fmt.Println("2. Save the key to ~/.config/hy/test-key")
	fmt.Println("3. Run integration tests:")
	fmt.Println("   HY_TEST_API_KEY=$(cat ~/.config/hy/test-key) go test ./integration/... -v -tags=integration")
}
