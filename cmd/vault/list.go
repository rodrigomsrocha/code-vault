package main

import (
	"fmt"
	"strings"

	"github.com/rodrigomsrocha/code-vault/internal/config"
	"github.com/rodrigomsrocha/code-vault/internal/models"
	"github.com/rodrigomsrocha/code-vault/internal/storage"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all your snippets",
	Long:  `Display a list of all snippets you've created.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		// Load configuration
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Connect to database
		db, err := storage.NewDB(&cfg.Database)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		// Initialize repositories
		snippetRepo := storage.NewSnippetRepository(db.DB)
		userRepo := storage.NewUserRepository(db.DB)

		// Get current user
		user, err := getOrCreateDefaultUser(userRepo, cfg)
		if err != nil {
			return fmt.Errorf("failed to get user: %w", err)
		}

		// List user's snippets
		snippets, err := snippetRepo.ListUserSnippets(ctx, user.ID)
		if err != nil {
			return fmt.Errorf("failed to list snippets: %w", err)
		}

		if len(snippets) == 0 {
			fmt.Println("No snippets found. Create one with: vault create <file>")
			return nil
		}

		// Display snippets
		fmt.Printf("📚 Your Snippets (%d total)\n", len(snippets))
		fmt.Println("─────────────────────────────────────────────────────────────────────────")

		for _, snippet := range snippets {
			displaySnippetSummary(&snippet)
			fmt.Println("─────────────────────────────────────────────────────────────────────────")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}

// displaySnippetSummary shows a one-line summary of a snippet
func displaySnippetSummary(snippet *models.Snippet) {
	// ID (first 8 chars for brevity)
	shortID := snippet.ID.String()[:8]

	// Tags
	var tagNames []string
	for _, tag := range snippet.Tags {
		tagNames = append(tagNames, tag.Name)
	}

	// Size
	size := "N/A"
	if snippet.CurrentVersion != nil {
		size = fmt.Sprintf("%d bytes", snippet.CurrentVersion.SizeBytes)
	}

	// Public indicator
	visibility := "private"
	if snippet.IsPublic {
		visibility = "public"
	}

	fmt.Printf("ID: %s...  %s\n", shortID, snippet.Title)
	fmt.Printf("   Language: %-12s Size: %-12s Visibility: %s\n",
		snippet.Language, size, visibility)
	if len(tagNames) > 0 {
		fmt.Printf("   Tags: %s\n", strings.Join(tagNames, ", "))
	}
	fmt.Printf("   Created: %s\n", snippet.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("   Get: vault get %s\n", snippet.ID)
}
