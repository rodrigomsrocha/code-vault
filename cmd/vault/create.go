package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rodrigomsrocha/code-vault/internal/config"
	"github.com/rodrigomsrocha/code-vault/internal/models"
	"github.com/rodrigomsrocha/code-vault/internal/service"
	"github.com/rodrigomsrocha/code-vault/internal/storage"
	"github.com/spf13/cobra"
)

var (
	// Flags for create command
	createTitle       string
	createDescription string
	createLanguage    string
	createTags        []string
	createPublic      bool
	createExpires     string
)

var createCmd = &cobra.Command{
	Use:   "create <file>",
	Short: "Create a new snippet from a file",
	Long: `Upload a file as a new snippet to Code Vault.
The file content will be stored in SeaweedFS and metadata in PostgreSQL.`,
	Args: cobra.ExactArgs(1), // Requires exactly 1 argument (the file path)
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		filePath := args[0]

		// Verify file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", filePath)
		}

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

		// Connect to SeaweedFS
		sfs, err := storage.NewSeaweedFS(&cfg.SeaweedFS)
		if err != nil {
			return fmt.Errorf("failed to connect to SeaweedFS: %w", err)
		}

		// Initialize repositories
		snippetRepo := storage.NewSnippetRepository(db.DB)
		userRepo := storage.NewUserRepository(db.DB)

		// Initialize service
		snippetService := service.NewSnippetService(snippetRepo, sfs)

		// For now, we'll create a default user if none exists
		// In a real app, you'd authenticate with API key from config
		user, err := getOrCreateDefaultUser(userRepo, cfg)
		if err != nil {
			return fmt.Errorf("failed to get user: %w", err)
		}

		// Auto-detect language if not provided
		language := createLanguage
		if language == "" {
			language = service.DetectLanguage(filePath)
		}

		// Auto-generate title if not provided
		title := createTitle
		if title == "" {
			title = filepath.Base(filePath)
		}

		// Parse expiration
		var expiresAt *time.Time
		if createExpires != "" {
			expires, err := parseExpiration(createExpires)
			if err != nil {
				return fmt.Errorf("invalid expiration: %w", err)
			}
			expiresAt = &expires
		}

		// Create snippet
		fmt.Println("📝 Creating snippet...")
		fmt.Printf("   File: %s\n", filePath)
		fmt.Printf("   Title: %s\n", title)
		fmt.Printf("   Language: %s\n", language)
		if len(createTags) > 0 {
			fmt.Printf("   Tags: %s\n", strings.Join(createTags, ", "))
		}
		fmt.Println()

		snippet, err := snippetService.CreateSnippet(ctx, service.CreateSnippetInput{
			UserID:      user.ID,
			Title:       title,
			Description: createDescription,
			Language:    language,
			IsPublic:    createPublic,
			Tags:        createTags,
			ExpiresAt:   expiresAt,
			FilePath:    filePath,
		})

		if err != nil {
			return fmt.Errorf("failed to create snippet: %w", err)
		}

		// Success!
		fmt.Println()
		fmt.Println("✅ Snippet created successfully!")
		fmt.Println()
		fmt.Printf("   ID: %s\n", snippet.ID)
		fmt.Printf("   Title: %s\n", snippet.Title)
		fmt.Printf("   Language: %s\n", snippet.Language)
		if snippet.CurrentVersion != nil {
			fmt.Printf("   Size: %d bytes\n", snippet.CurrentVersion.SizeBytes)
			fmt.Printf("   Hash: %s\n", snippet.CurrentVersion.ContentHash[:16]+"...")
		}
		if len(snippet.Tags) > 0 {
			var tagNames []string
			for _, tag := range snippet.Tags {
				tagNames = append(tagNames, tag.Name)
			}
			fmt.Printf("   Tags: %s\n", strings.Join(tagNames, ", "))
		}
		if snippet.ExpiresAt != nil {
			fmt.Printf("   Expires: %s\n", snippet.ExpiresAt.Format("2006-01-02 15:04:05"))
		}
		fmt.Println()
		fmt.Printf("Retrieve with: vault get %s\n", snippet.ID)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(createCmd)

	// Define flags
	createCmd.Flags().StringVarP(&createTitle, "title", "t", "", "Snippet title (default: filename)")
	createCmd.Flags().StringVarP(&createDescription, "description", "d", "", "Snippet description")
	createCmd.Flags().StringVarP(&createLanguage, "language", "l", "", "Programming language (auto-detected if not provided)")
	createCmd.Flags().StringSliceVar(&createTags, "tags", []string{}, "Comma-separated tags (e.g., --tags go,util,backup)")
	createCmd.Flags().BoolVarP(&createPublic, "public", "p", false, "Make snippet public")
	createCmd.Flags().StringVarP(&createExpires, "expires", "e", "", "Expiration duration (e.g., 24h, 7d, 30d)")
}

// getOrCreateDefaultUser gets or creates a default user for MVP
// In production, this would use API key authentication
func getOrCreateDefaultUser(userRepo *storage.UserRepository, cfg *config.Config) (*models.User, error) {
	ctx := context.Background()
	username := cfg.User.Username
	if username == "" {
		username = "default"
	}

	// Try to get existing user
	user, err := userRepo.GetUserByUsername(ctx, username)
	if err == nil {
		return user, nil
	}

	// User doesn't exist, create one
	email := username + "@localhost"
	apiKey := "default-api-key" // In production, generate secure random key

	user, err = userRepo.CreateUser(ctx, username, email, apiKey)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// parseExpiration parses expiration strings like "24h", "7d", "30d"
func parseExpiration(exp string) (time.Time, error) {
	// Parse duration
	var duration time.Duration
	var err error

	if days, found := strings.CutSuffix(exp, "d"); found {
		var d int
		_, err = fmt.Sscanf(days, "%d", &d)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid duration format")
		}
		duration = time.Duration(d) * 24 * time.Hour
	} else {
		// Standard Go duration (1h, 30m, etc.)
		duration, err = time.ParseDuration(exp)
		if err != nil {
			return time.Time{}, err
		}
	}

	return time.Now().Add(duration), nil
}
