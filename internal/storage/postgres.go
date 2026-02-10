package storage

import (
	"fmt"
	"time"

	"github.com/rodrigomsrocha/code-vault/internal/config"
	"github.com/rodrigomsrocha/code-vault/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DB struct {
	*gorm.DB
}

func NewDB(cfg *config.DatabaseConfig) (*DB, error) {
	connectionString := fmt.Sprintf(
		"user=%s password=%s dbname=%s host=%s port=%d sslmode=%s",
		cfg.User,
		cfg.Password,
		cfg.DBName,
		cfg.Host,
		cfg.Port,
		cfg.SSLMode,
	)

	db, err := gorm.Open(postgres.Open(connectionString), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
		// Disable foreign key constraint creation
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql database: %w", err)
	}

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(time.Hour)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{db}, nil
}

func (db *DB) Close() error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// AutoMigrate runs auto-migration for all models
func (db *DB) AutoMigrate() error {
	fmt.Println("   📝 Creating tables...")

	// Migrate in the correct order
	if err := db.DB.AutoMigrate(
		&models.User{},
		&models.Tag{},
		&models.Snippet{},
		&models.SnippetVersion{},
		&models.SnippetTag{},
	); err != nil {
		return fmt.Errorf("failed to auto-migrate: %w", err)
	}

	fmt.Println("   🔗 Adding foreign key constraints...")

	// Now manually add foreign key constraints
	if err := db.addForeignKeys(); err != nil {
		return fmt.Errorf("failed to add foreign keys: %w", err)
	}

	fmt.Println("   📊 Adding custom indexes...")

	// Add custom indexes
	if err := models.AddCustomIndexes(db.DB); err != nil {
		return fmt.Errorf("failed to add custom indexes: %w", err)
	}

	return nil
}

// addForeignKeys manually adds foreign key constraints after tables are created
func (db *DB) addForeignKeys() error {
	constraints := []struct {
		name  string
		table string
		sql   string
	}{
		{
			name:  "fk_snippets_user",
			table: "snippets",
			sql: `ALTER TABLE snippets
				  ADD CONSTRAINT fk_snippets_user
				  FOREIGN KEY (user_id)
				  REFERENCES users(id)
				  ON DELETE CASCADE`,
		},
		{
			name:  "fk_snippet_versions_snippet",
			table: "snippet_versions",
			sql: `ALTER TABLE snippet_versions
				  ADD CONSTRAINT fk_snippet_versions_snippet
				  FOREIGN KEY (snippet_id)
				  REFERENCES snippets(id)
				  ON DELETE CASCADE`,
		},
		{
			name:  "fk_snippets_current_version",
			table: "snippets",
			sql: `ALTER TABLE snippets
				  ADD CONSTRAINT fk_snippets_current_version
				  FOREIGN KEY (current_version_id)
				  REFERENCES snippet_versions(id)
				  ON DELETE SET NULL`,
		},
		{
			name:  "fk_snippet_tags_snippet",
			table: "snippet_tags",
			sql: `ALTER TABLE snippet_tags
				  ADD CONSTRAINT fk_snippet_tags_snippet
				  FOREIGN KEY (snippet_id)
				  REFERENCES snippets(id)
				  ON DELETE CASCADE`,
		},
		{
			name:  "fk_snippet_tags_tag",
			table: "snippet_tags",
			sql: `ALTER TABLE snippet_tags
				  ADD CONSTRAINT fk_snippet_tags_tag
				  FOREIGN KEY (tag_id)
				  REFERENCES tags(id)
				  ON DELETE CASCADE`,
		},
	}

	for _, constraint := range constraints {
		// Check if constraint already exists
		var count int64
		db.Raw(`
			SELECT COUNT(*)
			FROM information_schema.table_constraints
			WHERE constraint_name = ? AND table_name = ?
		`, constraint.name, constraint.table).Scan(&count)

		if count == 0 {
			if err := db.Exec(constraint.sql).Error; err != nil {
				return fmt.Errorf("failed to add constraint %s: %w", constraint.name, err)
			}
			fmt.Printf("      ✓ Added: %s\n", constraint.name)
		} else {
			fmt.Printf("      ⊘ Skipped: %s (already exists)\n", constraint.name)
		}
	}

	return nil
}
