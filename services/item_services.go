package services

import (
	"gin-fleamarket/dto"
	"gin-fleamarket/models"
	"gin-fleamarket/repositories"
)

type IItemService interface {
	FindAll() (*[]models.Item, error)
	FindById(itemID uint, userID uint) (*models.Item, error)
	Create(createItemInput dto.CreateItemInput, userID uint) (*models.Item, error)
	Update(itemID uint, userID uint, updateItemInput dto.UpdateItemInput) (*models.Item, error)
	Delete(itemID uint, userID uint) error
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
	return s.repository.FindById(itemID, userID)
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
	targetItem, err := s.FindById(itemID, userID)
	if err != nil {
		return nil, err
	}

	if updateItemInput.Name != nil {
		targetItem.Name = *updateItemInput.Name
	}
	if updateItemInput.Price != nil {
		targetItem.Price = *updateItemInput.Price
	}
	if updateItemInput.Description != nil {
		targetItem.Description = *updateItemInput.Description
	}
	if updateItemInput.SoldOut != nil {
		targetItem.SoldOut = *updateItemInput.SoldOut
	}
	return s.repository.Update(*targetItem)
}

func (s *ItemService) Delete(itemID uint, userID uint) error {
	return s.repository.Delete(itemID, userID)
}
