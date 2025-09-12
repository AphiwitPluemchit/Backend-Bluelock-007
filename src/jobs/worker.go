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

func HandleCompleteProgramTask(ctx context.Context, t *asynq.Task) error {
	log.Println("🎯 Start task handler")

	var payload ProgramPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		log.Println("❌ Payload decode error:", err)
		return err
	}

	collection := database.GetCollection("BluelockDB", "Programs")
	id, _ := primitive.ObjectIDFromHex(payload.ProgramID)

	// ✅ ตรวจสอบว่า program ยังมีอยู่ไหม
	var program bson.M
	err := collection.FindOne(ctx, bson.M{"_id": id}).Decode(&program)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Println("⚠️ Program not found. Possibly deleted. Skipping task:", id.Hex())
			return nil // ✅ ไม่ถือว่า error
		}
		log.Println("❌ Failed to find program:", err)
		return err
	}

	// ✅ ดำเนินการเปลี่ยนสถานะ
	_, err = collection.UpdateOne(ctx,
		bson.M{"_id": id},
		bson.M{"$set": bson.M{"programState": "complete"}},
	)

	if err != nil {
		log.Println("❌ Failed to update program state:", err)
		return err
	}

	log.Println("✅ Program closed:", id.Hex())
	return nil
}

func HandleCloseEnrollTask(ctx context.Context, t *asynq.Task) error {
	var payload ProgramPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return err
	}

	objectID, err := primitive.ObjectIDFromHex(payload.ProgramID)
	if err != nil {
		return err
	}

	// เปลี่ยน state → "close"
	filter := bson.M{"_id": objectID}
	update := bson.M{"$set": bson.M{"programState": "close"}}
	_, err = database.GetCollection("BluelockDB", "Programs").UpdateOne(context.TODO(), filter, update)

	if err == nil {
		log.Println("✅ Program auto-closed after enroll deadline:", payload.ProgramID)
	}

	return err
}
