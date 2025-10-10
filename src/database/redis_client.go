package database

import (
	"context"
	"fmt"
	"os"

	"github.com/redis/go-redis/v9"
)

var RedisURI string
var RedisClient *redis.Client
var RedisCtx = context.Background()

func InitRedis() {
	// Read REDIS_URI from env. If empty, default to localhost:6379 which is
	// common when running a Redis container mapped to host port 6379.
	RedisURI = os.Getenv("REDIS_URI")
	if RedisURI == "" {
		RedisURI = "localhost:6379"
	}

	RedisClient = redis.NewClient(&redis.Options{
		Addr:     RedisURI, // e.g., localhost:6379
		Password: "",       // set via env if needed
		DB:       0,
	})

	// Try ping; if fail, log a warning and set RedisClient to nil so callers
	// can detect it and handle gracefully (instead of crashing the whole app).
	if _, err := RedisClient.Ping(RedisCtx).Result(); err != nil {
		// don't panic in dev; log and keep RedisClient nil
		RedisClient = nil
		// Use fmt.Println to keep logs simple (main app logs will show this)
		fmt.Printf("⚠️ Redis connection failed (%s): %v\n", RedisURI, err)
		fmt.Println("⚠️ Running in DEV mode without Redis. Caching and background jobs are disabled.")
		return
	}

	fmt.Printf("✅ Redis initialized: %s\n", RedisURI)
}
