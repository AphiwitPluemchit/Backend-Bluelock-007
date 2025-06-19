package database

import (
	"fmt"
	"os"

	"github.com/hibiken/asynq"
)

var AsynqClient *asynq.Client

func InitAsynq() {
	RedisURI = os.Getenv("REDIS_URI")
	if RedisURI == "" {
		// RedisURI = "localhost:6379"
	} else {
		AsynqClient = asynq.NewClient(asynq.RedisClientOpt{Addr: RedisURI})
		fmt.Println("Redis URI:", RedisURI)
	}

	// AsynqClient = asynq.NewClient(asynq.RedisClientOpt{Addr: RedisURI})
}
