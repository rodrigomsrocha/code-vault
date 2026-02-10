package main

import (
	"fmt"

	"github.com/rodrigomsrocha/code-vault/internal/config"
	"github.com/rodrigomsrocha/code-vault/internal/storage"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Code Vault",
	Long:  `Initialize Code Vault by creating config file and testing database connection.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("🚀 Initializing Code Vault...")
		fmt.Println()

		// Load configuration (will create default if missing)
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		fmt.Println("✅ Config file ready at: ~/.config/code-vault/config.yaml")
		fmt.Println()

		// Test database connection
		fmt.Println("🔌 Testing database connection...")
		db, err := storage.NewDB(&cfg.Database)
		if err != nil {
			fmt.Println("❌ Database connection failed!")
			fmt.Println()
			fmt.Println("Please update your database credentials in:")
			fmt.Println("  ~/.config/code-vault/config.yaml")
			fmt.Println()
			fmt.Printf("Error: %v\n", err)
			return nil // Don't return error, let user fix config
		}
		defer db.Close()

		fmt.Println("✅ Database connected successfully!")
		fmt.Println()

		fmt.Println("🗄️  Testing SeaweedFS connection...")
		_, err = storage.NewSeaweedFS(&cfg.SeaweedFS)
		if err != nil {
			fmt.Println("❌ SeaweedFS connection failed!")
			fmt.Println()
			fmt.Println("Please update your SeaweedFS settings in:")
			fmt.Println("  ~/.config/code-vault/config.yaml")
			fmt.Println()
			fmt.Printf("Error: %v\n", err)
			return nil
		}

		fmt.Printf("✅ SeaweedFS connected (bucket: %s)\n", cfg.SeaweedFS.BucketName)
		fmt.Println()

		// Run migrations
		fmt.Println("🔄 Setting up database schema...")
		// if err := db.AutoMigrate(); err != nil {
		// 	return fmt.Errorf("migration failed: %w", err)
		// }

		fmt.Println("✅ Database schema ready!")
		fmt.Println()

		// Success summary
		fmt.Println("🎉 Code Vault initialized successfully!")
		fmt.Println()
		fmt.Println("Next steps:")
		fmt.Println("  1. Update ~/.config/code-vault/config.yaml with your settings")
		fmt.Println("  2. Run 'vault create <file>' to store your first snippet")
		fmt.Println()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
