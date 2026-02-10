package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rodrigomsrocha/code-vault/internal/config"
	"github.com/rodrigomsrocha/code-vault/internal/service"
	"github.com/rodrigomsrocha/code-vault/internal/storage"
	"github.com/spf13/cobra"
)

var (
	workerInterval string
	workerOnce     bool
)

var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Run expiration cleanup worker",
	Long: `Run a background worker that automatically deletes expired snippets.
The worker runs periodically and safely removes expired content.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		// Load config
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

		// Connect to SeaweedFS
		sfs, err := storage.NewSeaweedFS(&cfg.SeaweedFS)
		if err != nil {
			return fmt.Errorf("failed to connect to SeaweedFS: %w", err)
		}

		// Parse interval
		interval, err := time.ParseDuration(workerInterval)
		if err != nil {
			return fmt.Errorf("invalid interval: %w", err)
		}

		// Create worker
		snippetRepo := storage.NewSnippetRepository(db.DB)
		worker := service.NewWorker(snippetRepo, sfs, interval)

		// Run once mode
		if workerOnce {
			fmt.Println("🧹 Running cleanup once...")
			if err := worker.RunOnce(ctx); err != nil {
				return fmt.Errorf("cleanup failed: %w", err)
			}
			fmt.Println("✅ Cleanup complete")
			return nil
		}

		// Start worker
		if err := worker.Start(ctx); err != nil {
			return err
		}

		fmt.Printf("✅ Worker started (checking every %s)\n", interval)
		fmt.Println("Press Ctrl+C to stop...")

		// Wait for interrupt signal
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		<-sigChan
		fmt.Println("\n🛑 Shutting down...")

		worker.Stop()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(workerCmd)

	workerCmd.Flags().StringVar(&workerInterval, "interval", "1h", "Cleanup interval (e.g., 30m, 1h, 24h)")
	workerCmd.Flags().BoolVar(&workerOnce, "once", false, "Run cleanup once and exit")
}
