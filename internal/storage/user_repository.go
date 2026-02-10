package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"github.com/rodrigomsrocha/code-vault/internal/models"
	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) CreateUser(ctx context.Context, username, email, apiKey string) (*models.User, error) {
	hash := sha256.Sum256([]byte(apiKey))
	apiKeyHash := hex.EncodeToString(hash[:])

	user := &models.User{
		Username:   username,
		Email:      email,
		APIKeyHash: apiKeyHash,
	}

	if err := gorm.G[models.User](r.db).Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (r *UserRepository) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	user, err := gorm.G[models.User](r.db).Where("username = ?", username).First(ctx)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) ValidateAPIKey(ctx context.Context, apiKey string) (*models.User, error) {
	hash := sha256.Sum256([]byte(apiKey))
	apiKeyHash := hex.EncodeToString(hash[:])

	user, err := gorm.G[models.User](r.db).Where("api_key_hash = ?", apiKeyHash).First(ctx)

	if err != nil {
		return nil, err
	}

	return &user, nil
}
