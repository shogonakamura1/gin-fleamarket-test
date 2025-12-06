package repositories

import (
	"errors"
	"gin-fleamarket/models"
	"log"

	"gorm.io/gorm"
)

type IAuthRepository interface {
	CreateUser(user models.User) error
	FindUser(email string) (*models.User, error)
	CountUsers() (int64, error)
}

type AuthRepository struct {
	db *gorm.DB
}

func NewAuthRepository(db *gorm.DB) IAuthRepository {
	return &AuthRepository{db: db}
}

func (r *AuthRepository) CreateUser(user models.User) error {
	result := r.db.Create(&user)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (r *AuthRepository) FindUser(email string) (*models.User, error) {
	var user models.User
	result := r.db.First(&user, "email = ?", email)
	if result.Error != nil {
		if result.Error.Error() == "record not found" {
			return nil, errors.New("User not found")
		}
		return nil, result.Error
	}
	return &user, nil
}

func (r *AuthRepository) CountUsers() (int64, error) {
	var count int64
	// ソフトデリートされたレコードは除外される（gorm.ModelのDeletedAtがnilのもののみカウント）
	result := r.db.Model(&models.User{}).Count(&count)
	if result.Error != nil {
		return 0, result.Error
	}
	// デバッグ用ログ
	log.Printf("CountUsers: Found %d users in database", count)
	return count, nil
}
