package database

import (
	"log"

	"github.com/hibiken/asynq"
)

var AsynqClient *asynq.Client

// InitAsynq initializes Asynq client only if Redis is available
func InitAsynq() {
	// Check if Redis is available (RedisClient != nil means InitRedis was successful)
	if RedisClient == nil || RedisURI == "" {
		log.Println("⚠️ Redis not available. Asynq client will not be initialized.")
		return
	}

	AsynqClient = asynq.NewClient(asynq.RedisClientOpt{Addr: RedisURI})
	log.Println("✅ Asynq Client initialized successfully")
}
