package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/rodrigomsrocha/code-vault/internal/hash"
	"github.com/rodrigomsrocha/code-vault/internal/models"
	"github.com/rodrigomsrocha/code-vault/internal/storage"
)

type SnippetService struct {
	snippetRepo *storage.SnippetRepository
	seaweedfs   *storage.SeaweedFS
}

func NewSnippetService(snippetRepo *storage.SnippetRepository, seaweedfs *storage.SeaweedFS) *SnippetService {
	return &SnippetService{
		snippetRepo: snippetRepo,
		seaweedfs:   seaweedfs,
	}
}

type CreateSnippetInput struct {
	UserID      uuid.UUID
	Title       string
	Description string
	Language    string
	IsPublic    bool
	Tags        []string
	ExpiresAt   *time.Time
	FilePath    string
}

func (s *SnippetService) CreateSnippet(ctx context.Context, input CreateSnippetInput) (*models.Snippet, error) {
	content, err := os.ReadFile(input.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	contentHash := hash.CalculateHash(content)

	existingVersion, err := s.snippetRepo.GetVersionByHash(ctx, contentHash)

	if err != nil {
		return nil, fmt.Errorf("failed to check for existing version: %w", err)
	}

	var seaweedFSFID string

	if existingVersion != nil {
		seaweedFSFID = existingVersion.SeaweedFSFID
		fmt.Println("   ♻️  Content already exists, reusing storage (deduplication)")
	} else {
		seaweedFSFID = contentHash

		if _, err := s.seaweedfs.Upload(ctx, seaweedFSFID, content); err != nil {
			return nil, fmt.Errorf("failed to upload content to SeaweedFS: %w", err)
		}
		fmt.Println("   📦  Content uploaded to SeaweedFS")
	}

	snippet := &models.Snippet{
		UserID:      input.UserID,
		Title:       input.Title,
		Description: input.Description,
		Language:    input.Language,
		IsPublic:    input.IsPublic,
		ExpiresAt:   input.ExpiresAt,
	}

	if err := s.snippetRepo.CreateSnippet(ctx, snippet); err != nil {
		return nil, fmt.Errorf("failed to create snippet: %w", err)
	}

	version := &models.SnippetVersion{
		SnippetID:     snippet.ID,
		VersionNumber: 1,
		ContentHash:   contentHash,
		SeaweedFSFID:  seaweedFSFID,
		SizeBytes:     int64(len(content)),
	}

	if err := s.snippetRepo.CreateVersion(ctx, version); err != nil {
		return nil, fmt.Errorf("failed to create version: %w", err)
	}

	if err := s.snippetRepo.UpdateSnippetCurrentVersion(ctx, snippet.ID, version.ID); err != nil {
		return nil, fmt.Errorf("failed to update snippet current version: %w", err)
	}

	if len(input.Tags) > 0 {
		if err := s.snippetRepo.AddTagsToSnippet(ctx, snippet.ID, input.Tags); err != nil {
			return nil, fmt.Errorf("failed to add tags: %w", err)
		}
	}

	return s.snippetRepo.GetSnippetByID(ctx, snippet.ID)
}

type UpdateSnippetInput struct {
	SnippetID uuid.UUID
	FilePath  string
	Tags      []string
}

func (s *SnippetService) UpdateSnippet(ctx context.Context, input UpdateSnippetInput) (*models.Snippet, *models.SnippetVersion, error) {
	content, err := os.ReadFile(input.FilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read file %s: %w", input.FilePath, err)
	}

	contentHash := hash.CalculateHash(content)

	snippet, err := s.snippetRepo.GetSnippetByID(ctx, input.SnippetID)

	if snippet.CurrentVersion != nil && snippet.CurrentVersion.ContentHash == contentHash {
		return nil, nil, fmt.Errorf("content is identical to current version (no changes)")
	}

	existingVersion, err := s.snippetRepo.GetVersionByHash(ctx, contentHash)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to check for existing version: %w", err)
	}

	var seaweedFSFID string

	if existingVersion != nil {
		seaweedFSFID = existingVersion.SeaweedFSFID
		fmt.Println("   ♻️  Content already exists in another version, reusing storage")
	} else {
		seaweedFSFID = contentHash
		if _, err := s.seaweedfs.Upload(ctx, seaweedFSFID, content); err != nil {
			return nil, nil, fmt.Errorf("failed to upload content to seaweedfs: %w", err)
		}
		fmt.Println("   📤 Uploaded new version to SeaweedFS")
	}

	nextVersion, err := s.snippetRepo.GetNextVersionNumber(ctx, snippet.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get next version number: %w", err)
	}

	version := &models.SnippetVersion{
		SnippetID:     snippet.ID,
		VersionNumber: nextVersion,
		ContentHash:   contentHash,
		SeaweedFSFID:  seaweedFSFID,
		SizeBytes:     int64(len(content)),
	}

	if err := s.snippetRepo.CreateVersion(ctx, version); err != nil {
		return nil, nil, fmt.Errorf("failed to create version: %w", err)
	}

	if err := s.snippetRepo.UpdateSnippetCurrentVersion(ctx, snippet.ID, version.ID); err != nil {
		return nil, nil, fmt.Errorf("failed to update current version: %w", err)
	}

	if len(input.Tags) > 0 {
		if err := s.snippetRepo.ClearSnippetTags(snippet.ID); err != nil {
			return nil, nil, fmt.Errorf("failed to clear tags: %w", err)
		}

		if err := s.snippetRepo.AddTagsToSnippet(ctx, snippet.ID, input.Tags); err != nil {
			return nil, nil, fmt.Errorf("failed to add tags: %w", err)
		}
	}

	updatedSnippet, err := s.snippetRepo.GetSnippetByID(ctx, snippet.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get snippet: %w", err)
	}

	return updatedSnippet, version, nil
}

func (s *SnippetService) GetSnippetVersion(ctx context.Context, snippetID uuid.UUID, versionNumber int) ([]byte, *models.SnippetVersion, error) {
	version, err := s.snippetRepo.GetVersionByNumber(ctx, snippetID, versionNumber)
	if err != nil {
		return nil, nil, fmt.Errorf("version not found: %w", err)
	}

	content, err := s.seaweedfs.Download(ctx, version.SeaweedFSFID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to download content: %w", err)
	}

	return content, version, nil
}

func DetectLanguage(filePath string) string {
	ext := filepath.Ext(filePath)

	languageMap := map[string]string{
		".go":   "go",
		".js":   "javascript",
		".ts":   "typescript",
		".py":   "python",
		".java": "java",
		".c":    "c",
		".cpp":  "cpp",
		".rs":   "rust",
		".rb":   "ruby",
		".php":  "php",
		".sql":  "sql",
		".sh":   "bash",
		".yaml": "yaml",
		".yml":  "yaml",
		".json": "json",
		".xml":  "xml",
		".html": "html",
		".css":  "css",
		".md":   "markdown",
		".txt":  "text",
	}

	if lang, ok := languageMap[ext]; ok {
		return lang
	}

	return "text"
}
