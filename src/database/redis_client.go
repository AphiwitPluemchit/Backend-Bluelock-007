package database

import (
	"context"
	"os"

	"github.com/redis/go-redis/v9"
)

var RedisURI string
var RedisClient *redis.Client
var RedisCtx = context.Background()

func InitRedis() {

	RedisURI = os.Getenv("REDIS_URI")
	if RedisURI != "" {

		RedisClient = redis.NewClient(&redis.Options{
			Addr:     RedisURI, // เช่น localhost:6379
			Password: "",       // ถ้าไม่มีรหัสผ่าน
			DB:       0,
		})
		_, err := RedisClient.Ping(RedisCtx).Result()
		if err != nil {
			panic("❌ Failed to connect Redis: " + err.Error())
		}
	}
}
