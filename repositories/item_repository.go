package repositories

import (
	"errors"
	"gin-fleamarket/models"

	"gorm.io/gorm"
)

type IItemRepository interface {
	FindAll() (*[]models.Item, error)
	FindById(itemID uint, userID uint) (*models.Item, error)
	Create(newItem models.Item) (*models.Item, error)
	Update(updateItem models.Item) (*models.Item, error)
	Delete(itemID uint, userID uint) error
}

type ItemMemoryRepository struct {
	items []models.Item
}

func NewItemMemoryRepository(items []models.Item) IItemRepository {
	return &ItemMemoryRepository{items: items}
}

func (r *ItemMemoryRepository) FindAll() (*[]models.Item, error) {
	return &r.items, nil
}

func (r *ItemMemoryRepository) FindById(itemID uint, userID uint) (*models.Item, error) {
	for _, v := range r.items {
		if v.ID == itemID {
			return &v, nil
		}
	}
	return nil, errors.New("Item not found")
}

func (r *ItemMemoryRepository) Create(newItem models.Item) (*models.Item, error) {
	newItem.ID = uint(len(r.items) + 1)
	r.items = append(r.items, newItem)
	return &newItem, nil
}

func (r *ItemMemoryRepository) Update(updateItem models.Item) (*models.Item, error) {
	for i, v := range r.items {
		if v.ID == updateItem.ID {
			r.items[i] = updateItem
			return &r.items[i], nil
		}
	}
	return nil, errors.New("Unexpected error")
}

func (r *ItemMemoryRepository) Delete(itemID uint, userID uint) error {
	for i, v := range r.items {
		if v.ID == itemID {
			r.items = append(r.items[:i], r.items[i+1:]...)
			return nil
		}
	}
	return errors.New("Item not found")
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

func (r *ItemRepository) Delete(itemID uint, userID uint) error {
	var deleteItem models.Item

	// userIDが0の場合はadminロールなので、user_idのチェックをスキップ
	if userID == 0 {
		result := r.db.First(&deleteItem, "id = ?", itemID)
		if result.Error != nil {
			if result.Error.Error() == "record not found" {
				return errors.New("Item not found")
			}
			return result.Error
		}
	} else {
		item, err := r.FindById(itemID, userID)
		if err != nil {
			return err
		}
		deleteItem = *item
	}

	result := r.db.Delete(&deleteItem)
	if result.Error != nil {
		return result.Error
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
		if result.Error.Error() == "record not found" {
			return nil, errors.New("Item not found")
		}
		return nil, result.Error
	}
	return &item, nil
}

func (r *ItemRepository) Update(updateItem models.Item) (*models.Item, error) {
	result := r.db.Save(&updateItem)
	if result.Error != nil {
		return nil, result.Error
	}
	return &updateItem, nil
}

func NewItemRepository(db *gorm.DB) IItemRepository {
	return &ItemRepository{db: db}
}
