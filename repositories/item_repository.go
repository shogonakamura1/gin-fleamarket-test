package repositories

import (
	"gin-fleamarket/models"

	"gorm.io/gorm"
)

type IItemRepository interface {
	FindAll() (*[]models.Item, error)
	FindById(itemID uint, userID uint) (*models.Item, error)
	Create(newItem models.Item) (*models.Item, error)
	Update(itemID uint, userID uint, updates map[string]interface{}) (*models.Item, error)
	Delete(itemID uint) error
}

type ItemRepository struct {
	db *gorm.DB
}

func (r *ItemRepository) Create(newItem models.Item) (*models.Item, error) {
	result := r.db.Create(&newItem)
	if result.Error != nil {
		return nil, result.Error
	}
	return &newItem, nil
}

func (r *ItemRepository) Delete(itemID uint) error {
	result := r.db.Delete(&models.Item{}, "id = ?", itemID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *ItemRepository) FindAll() (*[]models.Item, error) {
	var items []models.Item
	result := r.db.Find(&items)
	if result.Error != nil {
		return nil, result.Error
	}
	return &items, nil
}

func (r *ItemRepository) FindById(itemID uint, userID uint) (*models.Item, error) {
	var item models.Item
	result := r.db.First(&item, "id = ? AND user_id = ?", itemID, userID)
	if result.Error != nil {
		return nil, result.Error
	}
	return &item, nil
}

func (r *ItemRepository) Update(itemID uint, userID uint, updates map[string]interface{}) (*models.Item, error) {
	result := r.db.Model(&models.Item{}).
		Where("id = ? AND user_id = ?", itemID, userID).
		Updates(updates)

	if result.Error != nil {
		return nil, result.Error
	}

	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	var updatedItem models.Item
	if err := r.db.First(&updatedItem, "id = ?", itemID).Error; err != nil {
		return nil, err
	}

	return &updatedItem, nil
}

func NewItemRepository(db *gorm.DB) IItemRepository {
	return &ItemRepository{db: db}
}
