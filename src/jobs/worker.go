package jobs

import (
	"Backend-Bluelock-007/src/database"
	"context"
	"encoding/json"
	"log"

	"github.com/hibiken/asynq"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func HandleCloseActivityTask(ctx context.Context, t *asynq.Task) error {
	log.Println("🎯 Start task handler")
	var payload CloseActivityPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		log.Println("❌ Payload decode error:", err)
		return err
	}

	id, _ := primitive.ObjectIDFromHex(payload.ActivityID)
	_, err := database.GetCollection("BluelockDB", "activitys").UpdateOne(ctx,
		bson.M{"_id": id},
		bson.M{"$set": bson.M{"activityState": "close"}},
	)
	if err != nil {
		log.Println("❌ Failed to update activity state:", err)
		return err
	}

	log.Println("🎯 Running CloseActivity Task for", payload.ActivityID)

	log.Println("✅ Activity closed:", id.Hex())
	return nil
}
