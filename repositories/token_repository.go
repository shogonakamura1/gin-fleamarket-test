package repositories

import (
	"gin-fleamarket/models"
	"time"

	"gorm.io/gorm"
)

type ITokenRepository interface {
	AddBlacklistedToken(token string, expiresAt int64) error
	IsTokenBlacklisted(token string) (bool, error)
	CleanExpiredTokens() error
}

type TokenRepository struct {
	db *gorm.DB
}

func NewTokenRepository(db *gorm.DB) ITokenRepository {
	return &TokenRepository{db: db}
}

func (r *TokenRepository) AddBlacklistedToken(token string, expiresAt int64) error {
	blacklistedToken := models.BlacklistedToken{
		Token:     token,
		ExpiresAt: expiresAt,
	}
	result := r.db.Create(&blacklistedToken)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (r *TokenRepository) IsTokenBlacklisted(token string) (bool, error) {
	var blacklistedToken models.BlacklistedToken
	result := r.db.Where("token = ?", token).First(&blacklistedToken)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, result.Error
	}
	return true, nil
}

func (r *TokenRepository) CleanExpiredTokens() error {
	now := time.Now().Unix()
	result := r.db.Where("expires_at < ?", now).Delete(&models.BlacklistedToken{})
	if result.Error != nil {
		return result.Error
	}
	return nil
}
