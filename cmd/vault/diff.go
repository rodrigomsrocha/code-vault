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
	diffFrom int
	diffTo   int
	diffRaw  bool
)

var diffCmd = &cobra.Command{
	Use:   "diff <snippet-id>",
	Short: "Compare two versions of a snippet",
	Long: `Show the differences between two versions of a snippet.
Use --from and --to flags to specify versions (default: compares v1 to latest).`,
	Args: cobra.ExactArgs(1),
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

		sfs, err := storage.NewSeaweedFS(&cfg.SeaweedFS)
		if err != nil {
			return fmt.Errorf("failed to connect to SeaweedFS: %w", err)
		}

		snippetRepo := storage.NewSnippetRepository(db.DB)
		snippetService := service.NewSnippetService(snippetRepo, sfs)

		snippet, err := snippetRepo.GetSnippetByID(ctx, snippetID)
		if err != nil {
			return fmt.Errorf("snippet not found: %w", err)
		}

		if diffFrom == 0 {
			diffFrom = 1
		}

		if diffTo == 0 {
			if snippet.CurrentVersion != nil {
				diffTo = snippet.CurrentVersion.VersionNumber
			} else {
				return fmt.Errorf("snippet has no versions")
			}
		}

		if diffFrom >= diffTo {
			return fmt.Errorf("--from must be less than --to (got: from=%d, to=%d)", diffFrom, diffTo)
		}

		fmt.Println("🔍 Comparing versions...")
		fmt.Println()

		diff, err := snippetService.DiffVersions(ctx, snippetID, diffFrom, diffTo)
		if err != nil {
			return fmt.Errorf("failed to generate diff: %w", err)
		}

		fmt.Println(diff)

		fromLines, toLines, err := snippetService.DiffStats(ctx, snippetID, diffFrom, diffTo)
		if err != nil {
			return fmt.Errorf("failed to get diff stats: %w", err)
		}

		fmt.Println()
		fmt.Println("─────────────────────────────────────────────────────────────────────────")
		fmt.Printf("Lines changed: %d → %d ", fromLines, toLines)

		if toLines > fromLines {
			fmt.Printf("(+%d lines)\n", toLines-fromLines)
		} else if toLines < fromLines {
			fmt.Printf("(-%d lines)\n", fromLines-toLines)
		} else {
			fmt.Println("(no line count change)")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(diffCmd)

	diffCmd.Flags().IntVar(&diffFrom, "from", 0, "Starting version number (default: 1)")
	diffCmd.Flags().IntVar(&diffTo, "to", 0, "Ending version number (default: latest)")
	diffCmd.Flags().BoolVar(&diffRaw, "raw", false, "Show raw diff without colors")
}
