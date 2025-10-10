package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// getTokenExpiration parses duration from environment variable
func getTokenExpiration(envKey string, defaultDuration time.Duration) time.Duration {
	durationStr := strings.TrimSpace(os.Getenv(envKey))
	if durationStr == "" {
		return defaultDuration
	}

	// Support day suffix like "7d" -> convert to hours
	if strings.HasSuffix(durationStr, "d") || strings.HasSuffix(durationStr, "D") {
		v := strings.TrimRight(strings.TrimRight(durationStr, "d"), "D")
		if days, err := strconv.Atoi(v); err == nil {
			return time.Duration(days) * 24 * time.Hour
		}
		log.Printf("Warning: Invalid %s format '%s', using default %v", envKey, durationStr, defaultDuration)
		return defaultDuration
	}

	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		log.Printf("Warning: Invalid %s format '%s', using default %v", envKey, durationStr, defaultDuration)
		return defaultDuration
	}

	return duration
}

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
// Token expiration durations are read from environment variables:
// - ACCESS_TOKEN_EXPIRE (default: 15m)
// - REFRESH_TOKEN_EXPIRE (default: 7d)
func GenerateTokenPair(userID, email, role string) (accessToken string, refreshToken string, err error) {
	// Get token expiration durations from environment
	accessTokenExpire := getTokenExpiration("ACCESS_TOKEN_EXPIRE", 15*time.Minute)
	refreshTokenExpire := getTokenExpiration("REFRESH_TOKEN_EXPIRE", 7*24*time.Hour)

	log.Printf("🔑 Token Config - Access: %v, Refresh: %v", accessTokenExpire, refreshTokenExpire)

	// 1. Access Token
	accessClaims := JWTClaims{
		UserID: userID,
		Email:  email,
		Role:   role,
		Type:   "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(accessTokenExpire)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	accessTokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessToken, err = accessTokenObj.SignedString(getJWTSecret())
	if err != nil {
		return "", "", fmt.Errorf("failed to generate access token: %v", err)
	}

	// 2. Refresh Token
	refreshClaims := JWTClaims{
		UserID: userID,
		Email:  email,
		Role:   role,
		Type:   "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(refreshTokenExpire)),
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

// GetAccessTokenExpiration returns the access token expiration duration from ENV
func GetAccessTokenExpiration() time.Duration {
	return getTokenExpiration("ACCESS_TOKEN_EXPIRE", 15*time.Minute)
}

// GetRefreshTokenExpiration returns the refresh token expiration duration from ENV
func GetRefreshTokenExpiration() time.Duration {
	return getTokenExpiration("REFRESH_TOKEN_EXPIRE", 7*24*time.Hour)
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
