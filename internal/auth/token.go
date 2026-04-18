package auth

import (
	"errors"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken = errors.New("invalid token")
)

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type TokenService interface {
	GenerateTokenPair(userID int, role string) (*TokenPair, error)
	ValidateAccessToken(tokenString string) (userID int, role string, err error)
	ValidateRefreshToken(tokenString string) (userID int, err error)
}

type jwtTokenService struct {
	secretKey     []byte
	refreshSecret []byte
}

func NewTokenService(secret, refreshSecret string) TokenService {
	return &jwtTokenService{
		secretKey:     []byte(secret),
		refreshSecret: []byte(refreshSecret),
	}
}

func (s *jwtTokenService) GenerateTokenPair(userID int, role string) (*TokenPair, error) {
	now := time.Now()

	// Access Token (15 mins)
	accessClaims := jwt.MapClaims{
		"user_id": userID,
		"role":    role,
		"exp":     now.Add(15 * time.Minute).Unix(),
		"iat":     now.Unix(),
		"jti":     uuid.New().String(),
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString(s.secretKey)
	if err != nil {
		return nil, err
	}

	// Refresh Token (7 days)
	refreshClaims := jwt.MapClaims{
		"user_id": userID,
		"exp":     now.Add(7 * 24 * time.Hour).Unix(),
		"iat":     now.Unix(),
		"jti":     uuid.New().String(),
	}
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString(s.refreshSecret)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *jwtTokenService) ValidateAccessToken(tokenString string) (int, string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return s.secretKey, nil
	})

	if err != nil {
		log.Printf("Token parse error: %v", err)
		return 0, "", ErrInvalidToken
	}

	if !token.Valid {
		log.Printf("Token invalid (not valid)")
		return 0, "", ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		log.Printf("Invalid claims type")
		return 0, "", ErrInvalidToken
	}

	userIDF, ok := claims["user_id"]
	if !ok {
		log.Printf("user_id claim missing")
		return 0, "", ErrInvalidToken
	}

	userIDFloat, ok := userIDF.(float64)
	if !ok {
		// Try int64
		if userIDInt, ok := userIDF.(int64); ok {
			return int(userIDInt), "", nil
		}
		log.Printf("user_id not numeric: %T", userIDF)
		return 0, "", ErrInvalidToken
	}

	role, _ := claims["role"].(string)
	return int(userIDFloat), role, nil
}

func (s *jwtTokenService) ValidateRefreshToken(tokenString string) (int, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return s.refreshSecret, nil
	})

	if err != nil || !token.Valid {
		return 0, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, ErrInvalidToken
	}

	userIDF, ok := claims["user_id"].(float64)
	if !ok {
		return 0, ErrInvalidToken
	}

	return int(userIDF), nil
}
