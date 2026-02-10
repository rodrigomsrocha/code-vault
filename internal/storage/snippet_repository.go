package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rodrigomsrocha/code-vault/internal/models"
	"gorm.io/gorm"
)

type SnippetRepository struct {
	db *gorm.DB
}

func NewSnippetRepository(db *gorm.DB) *SnippetRepository {
	return &SnippetRepository{db: db}
}

// CreateSnippet creates a new snippet (without version yet)
func (r *SnippetRepository) CreateSnippet(ctx context.Context, snippet *models.Snippet) error {
	return gorm.G[models.Snippet](r.db).Create(ctx, snippet)
}

// CreateVersion creates a new version for a snippet
func (r *SnippetRepository) CreateVersion(ctx context.Context, version *models.SnippetVersion) error {
	return gorm.G[models.SnippetVersion](r.db).Create(ctx, version)
}

// UpdateSnippetCurrentVersion updates the snippet's current version pointer
func (r *SnippetRepository) UpdateSnippetCurrentVersion(ctx context.Context, snippetID, versionID uuid.UUID) error {
	_, err := gorm.G[models.Snippet](r.db).Where("id = ?", snippetID).Update(ctx, "current_version_id", versionID)
	return err
}

// GetSnippetByID retrieves a snippet with its current version
func (r *SnippetRepository) GetSnippetByID(ctx context.Context, id uuid.UUID) (*models.Snippet, error) {
	snippet, err := gorm.G[models.Snippet](r.db).
		Where("id = ?", id).
		Preload("CurrentVersion", nil).
		Preload("User", nil).
		Preload("Tags", nil).
		First(ctx)

	if err != nil {
		return nil, err
	}

	return &snippet, nil
}

// GetVersionByHash checks if a version with this hash already exists
func (r *SnippetRepository) GetVersionByHash(ctx context.Context, hash string) (*models.SnippetVersion, error) {
	var version models.SnippetVersion

	err := r.db.WithContext(ctx).
		Table("snippet_versions").
		Select("snippet_versions.*").
		Joins("INNER JOIN snippets ON snippet_versions.snippet_id = snippets.id").
		Where("snippet_versions.content_hash = ?", hash).
		Where("snippets.deleted_at IS NULL").
		First(&version).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &version, nil
}

// GetNextVersionNumber gets the next version number for a snippet
func (r *SnippetRepository) GetNextVersionNumber(ctx context.Context, snippetID uuid.UUID) (int, error) {
	var maxVersion int
	err := gorm.G[models.SnippetVersion](r.db).
		Where("snippet_id = ?", snippetID).
		Select("COALESCE(MAX(version_number), 0)").
		Scan(ctx, &maxVersion)

	if err != nil {
		return 0, err
	}

	return maxVersion + 1, nil
}

// FindOrCreateTag finds an existing tag or creates a new one
func (r *SnippetRepository) FindOrCreateTag(ctx context.Context, name string) (*models.Tag, error) {
	tag, err := gorm.G[models.Tag](r.db).Where("name = ?", name).First(ctx)

	if err == gorm.ErrRecordNotFound {
		// Tag doesn't exist, create it
		tag = models.Tag{Name: name}
		if err := gorm.G[models.Tag](r.db).Create(ctx, &tag); err != nil {
			return nil, err
		}
		return &tag, nil
	}

	if err != nil {
		return nil, err
	}

	return &tag, nil
}

// AddTagsToSnippet associates tags with a snippet
func (r *SnippetRepository) AddTagsToSnippet(ctx context.Context, snippetID uuid.UUID, tagNames []string) error {
	var tags []models.Tag

	for _, name := range tagNames {
		tag, err := r.FindOrCreateTag(ctx, name)
		if err != nil {
			return fmt.Errorf("failed to find/create tag %s: %w", name, err)
		}
		tags = append(tags, *tag)
	}

	// Associate tags with snippet
	return r.db.Model(&models.Snippet{ID: snippetID}).
		Association("Tags").
		Append(tags)
}

