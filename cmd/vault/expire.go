package main

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rodrigomsrocha/code-vault/internal/config"
	"github.com/rodrigomsrocha/code-vault/internal/storage"
	"github.com/spf13/cobra"
)

var (
	expireSet    string
	expireRemove bool
	expireExtend string
)

var expireCmd = &cobra.Command{
	Use:   "expire <snippet-id>",
	Short: "Manage snippet expiration",
	Long:  `Set, remove, or extend the expiration time of a snippet.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

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

		// Get current snippet
		snippet, err := snippetRepo.GetSnippetByID(ctx, snippetID)
		if err != nil {
			return fmt.Errorf("snippet not found: %w", err)
		}

		// Remove expiration
		if expireRemove {
			if err := snippetRepo.UpdateExpiration(ctx, snippetID, nil); err != nil {
				return fmt.Errorf("failed to remove expiration: %w", err)
			}
			fmt.Println("✅ Expiration removed (snippet will not expire)")
			return nil
		}

		// Set new expiration
		if expireSet != "" {
			expiresAt, err := parseExpiration(expireSet)
			if err != nil {
				return fmt.Errorf("invalid expiration: %w", err)
			}

			if err := snippetRepo.UpdateExpiration(ctx, snippetID, &expiresAt); err != nil {
				return fmt.Errorf("failed to set expiration: %w", err)
			}

			fmt.Printf("✅ Expiration set to: %s\n", expiresAt.Format("2006-01-02 15:04:05"))
			return nil
		}

		// Extend expiration
		if expireExtend != "" {
			if snippet.ExpiresAt == nil {
				return fmt.Errorf("snippet has no expiration set (use --set instead)")
			}

			duration, err := time.ParseDuration(expireExtend)
			if err != nil {
				// Try parsing as days
				if expireExtend[len(expireExtend)-1] == 'd' {
					days := expireExtend[:len(expireExtend)-1]
					var d int
					_, err = fmt.Sscanf(days, "%d", &d)
					if err != nil {
						return fmt.Errorf("invalid duration format")
					}
					duration = time.Duration(d) * 24 * time.Hour
				} else {
					return fmt.Errorf("invalid duration: %w", err)
				}
			}

			newExpiry := snippet.ExpiresAt.Add(duration)
			if err := snippetRepo.UpdateExpiration(ctx, snippetID, &newExpiry); err != nil {
				return fmt.Errorf("failed to extend expiration: %w", err)
			}

			fmt.Printf("✅ Expiration extended to: %s\n", newExpiry.Format("2006-01-02 15:04:05"))
			return nil
		}

		// Show current expiration
		if snippet.ExpiresAt != nil {
			fmt.Printf("📅 Current expiration: %s\n", snippet.ExpiresAt.Format("2006-01-02 15:04:05"))
			timeUntil := time.Until(*snippet.ExpiresAt)
			if timeUntil > 0 {
				fmt.Printf("⏰ Expires in: %s\n", formatDuration(timeUntil))
			} else {
				fmt.Println("⚠️  Already expired")
			}
		} else {
			fmt.Println("📅 No expiration set (snippet will not expire)")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(expireCmd)

	expireCmd.Flags().StringVar(&expireSet, "set", "", "Set expiration (e.g., 24h, 7d, 30d)")
	expireCmd.Flags().BoolVar(&expireRemove, "remove", false, "Remove expiration")
	expireCmd.Flags().StringVar(&expireExtend, "extend", "", "Extend expiration by duration (e.g., 3d, 12h)")
}
