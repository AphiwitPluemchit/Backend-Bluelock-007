package jobs

import (
	"Backend-Bluelock-007/src/database"
	"context"
	"encoding/json"
	"log"

	"github.com/hibiken/asynq"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func HandlecompleteActivityTask(ctx context.Context, t *asynq.Task) error {
	log.Println("🎯 Start task handler")

	var payload ActivityPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		log.Println("❌ Payload decode error:", err)
		return err
	}

	collection := database.GetCollection("BluelockDB", "activitys")
	id, _ := primitive.ObjectIDFromHex(payload.ActivityID)

	// ✅ ตรวจสอบว่า activity ยังมีอยู่ไหม
	var activity bson.M
	err := collection.FindOne(ctx, bson.M{"_id": id}).Decode(&activity)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Println("⚠️ Activity not found. Possibly deleted. Skipping task:", id.Hex())
			return nil // ✅ ไม่ถือว่า error
		}
		log.Println("❌ Failed to find activity:", err)
		return err
	}

	// ✅ ดำเนินการเปลี่ยนสถานะ
	_, err = collection.UpdateOne(ctx,
		bson.M{"_id": id},
		bson.M{"$set": bson.M{"activityState": "complete"}},
	)

	if err != nil {
		log.Println("❌ Failed to update activity state:", err)
		return err
	}

	log.Println("✅ Activity closed:", id.Hex())
	return nil
}

func HandleCloseEnrollTask(ctx context.Context, t *asynq.Task) error {
	var payload ActivityPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return err
	}

	objectID, err := primitive.ObjectIDFromHex(payload.ActivityID)
	if err != nil {
		return err
	}

	// เปลี่ยน state → "close"
	filter := bson.M{"_id": objectID}
	update := bson.M{"$set": bson.M{"activityState": "close"}}
	_, err = database.GetCollection("BluelockDB", "activitys").UpdateOne(context.TODO(), filter, update)

	if err == nil {
		log.Println("✅ Activity auto-closed after enroll deadline:", payload.ActivityID)
	}

	return err
}
