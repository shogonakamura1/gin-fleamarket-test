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

	user := models.User{
		Email:    email,
		Password: string(hashedPassword),
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
		"exp":   time.Now().Add(7 * 24 * time.Hour).Unix(), // 7日間有効
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
		// トークンタイプがaccessであることを確認（リフレッシュトークンは受け付けない）
		if tokenType, ok := claims["type"].(string); ok && tokenType != "access" {
			return nil, fmt.Errorf("invalid token type: access token required")
		}

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

func (s *AuthService) RefreshToken(refreshTokenString string) (*TokenPair, error) {
	// リフレッシュトークンをパース
	token, err := jwt.Parse(refreshTokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected method: %v", token.Header["alg"])
		}
		return []byte(os.Getenv("SECRET_KEY")), nil
	})
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %v", err)
	}

	// トークンの有効性をチェック
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		// トークンタイプがrefreshであることを確認
		if tokenType, ok := claims["type"].(string); !ok || tokenType != "refresh" {
			return nil, fmt.Errorf("invalid token type")
		}

		// 有効期限をチェック
		if float64(time.Now().Unix()) > claims["exp"].(float64) {
			return nil, jwt.ErrTokenExpired
		}

		// リフレッシュトークンがブラックリストに含まれているかチェック
		isBlacklisted, err := s.tokenRepository.IsTokenBlacklisted(refreshTokenString)
		if err != nil {
			return nil, err
		}
		if isBlacklisted {
			return nil, fmt.Errorf("refresh token is blacklisted")
		}

		// ユーザー情報を取得
		userID := uint(claims["sub"].(float64))
		email := claims["email"].(string)
		role := claims["role"].(string)

		// 新しいトークンペアを生成
		accessToken, err := CreateAccessToken(userID, email, role)
		if err != nil {
			return nil, err
		}

		newRefreshToken, err := CreateRefreshToken(userID, email, role)
		if err != nil {
			return nil, err
		}

		// 古いリフレッシュトークンをブラックリストに追加
		var expiresAt int64
		if exp, ok := claims["exp"].(float64); ok {
			expiresAt = int64(exp)
		} else {
			expiresAt = time.Now().Add(7 * 24 * time.Hour).Unix()
		}
		if err := s.tokenRepository.AddBlacklistedToken(refreshTokenString, expiresAt); err != nil {
			// ログは出力するが、エラーは返さない（トークン生成は成功しているため）
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
