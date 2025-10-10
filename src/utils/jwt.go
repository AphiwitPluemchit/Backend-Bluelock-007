package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func getJWTSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "your_secret_key" // fallback for development
	}
	return []byte(secret)
}

type JWTClaims struct {
	UserID string `json:"userId"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	Type   string `json:"type"` // "access" or "refresh"
	jwt.RegisteredClaims
}

// GenerateJWT generates a single access token (legacy, for backward compatibility)
func GenerateJWT(userID, email, role string) (string, error) {
	claims := JWTClaims{
		UserID: userID,
		Email:  email,
		Role:   role,
		Type:   "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(getJWTSecret())
}

// GenerateTokenPair generates both access token and refresh token
func GenerateTokenPair(userID, email, role string) (accessToken string, refreshToken string, err error) {
	// 1. Access Token (อายุ 15 นาที)
	accessClaims := JWTClaims{
		UserID: userID,
		Email:  email,
		Role:   role,
		Type:   "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	accessTokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessToken, err = accessTokenObj.SignedString(getJWTSecret())
	if err != nil {
		return "", "", fmt.Errorf("failed to generate access token: %v", err)
	}

	// 2. Refresh Token (อายุ 7 วัน)
	refreshClaims := JWTClaims{
		UserID: userID,
		Email:  email,
		Role:   role,
		Type:   "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	refreshTokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshToken, err = refreshTokenObj.SignedString(getJWTSecret())
	if err != nil {
		return "", "", fmt.Errorf("failed to generate refresh token: %v", err)
	}

	return accessToken, refreshToken, nil
}

func ParseJWT(tokenStr string) (*JWTClaims, error) {
	if tokenStr == "" {
		return nil, fmt.Errorf("empty token string")
	}

	token, err := jwt.ParseWithClaims(tokenStr, &JWTClaims{}, func(t *jwt.Token) (interface{}, error) {
		return getJWTSecret(), nil
	})

	if err != nil || token == nil {
		return nil, fmt.Errorf("token parsing failed: %v", err)
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

// GenerateRandomString generates a random string of specified length
func GenerateRandomString(length int) string {
	bytes := make([]byte, length/2)
	if _, err := rand.Read(bytes); err != nil {
		return ""
	}
	return hex.EncodeToString(bytes)
}
