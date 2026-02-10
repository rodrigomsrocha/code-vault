package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Snippet struct {
	ID               uuid.UUID        `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID           uuid.UUID        `gorm:"type:uuid;not null;index" json:"user_id"`
	User             User             `gorm:"foreignKey:UserID" json:"user"`
	Title            string           `gorm:"size:255;not null" json:"title"`
	Description      string           `gorm:"type:text" json:"description"`
	Language         string           `gorm:"size:50;not null;index" json:"language"`
	IsPublic         bool             `gorm:"default:false;index" json:"is_public"`
	CurrentVersionID *uuid.UUID       `gorm:"type:uuid;index" json:"current_version_id,omitempty"`
	CurrentVersion   *SnippetVersion  `gorm:"foreignKey:CurrentVersionID;constraint:OnDelete:SET NULL" json:"current_version,omitempty"`
	Versions         []SnippetVersion `gorm:"foreignKey:SnippetID" json:"versions,omitempty"`
	Tags             []Tag            `gorm:"many2many:snippet_tags;" json:"tags,omitempty"`
	ExpiresAt        *time.Time       `gorm:"index" json:"expires_at,omitempty"`
	CreatedAt        time.Time        `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time        `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt        gorm.DeletedAt   `gorm:"index" json:"deleted_at"`
}

type SnippetVersion struct {
	ID            uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	SnippetID     uuid.UUID `gorm:"type:uuid;not null;index" json:"snippet_id"`
	Snippet       Snippet   `gorm:"foreignKey:SnippetID;constraint:OnDelete:CASCADE" json:"snippet"`
	VersionNumber int       `gorm:"not null" json:"version_number"`
	ContentHash   string    `gorm:"size:64;not null;index" json:"content_hash"`
	SeaweedFSFID  string    `gorm:"size:255;not null" json:"seaweedfs_fid"`
	SizeBytes     int64     `gorm:"not null" json:"size_bytes"`
	CreatedAt     time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (SnippetVersion) TableName() string {
	return "snippet_versions"
}

type Tag struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name      string    `gorm:"size:50;uniqueIndex;not null" json:"name"`
	Snippets  []Snippet `gorm:"many2many:snippet_tags;" json:"snippets,omitempty"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

type SnippetTag struct {
	SnippetID uuid.UUID `gorm:"type:uuid;primaryKey" json:"snippet_id"`
	TagID     uuid.UUID `gorm:"type:uuid;primaryKey" json:"tag_id"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (SnippetTag) TableName() string {
	return "snippet_tags"
}

type User struct {
	ID         uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Username   string         `gorm:"size:50;uniqueIndex;not null" json:"username"`
	Email      string         `gorm:"size:255;uniqueIndex;not null" json:"email"`
	APIKeyHash string         `gorm:"size:255;not null" json:"-"`
	Snippets   []Snippet      `gorm:"foreignKey:UserID" json:"snippets,omitempty"`
	CreatedAt  time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

func (sv *SnippetVersion) BeforeCreate(tx *gorm.DB) error {
	if sv.ID == uuid.Nil {
		sv.ID = uuid.New()
	}
	return nil
}

func (s *Snippet) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

func AddCustomIndexes(db *gorm.DB) error {
	// Custom composite unique index for version numbers
	if err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_snippet_version_unique
		ON snippet_versions(snippet_id, version_number)
	`).Error; err != nil {
		return fmt.Errorf("failed to create version unique index: %w", err)
	}

	// Partial index for public snippets
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_snippets_public
		ON snippets(is_public) WHERE is_public = true
	`).Error; err != nil {
		return fmt.Errorf("failed to create public snippets index: %w", err)
	}

	// Partial index for expiring snippets
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_snippets_expiring
		ON snippets(expires_at) WHERE expires_at IS NOT NULL
	`).Error; err != nil {
		return fmt.Errorf("failed to create expiring snippets index: %w", err)
	}

	return nil
}
