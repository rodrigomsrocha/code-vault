package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	"github.com/rodrigomsrocha/code-vault/internal/config"
	"github.com/rodrigomsrocha/code-vault/internal/storage"
	"github.com/rodrigomsrocha/code-vault/internal/ui"
	"github.com/spf13/cobra"
)

var browseCmd = &cobra.Command{
	Use:   "browse",
	Short: "Browse snippets interactively",
	Long:  `Open an interactive browser to view and select snippets.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

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
		userRepo := storage.NewUserRepository(db.DB)

		user, err := getOrCreateDefaultUser(userRepo, cfg)
		if err != nil {
			return fmt.Errorf("failed to get user: %w", err)
		}

		snippets, err := snippetRepo.ListUserSnippets(ctx, user.ID)
		if err != nil {
			return fmt.Errorf("failed to list snippets: %w", err)
		}

		if len(snippets) == 0 {
			fmt.Println("No snippets found. Create one with: vault create <file>")
			return nil
		}

		// Content loader
		contentLoader := func(snippetID string) (string, error) {
			id, err := uuid.Parse(snippetID)
			if err != nil {
				return "", err
			}

			snippet, err := snippetRepo.GetSnippetByID(ctx, id)
			if err != nil {
				return "", err
			}

			if snippet.CurrentVersion == nil {
				return "", fmt.Errorf("no content")
			}

			content, err := sfs.Download(ctx, snippet.CurrentVersion.SeaweedFSFID)
			if err != nil {
				return "", err
			}

			return string(content), nil
		}

		p := tea.NewProgram(
			ui.NewListModel(snippets, contentLoader),
			tea.WithAltScreen(),
		)
		_, err = p.Run()
		if err != nil {
			return fmt.Errorf("failed to run browser: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(browseCmd)
}
