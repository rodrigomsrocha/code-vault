package main

import (
	"fmt"
	"os"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	"github.com/rodrigomsrocha/code-vault/internal/config"
	"github.com/rodrigomsrocha/code-vault/internal/models"
	"github.com/rodrigomsrocha/code-vault/internal/service"
	"github.com/rodrigomsrocha/code-vault/internal/storage"
	"github.com/rodrigomsrocha/code-vault/internal/ui"
	"github.com/spf13/cobra"
)

var (
	getOutput    string
	getRaw       bool
	getVersion   int
	getCopy      bool
	getHighlight bool
)

var getCmd = &cobra.Command{
	Use:   "get <snippet-id>",
	Short: "Retrieve a snippet by ID",
	Long: `Download and display a snippet from code Vault by its ID.
If no ID is provided, opens an interactive selector.`,
	Args: cobra.MaximumNArgs(1),
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
		snippetService := service.NewSnippetService(snippetRepo, sfs)

		var snippetID uuid.UUID
		if len(args) == 0 {
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
			m, err := p.Run()
			if err != nil {
				return fmt.Errorf("failed to run selector: %w", err)
			}

			listModel := m.(ui.ListModel)
			choice := listModel.GetChoice()
			if choice == nil {
				fmt.Println("Cancelled.")
				return nil
			}

			snippetID = choice.ID
		} else {
			snippetID, err = uuid.Parse(args[0])
			if err != nil {
				return fmt.Errorf("Invalid snippet ID: %w", err)
			}
		}

		snippet, err := snippetRepo.GetSnippetByID(ctx, snippetID)
		if err != nil {
			return fmt.Errorf("snippet not found: %w", err)
		}

		var content []byte
		var version *models.SnippetVersion

		if getVersion > 0 {
			versionData, versionModel, err := snippetService.GetSnippetVersion(ctx, snippetID, getVersion)
			if err != nil {
				return fmt.Errorf("failed to get snippet version: %w", err)
			}
			content = versionData
			version = versionModel
		} else {
			if snippet.CurrentVersion == nil {
				return fmt.Errorf("snippet has no content")
			}

			content, err = sfs.Download(ctx, snippet.CurrentVersion.SeaweedFSFID)
			if err != nil {
				return fmt.Errorf("failed to download content: %w", err)
			}

			version = snippet.CurrentVersion
		}

		if getCopy {
			if err := clipboard.WriteAll(string(content)); err != nil {
				fmt.Println("⚠️  Failed to copy to clipboard:", err)
			} else {
				fmt.Println("✅ Copied to clipboard!")
			}
		}

		if getRaw {
			fmt.Print(string(content))
			return nil
		}

		if getOutput != "" {
			if err := os.WriteFile(getOutput, content, 0644); err != nil {
				return fmt.Errorf("failed to write to file: %w", err)
			}
			fmt.Printf("✅ Snippet saved to: %s\n", getOutput)
			return nil
		}

		displaySnippet(snippet, content, version, getHighlight)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(getCmd)

	getCmd.Flags().StringVarP(&getOutput, "output", "o", "", "Save content to file")
	getCmd.Flags().BoolVar(&getRaw, "raw", false, "Show only raw content (no metadata)")
	getCmd.Flags().IntVarP(&getVersion, "version", "v", 0, "Get a specific version of the snippet")
	getCmd.Flags().BoolVarP(&getCopy, "copy", "c", false, "Copy content to clipboard")
	getCmd.Flags().BoolVar(&getHighlight, "highlight", true, "Enable syntax highlighting")
}

func displaySnippet(snippet *models.Snippet, content []byte, version *models.SnippetVersion, highlight bool) {
	fmt.Println("📄 Snippet Details")
	fmt.Println("─────────────────────────────────────────────────────────")
	fmt.Printf("ID:          %s\n", snippet.ID)
	fmt.Printf("Title:       %s\n", snippet.Title)
	if snippet.Description != "" {
		fmt.Printf("Description: %s\n", snippet.Description)
	}
	fmt.Printf("Language:    %s\n", snippet.Language)
	fmt.Printf("Public:      %v\n", snippet.IsPublic)

	if len(snippet.Tags) > 0 {
		fmt.Print("Tags:        ")
		for i, tag := range snippet.Tags {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Print(tag.Name)
		}
		fmt.Println()
	}

	if snippet.CurrentVersion != nil {
		fmt.Printf("Version:     %d\n", version.VersionNumber)
		fmt.Printf("Size:        %d bytes\n", version.SizeBytes)
		fmt.Printf("Hash:        %s\n", version.ContentHash[:16]+"...")
	}

	fmt.Printf("Created:     %s\n", snippet.CreatedAt.Format("2006-01-02 15:04:05"))

	if snippet.ExpiresAt != nil {
		fmt.Printf("Expires:     %s\n", snippet.ExpiresAt.Format("2006-01-02 15:04:05"))
	}

	fmt.Println("─────────────────────────────────────────────────────────")
	fmt.Println()
	fmt.Println("📝 Content:")
	fmt.Println()

	if highlight && snippet.Language != "" {
		highlighted, err := ui.HighlightCode(string(content), snippet.Language)
		if err == nil {
			fmt.Println(highlighted)
		} else {
			fmt.Println(string(content))
		}
	} else {
		fmt.Println(string(content))
	}
}
