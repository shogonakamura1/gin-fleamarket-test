package controllers

import (
	"gin-fleamarket/constants"
	"gin-fleamarket/dto"
	"gin-fleamarket/models"
	"gin-fleamarket/services"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type IItemController interface {
	FindAll(ctx *gin.Context)
	FindById(ctx *gin.Context)
	Create(ctx *gin.Context)
	Update(ctx *gin.Context)
	Delete(ctx *gin.Context)
}

type ItemController struct {
	service services.IItemService
}

func NewItemController(service services.IItemService) IItemController {
	return &ItemController{service: service}
}

func (c *ItemController) FindAll(ctx *gin.Context) {
	items, err := c.service.FindAll()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": constants.ErrUnexpected})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"data": items})
}

func (c *ItemController) FindById(ctx *gin.Context) {
	user, exists := ctx.Get("user")
	if !exists {
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	userID := user.(*models.User).ID

	itemID, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": constants.ErrInvalidID})
		return
	}

	item, err := c.service.FindById(uint(itemID), userID)
	if err != nil {
		if err.Error() == constants.ErrItemNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"error": constants.ErrItemNotFound})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": constants.ErrUnexpected})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"data": item})
}

func (c *ItemController) Create(ctx *gin.Context) {
	user, exists := ctx.Get("user")
	if !exists {
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	userID := user.(*models.User).ID

	var input dto.CreateItemInput
	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": constants.ErrInvalidInput})
		return
	}

	newItem, err := c.service.Create(input, userID)
	if err != nil {
		log.Printf("Create item error: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": constants.ErrUnexpected})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{"data": newItem})
}

func (c *ItemController) Update(ctx *gin.Context) {
	user, exists := ctx.Get("user")
	if !exists {
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	userID := user.(*models.User).ID

	itemID, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": constants.ErrInvalidID})
		return
	}
	var input dto.UpdateItemInput
	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": constants.ErrInvalidInput})
		return
	}

	updatedItem, err := c.service.Update(uint(itemID), userID, input)
	if err != nil {
		if err.Error() == constants.ErrItemNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"error": constants.ErrItemNotFound})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": constants.ErrUnexpected})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"data": updatedItem})
}

func (c *ItemController) Delete(ctx *gin.Context) {
	itemID, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": constants.ErrInvalidID})
		return
	}

	err = c.service.Delete(uint(itemID))

	if err != nil {
		if err.Error() == constants.ErrItemNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"error": constants.ErrItemNotFound})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": constants.ErrUnexpected})
		return
	}

	ctx.Status(http.StatusOK)
}
