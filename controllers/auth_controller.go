package controllers

import (
	"gin-fleamarket/dto"
	"gin-fleamarket/services"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type IAuthController interface {
	Signup(ctx *gin.Context)
	Login(ctx *gin.Context)
	RefreshToken(ctx *gin.Context)
	Logout(ctx *gin.Context)
}

type AuthController struct {
	service services.IAuthService
}

func NewAuthController(service services.IAuthService) IAuthController {
	return &AuthController{service: service}
}

func (c *AuthController) Signup(ctx *gin.Context) {
	var input dto.SignupInput
	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := c.service.Signup(input.Email, input.Password)
	if err != nil {
		log.Printf("Signup error: %v", err)
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "UNIQUE constraint") {
			ctx.JSON(http.StatusConflict, gin.H{"error": "Email already exists"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Unexpected error"})
		return
	}
	ctx.Status(http.StatusCreated)
}

func (c *AuthController) Login(ctx *gin.Context) {
	var input dto.LoginInput
	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tokenPair, err := c.service.Login(input.Email, input.Password)
	if err != nil {
		if err.Error() == "User not found" {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Unexpected error"})
		return
	}
	ctx.JSON(http.StatusOK, dto.LoginResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
	})
}

func (c *AuthController) RefreshToken(ctx *gin.Context) {
	var input dto.RefreshTokenInput
	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tokenPair, err := c.service.RefreshToken(input.RefreshToken)
	if err != nil {
		if err.Error() == "token is expired" || err.Error() == "invalid refresh token" || err.Error() == "invalid token type" || err.Error() == "refresh token is blacklisted" {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Unexpected error"})
		return
	}
	ctx.JSON(http.StatusOK, dto.LoginResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
	})
}

func (c *AuthController) Logout(ctx *gin.Context) {
	header := ctx.GetHeader("Authorization")
	if header == "" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
		return
	}

	if !strings.HasPrefix(header, "Bearer ") {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
		return
	}

	tokenString := strings.TrimPrefix(header, "Bearer ")
	err := c.service.Logout(tokenString)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to logout"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Successfully logged out"})
}
