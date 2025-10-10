package utils

import (
	"context"
	"fmt"
	"time"

	DB "Backend-Bluelock-007/src/database"

	"github.com/redis/go-redis/v9"
)

var Ctx = context.Background()

// ensureClient returns the shared Redis client managed by the database package.
// If the database package didn't initialize Redis, this will return nil and
// callers should handle that case (they already do).
func ensureClient() *redis.Client {
	return DB.RedisClient
}

// InitRedis delegates initialization to database.InitRedis so there is a single
// place responsible for creating and pinging the Redis client.
func InitRedis() {
	DB.InitRedis()
}

// StoreRefreshToken เก็บ refresh token ใน Redis พร้อม expiration
// Returns nil if Redis is not available (development mode)
func StoreRefreshToken(userID, refreshToken string, expiresIn time.Duration) error {
	client := ensureClient()
	if client == nil {
		fmt.Println("redis client not initialized")
		// ไม่มี Redis ใน dev mode - ข้าม
		return nil
	}

	key := fmt.Sprintf("refresh_token:%s", userID)
	err := client.Set(Ctx, key, refreshToken, expiresIn).Err()
	if err != nil {
		return fmt.Errorf("failed to store refresh token: %v", err)
	}
	return nil
}

// ValidateRefreshToken ตรวจสอบว่า refresh token ตรงกับที่เก็บไว้ใน Redis หรือไม่
// Returns true if Redis is not available (development mode - skip validation)
func ValidateRefreshToken(userID, refreshToken string) (bool, error) {
	client := ensureClient()
	if client == nil {
		fmt.Println("redis client not initialized")
		// ไม่มี Redis ใน dev mode - ข้ามการตรวจสอบ (อนุญาตให้ผ่าน)
		return true, nil
	}

	key := fmt.Sprintf("refresh_token:%s", userID)
	storedToken, err := client.Get(Ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil // Token ไม่มีใน Redis
		}
		return false, fmt.Errorf("failed to get refresh token: %v", err)
	}

	return storedToken == refreshToken, nil
}

// DeleteRefreshToken ลบ refresh token จาก Redis (ใช้ตอน logout)
// Returns nil if Redis is not available (development mode)
func DeleteRefreshToken(userID string) error {
	client := ensureClient()
	if client == nil {
		fmt.Println("redis client not initialized")
		// ไม่มี Redis ใน dev mode - ข้าม
		return nil
	}

	key := fmt.Sprintf("refresh_token:%s", userID)
	err := client.Del(Ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete refresh token: %v", err)
	}
	return nil
}

// BlacklistToken เพิ่ม access token เข้า blacklist (ใช้ตอน logout)
// Returns nil if Redis is not available (development mode)
func BlacklistToken(token string, expiresIn time.Duration) error {
	client := ensureClient()
	if client == nil {
		fmt.Println("redis client not initialized")
		// ไม่มี Redis ใน dev mode - ข้าม
		return nil
	}

	key := fmt.Sprintf("blacklist:%s", token)
	err := client.Set(Ctx, key, "1", expiresIn).Err()
	if err != nil {
		return fmt.Errorf("failed to blacklist token: %v", err)
	}
	return nil
}

// IsTokenBlacklisted ตรวจสอบว่า token อยู่ใน blacklist หรือไม่
// Returns false if Redis is not available (development mode - allow all tokens)
func IsTokenBlacklisted(token string) (bool, error) {
	client := ensureClient()
	if client == nil {
		fmt.Println("redis client not initialized")
		// ไม่มี Redis ใน dev mode - ไม่มี blacklist (อนุญาตให้ผ่าน)
		return false, nil
	}

	key := fmt.Sprintf("blacklist:%s", token)
	_, err := client.Get(Ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil // Token ไม่อยู่ใน blacklist
		}
		return false, fmt.Errorf("failed to check blacklist: %v", err)
	}
	return true, nil
}
