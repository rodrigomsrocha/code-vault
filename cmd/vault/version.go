package main

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rodrigomsrocha/code-vault/internal/config"
	"github.com/rodrigomsrocha/code-vault/internal/storage"
	"github.com/spf13/cobra"
)

var versionsCmd = &cobra.Command{
	Use:   "versions <snippet-id>",
	Short: "List all versions of a snippet",
	Long:  `Display the complete version history of a snippet.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		snippetID, err := uuid.Parse(args[0])
		if err != nil {
			return fmt.Errorf("invalid snippet ID: %w", err)
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		db, err := storage.NewDB(&cfg.Database)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		defer db.Close()

		snippetRepo := storage.NewSnippetRepository(db.DB)

		snippet, err := snippetRepo.GetSnippetByID(cmd.Context(), snippetID)
		if err != nil {
			return fmt.Errorf("snippet not found: %w", err)
		}

		versions, err := snippetRepo.ListSnippetVersions(snippetID)
		if err != nil {
			return fmt.Errorf("failed to list versions: %w", err)
		}

		if len(versions) == 0 {
			fmt.Println("No versions found.")
			return nil
		}

		fmt.Printf("📜 Version History: %s\n", snippet.Title)
		fmt.Println("─────────────────────────────────────────────────────────────────────────")
		fmt.Printf("Total versions: %d\n", len(versions))
		fmt.Println()

		for _, version := range versions {
			isCurrent := snippet.CurrentVersionID != nil && *snippet.CurrentVersionID == version.ID

			marker := "  "
			if isCurrent {
				marker = "→ "
			}

			fmt.Printf("%sVersion %d", marker, version.VersionNumber)
			if isCurrent {
				fmt.Print(" (current)")
			}
			fmt.Println()

			fmt.Printf("  Created:  %s (%s ago)\n",
				version.CreatedAt.Format("2006-01-02 15:04:05"),
				formatDuration(time.Since(version.CreatedAt)))

			fmt.Printf("  Size:     %d bytes\n", version.SizeBytes)
			fmt.Printf("  Hash:     %s...\n", version.ContentHash[:16])
			fmt.Printf("  Get:      vault get %s --version %d\n", snippetID, version.VersionNumber)
			fmt.Println()
		}

		fmt.Println("─────────────────────────────────────────────────────────────────────────")
		fmt.Printf("Compare versions: vault diff %s --from 1 --to %d\n", snippetID, versions[0].VersionNumber)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionsCmd)
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		minutes := int(d.Minutes())
		if minutes == 1 {
			return "1 minute"
		}
		return fmt.Sprintf("%d minutes", minutes)
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		if hours == 1 {
			return "1 hour"
		}
		return fmt.Sprintf("%d hours", hours)
	}
	days := int(d.Hours() / 24)
	if days == 1 {
		return "1 day"
	}
	if days < 30 {
		return fmt.Sprintf("%d days", days)
	}
	months := days / 30
	if months == 1 {
		return "1 month"
	}
	if months < 12 {
		return fmt.Sprintf("%d months", months)
	}
	years := months / 12
	if years == 1 {
		return "1 year"
	}
	return fmt.Sprintf("%d years", years)
}
