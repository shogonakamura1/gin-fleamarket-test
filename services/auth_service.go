package services

import (
	"fmt"
	"gin-fleamarket/constants"
	"gin-fleamarket/models"
	"gin-fleamarket/repositories"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

type IAuthService interface {
	Signup(email string, password string) error
	Login(email string, password string) (*TokenPair, error)
	RefreshToken(refreshTokenString string) (*TokenPair, error)
	GetUserFromToken(tokenString string) (*models.User, error)
	Logout(tokenString string) error
}

type AuthService struct {
	repository      repositories.IAuthRepository
	tokenRepository repositories.ITokenRepository
}

func NewAuthService(repository repositories.IAuthRepository, tokenRepository repositories.ITokenRepository) IAuthService {
	return &AuthService{
		repository:      repository,
		tokenRepository: tokenRepository,
	}
}

func (s *AuthService) Signup(email string, password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	userCount, err := s.repository.CountUsers()
	if err != nil {
		return err
	}

	log.Printf("Signup: Current user count in DB = %d", userCount)

	role := constants.RoleUser
	if userCount == 0 {
		role = constants.RoleAdmin
		log.Printf("Signup: First user detected, setting role to admin for email=%s", email)
	} else {
		log.Printf("Signup: Existing users found (%d), setting role to user for email=%s", userCount, email)
	}

	user := models.User{
		Email:    email,
		Password: string(hashedPassword),
		Role:     role,
	}
	return s.repository.CreateUser(user)
}

func (s *AuthService) Login(email string, password string) (*TokenPair, error) {
	foundUser, err := s.repository.FindUser(email)
	if err != nil {
		return nil, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(foundUser.Password), []byte(password))
	if err != nil {
		return nil, err
	}

	accessToken, err := CreateAccessToken(foundUser.ID, foundUser.Email, foundUser.Role)
	if err != nil {
		return nil, err
	}

	refreshToken, err := CreateRefreshToken(foundUser.ID, foundUser.Email, foundUser.Role)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  *accessToken,
		RefreshToken: *refreshToken,
	}, nil
}

func CreateAccessToken(userID uint, email string, role string) (*string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"role":  role,
		"type":  "access",
		"exp":   time.Now().Add(time.Hour).Unix(),
	})

	tokenString, err := token.SignedString([]byte(os.Getenv("SECRET_KEY")))
	if err != nil {
		return nil, err
	}
	return &tokenString, nil
}

func CreateRefreshToken(userID uint, email string, role string) (*string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"role":  role,
		"type":  "refresh",
		"exp":   time.Now().Add(7 * 24 * time.Hour).Unix(),
	})

	tokenString, err := token.SignedString([]byte(os.Getenv("SECRET_KEY")))
	if err != nil {
		return nil, err
	}
	return &tokenString, nil
}

func (s *AuthService) GetUserFromToken(tokenString string) (*models.User, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected method: %v", token.Header["alg"])
		}
		return []byte(os.Getenv("SECRET_KEY")), nil
	})
	if err != nil {
		return nil, err
	}

	var user *models.User
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		if tokenType, ok := claims["type"].(string); ok && tokenType != "access" {
			return nil, fmt.Errorf("invalid token type: access token required")
		}

		if float64(time.Now().Unix()) > claims["exp"].(float64) {
			return nil, jwt.ErrTokenExpired
		}

		isBlacklisted, err := s.tokenRepository.IsTokenBlacklisted(tokenString)
		if err != nil {
			return nil, err
		}
		if isBlacklisted {
			return nil, fmt.Errorf("token is blacklisted")
		}

		email := claims["email"].(string)
		user, err = s.repository.FindUser(email)
		if err != nil {
			return nil, err
		}
		log.Printf("GetUserFromToken: Retrieved user from DB - ID=%d, Email=%s, Role=%s",
			user.ID, user.Email, user.Role)
	}
	return user, nil
}

func (s *AuthService) RefreshToken(refreshTokenString string) (*TokenPair, error) {
	token, err := jwt.Parse(refreshTokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected method: %v", token.Header["alg"])
		}
		return []byte(os.Getenv("SECRET_KEY")), nil
	})
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %v", err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		if tokenType, ok := claims["type"].(string); !ok || tokenType != "refresh" {
			return nil, fmt.Errorf("invalid token type")
		}

		if float64(time.Now().Unix()) > claims["exp"].(float64) {
			return nil, jwt.ErrTokenExpired
		}

		isBlacklisted, err := s.tokenRepository.IsTokenBlacklisted(refreshTokenString)
		if err != nil {
			return nil, err
		}
		if isBlacklisted {
			return nil, fmt.Errorf("refresh token is blacklisted")
		}

		userID := uint(claims["sub"].(float64))
		email := claims["email"].(string)
		role := claims["role"].(string)

		accessToken, err := CreateAccessToken(userID, email, role)
		if err != nil {
			return nil, err
		}

		newRefreshToken, err := CreateRefreshToken(userID, email, role)
		if err != nil {
			return nil, err
		}

		var expiresAt int64
		if exp, ok := claims["exp"].(float64); ok {
			expiresAt = int64(exp)
		} else {
			expiresAt = time.Now().Add(7 * 24 * time.Hour).Unix()
		}
		if err := s.tokenRepository.AddBlacklistedToken(refreshTokenString, expiresAt); err != nil {
			fmt.Printf("Warning: Failed to blacklist old refresh token: %v\n", err)
		}

		return &TokenPair{
			AccessToken:  *accessToken,
			RefreshToken: *newRefreshToken,
		}, nil
	}

	return nil, fmt.Errorf("invalid token claims")
}

func (s *AuthService) Logout(tokenString string) error {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected method: %v", token.Header["alg"])
		}
		return []byte(os.Getenv("SECRET_KEY")), nil
	})
	if err != nil {
		return err
	}

	var expiresAt int64
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		if exp, ok := claims["exp"].(float64); ok {
			expiresAt = int64(exp)
		} else {
			expiresAt = time.Now().Add(time.Hour).Unix()
		}
	} else {
		expiresAt = time.Now().Add(time.Hour).Unix()
	}

	return s.tokenRepository.AddBlacklistedToken(tokenString, expiresAt)
}
