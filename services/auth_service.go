package services

import (
	"fmt"
	"gin-fleamarket/models"
	"gin-fleamarket/repositories"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type IAuthService interface {
	Signup(email string, password string) error
	Login(email string, password string) (*string, error)
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

	user := models.User{
		Email:    email,
		Password: string(hashedPassword),
	}
	return s.repository.CreateUser(user)
}

func (s *AuthService) Login(email string, password string) (*string, error) {
	foundUser, err := s.repository.FindUser(email)
	if err != nil {
		return nil, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(foundUser.Password), []byte(password))
	if err != nil {
		return nil, err
	}

	token, err := CreateToken(foundUser.ID, foundUser.Email, foundUser.Role)
	if err != nil {
		return nil, err
	}

	return token, nil
}

func CreateToken(userID uint, email string, role string) (*string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"role":  role,
		"exp":   time.Now().Add(time.Hour).Unix(),
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
		if float64(time.Now().Unix()) > claims["exp"].(float64) {
			return nil, jwt.ErrTokenExpired
		}

		// トークンがブラックリストに含まれているかチェック
		isBlacklisted, err := s.tokenRepository.IsTokenBlacklisted(tokenString)
		if err != nil {
			return nil, err
		}
		if isBlacklisted {
			return nil, fmt.Errorf("token is blacklisted")
		}

		user, err = s.repository.FindUser(claims["email"].(string))
		if err != nil {
			return nil, err
		}
	}
	return user, nil
}

func (s *AuthService) Logout(tokenString string) error {
	// トークンをパースして有効期限を取得
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
			// 有効期限が取得できない場合は、現在時刻から1時間後を設定
			expiresAt = time.Now().Add(time.Hour).Unix()
		}
	} else {
		expiresAt = time.Now().Add(time.Hour).Unix()
	}

	// トークンをブラックリストに追加
	return s.tokenRepository.AddBlacklistedToken(tokenString, expiresAt)
}
