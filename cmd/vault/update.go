package main

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/rodrigomsrocha/code-vault/internal/config"
	"github.com/rodrigomsrocha/code-vault/internal/service"
	"github.com/rodrigomsrocha/code-vault/internal/storage"
	"github.com/spf13/cobra"
)

var (
	updateTags []string
)

var updateCmd = &cobra.Command{
	Use:   "update <snippet-id> <file>",
	Short: "Update a snippet with new content (creates new version)",
	Long: `Upload a new version of an existing snippet.
	This creates a new version while preserving the complete version history.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		snippetID, err := uuid.Parse(args[0])
		if err != nil {
			return fmt.Errorf("invalid snippet ID: %w", err)
		}

		filePath := args[1]

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		db, err := storage.NewDB(&cfg.Database)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		sfs, err := storage.NewSeaweedFS(&cfg.SeaweedFS)
		if err != nil {
			return fmt.Errorf("failed to connect to SeaweedFS: %w", err)
		}

		snippetRepo := storage.NewSnippetRepository(db.DB)
		snippetService := service.NewSnippetService(snippetRepo, sfs)

		fmt.Println("🔄 Updating snippet...")
		fmt.Printf("   Snippet ID: %s\n", snippetID)
		fmt.Printf("   New content: %s\n", filePath)
		fmt.Println()

		snippet, version, err := snippetService.UpdateSnippet(cmd.Context(), service.UpdateSnippetInput{
			SnippetID: snippetID,
			FilePath:  filePath,
			Tags:      updateTags,
		})

		if err != nil {
			return fmt.Errorf("failed to update snippet: %w", err)
		}

		// Success!
		fmt.Println()
		fmt.Println("✅ Snippet updated successfully!")
		fmt.Println()
		fmt.Printf("   Title: %s\n", snippet.Title)
		fmt.Printf("   New version: %d\n", version.VersionNumber)
		fmt.Printf("   Size: %d bytes\n", version.SizeBytes)
		fmt.Printf("   Hash: %s...\n", version.ContentHash[:16])
		fmt.Println()
		fmt.Printf("View all versions: vault versions %s\n", snippetID)
		fmt.Printf("Get this version:  vault get %s --version %d\n", snippetID, version.VersionNumber)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)

	updateCmd.Flags().StringSliceVar(&updateTags, "tags", []string{}, "Update tags (replaces existing tags)")
}