// ListUserSnippets lists all snippets for a user
func (r *SnippetRepository) ListUserSnippets(ctx context.Context, userID uuid.UUID) ([]models.Snippet, error) {
	snippets, err := gorm.G[models.Snippet](r.db).
		Preload("CurrentVersion", nil).
		Preload("Tags", nil).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(ctx)

	if err != nil {
		return nil, err
	}

	return snippets, nil
}

// DeleteSnippet soft-deletes a snippet
func (r *SnippetRepository) DeleteSnippet(ctx context.Context, id uuid.UUID) error {
	_, err := gorm.G[models.Snippet](r.db).
		Where("id = ?", id).
		Update(ctx, "deleted_at", time.Now())

	return err
}

func (r *SnippetRepository) ClearSnippetTags(snippetID uuid.UUID) error {
	return r.db.Model(&models.Snippet{ID: snippetID}).
		Association("Tags").
		Clear()
}

func (r *SnippetRepository) GetVersionByNumber(ctx context.Context, snippetID uuid.UUID, versionNumber int) (*models.SnippetVersion, error) {
	version, err := gorm.G[models.SnippetVersion](r.db).
		Where("snippet_id = ? AND version_number = ?", snippetID, versionNumber).
		First(ctx)

	if err != nil {
		return nil, err
	}

	return &version, nil
}

func (r *SnippetRepository) ListSnippetVersions(snippetID uuid.UUID) ([]models.SnippetVersion, error) {
	var versions []models.SnippetVersion
	err := r.db.Where("snippet_id = ?", snippetID).
		Order("version_number DESC").
		Find(&versions).Error

	if err != nil {
		return nil, err
	}

	return versions, nil
}

func (r *SnippetRepository) GetExpiredSnippets(ctx context.Context) ([]models.Snippet, error) {
	snippets, err := gorm.G[models.Snippet](r.db).
		Preload("CurrentVersion", nil).
		Where("expires_at IS NOT NULL AND expires_at < ?", time.Now()).
		Find(ctx)
	if err != nil {
		return nil, err
	}

	return snippets, nil
}

func (r *SnippetRepository) GetExpiringSoon(ctx context.Context, hours int) ([]models.Snippet, error) {
	expiryThreshold := time.Now().Add(time.Duration(hours) * time.Hour)

	snippets, err := gorm.G[models.Snippet](r.db).
		Preload("CurrentVersion", nil).
		Preload("Tags", nil).
		Where("expires_at IS NOT NULL AND expires_at BETWEEN ? AND ?", time.Now(), expiryThreshold).
		Find(ctx)

	if err != nil {
		return nil, err
	}

	return snippets, nil
}

func (r *SnippetRepository) UpdateExpiration(ctx context.Context, snippetID uuid.UUID, expiresAt *time.Time) error {
	_, err := gorm.G[models.Snippet](r.db).
		Where("id = ?", snippetID).
		Update(ctx, "expires_at", expiresAt)

	return err
}

func (r *SnippetRepository) GetContentHashUsageCount(ctx context.Context, contentHash string) (int64, error) {
	var count int64

	query := `
		SELECT COUNT(*)
		FROM snippet_versions sv
		INNER JOIN snippets s ON sv.snippet_id = s.id
		WHERE sv.content_hash = $1
		  AND s.deleted_at IS NULL
	`

	err := r.db.WithContext(ctx).Raw(query, contentHash).Scan(&count).Error
	if err != nil {
		return 0, err
	}

	return count, err
}

func (r *SnippetRepository) GetAllVersionsForSnippet(ctx context.Context, snippetID uuid.UUID) ([]models.SnippetVersion, error) {
	versions, err := gorm.G[models.SnippetVersion](r.db).
		Where("snippet_id = ?", snippetID).
		Find(ctx)

	return versions, err
}

func (r *SnippetRepository) DB() *gorm.DB {
	return r.db
}
