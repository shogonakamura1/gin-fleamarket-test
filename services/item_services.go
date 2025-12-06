package services

import (
	"errors"
	"gin-fleamarket/constants"
	"gin-fleamarket/dto"
	"gin-fleamarket/models"
	"gin-fleamarket/repositories"

	"gorm.io/gorm"
)

type IItemService interface {
	FindAll() (*[]models.Item, error)
	FindById(itemID uint, userID uint) (*models.Item, error)
	Create(createItemInput dto.CreateItemInput, userID uint) (*models.Item, error)
	Update(itemID uint, userID uint, updateItemInput dto.UpdateItemInput) (*models.Item, error)
	Delete(itemID uint) error
}

type ItemService struct {
	repository repositories.IItemRepository
}

func NewItemService(repository repositories.IItemRepository) IItemService {
	return &ItemService{repository: repository}
}

func (s *ItemService) FindAll() (*[]models.Item, error) {
	return s.repository.FindAll()
}

func (s *ItemService) FindById(itemID uint, userID uint) (*models.Item, error) {
	item, err := s.repository.FindById(itemID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New(constants.ErrItemNotFound)
		}
		return nil, err
	}
	return item, nil
}

func (s *ItemService) Create(createItemInput dto.CreateItemInput, userID uint) (*models.Item, error) {
	newItem := models.Item{
		Name:        createItemInput.Name,
		Price:       createItemInput.Price,
		Description: createItemInput.Description,
		SoldOut:     false,
		UserID:      userID,
	}
	return s.repository.Create(newItem)
}

func (s *ItemService) Update(itemID uint, userID uint, updateItemInput dto.UpdateItemInput) (*models.Item, error) {
	updates := make(map[string]interface{})

	if updateItemInput.Name != nil {
		updates["name"] = *updateItemInput.Name
	}
	if updateItemInput.Price != nil {
		updates["price"] = *updateItemInput.Price
	}
	if updateItemInput.Description != nil {
		updates["description"] = *updateItemInput.Description
	}
	if updateItemInput.SoldOut != nil {
		updates["sold_out"] = *updateItemInput.SoldOut
	}

	if len(updates) == 0 {
		return nil, errors.New("no fields to update")
	}

	updatedItem, err := s.repository.Update(itemID, userID, updates)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New(constants.ErrItemNotFound)
		}
		return nil, err
	}

	return updatedItem, nil
}

func (s *ItemService) Delete(itemID uint) error {
	err := s.repository.Delete(itemID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New(constants.ErrItemNotFound)
		}
		return err
	}
	return nil
}
