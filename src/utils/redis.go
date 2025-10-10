package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var Ctx = context.Background()
var RedisClient *redis.Client

func InitRedis() {
	RedisClient = redis.NewClient(&redis.Options{
		Addr: "localhost:6379", // แก้ตาม .env ถ้ามี
	})
}

// StoreRefreshToken เก็บ refresh token ใน Redis พร้อม expiration
func StoreRefreshToken(userID, refreshToken string, expiresIn time.Duration) error {
	if RedisClient == nil {
		return fmt.Errorf("redis client not initialized")
	}

	key := fmt.Sprintf("refresh_token:%s", userID)
	err := RedisClient.Set(Ctx, key, refreshToken, expiresIn).Err()
	if err != nil {
		return fmt.Errorf("failed to store refresh token: %v", err)
	}
	return nil
}

// ValidateRefreshToken ตรวจสอบว่า refresh token ตรงกับที่เก็บไว้ใน Redis หรือไม่
func ValidateRefreshToken(userID, refreshToken string) (bool, error) {
	if RedisClient == nil {
		return false, fmt.Errorf("redis client not initialized")
	}

	key := fmt.Sprintf("refresh_token:%s", userID)
	storedToken, err := RedisClient.Get(Ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil // Token ไม่มีใน Redis
		}
		return false, fmt.Errorf("failed to get refresh token: %v", err)
	}

	return storedToken == refreshToken, nil
}

// DeleteRefreshToken ลบ refresh token จาก Redis (ใช้ตอน logout)
func DeleteRefreshToken(userID string) error {
	if RedisClient == nil {
		return fmt.Errorf("redis client not initialized")
	}

	key := fmt.Sprintf("refresh_token:%s", userID)
	err := RedisClient.Del(Ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete refresh token: %v", err)
	}
	return nil
}

// BlacklistToken เพิ่ม access token เข้า blacklist (ใช้ตอน logout)
func BlacklistToken(token string, expiresIn time.Duration) error {
	if RedisClient == nil {
		return fmt.Errorf("redis client not initialized")
	}

	key := fmt.Sprintf("blacklist:%s", token)
	err := RedisClient.Set(Ctx, key, "1", expiresIn).Err()
	if err != nil {
		return fmt.Errorf("failed to blacklist token: %v", err)
	}
	return nil
}

// IsTokenBlacklisted ตรวจสอบว่า token อยู่ใน blacklist หรือไม่
func IsTokenBlacklisted(token string) (bool, error) {
	if RedisClient == nil {
		return false, fmt.Errorf("redis client not initialized")
	}

	key := fmt.Sprintf("blacklist:%s", token)
	_, err := RedisClient.Get(Ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil // Token ไม่อยู่ใน blacklist
		}
		return false, fmt.Errorf("failed to check blacklist: %v", err)
	}
	return true, nil
}
