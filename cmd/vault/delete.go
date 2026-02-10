package main

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/rodrigomsrocha/code-vault/internal/config"
	"github.com/rodrigomsrocha/code-vault/internal/storage"
	"github.com/spf13/cobra"
)

var hard bool

var deleteCmd = &cobra.Command{
	Use:   "delete <snippet-id>",
	Short: "Delete a secret by ID",
	Long:  "Delete a secret from the vault by its ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		snippetID, err := uuid.Parse(args[0])
		if err != nil {
			return fmt.Errorf("Invalid snippet ID: %w", err)
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

		sfs, err := storage.NewSeaweedFS(&cfg.SeaweedFS)
		if err != nil {
			return fmt.Errorf("failed to connect to SeaweedFS: %w", err)
		}

		snippetRepo := storage.NewSnippetRepository(db.DB)

		if hard {
			snippet, err := snippetRepo.GetSnippetByID(ctx, snippetID)
			if err != nil {
				return fmt.Errorf("failed to get snippet: %w", err)
			}
			sfs.Delete(ctx, snippet.CurrentVersion.ContentHash)
		}

		err = snippetRepo.DeleteSnippet(ctx, snippetID)
		if err != nil {
			return fmt.Errorf("failed to delete snippet: %w", err)
		}

		fmt.Printf("🗑 Snippet was deleted from the Vault\n")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)

	deleteCmd.Flags().BoolVar(&hard, "hard", false, "Delete snippet from storage")
}
