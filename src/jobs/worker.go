package jobs

import (
	DB "Backend-Bluelock-007/src/database"
	hourhistory "Backend-Bluelock-007/src/services/hour-history"
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/hibiken/asynq"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Ensure worker process uses Asia/Bangkok timezone for any time operations.
func init() {
	loc, err := time.LoadLocation("Asia/Bangkok")
	if err != nil {
		log.Println("⚠️ Failed to load Asia/Bangkok location in jobs package:", err)
		return
	}
	time.Local = loc
	log.Println("✅ Set jobs process timezone to Asia/Bangkok (time.Local)")
}

func HandleCompleteProgramTask(ctx context.Context, t *asynq.Task) error {
	log.Println("🎯 Start task handler")

	var payload ProgramPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		log.Println("❌ Payload decode error:", err)
		return err
	}

	id, _ := primitive.ObjectIDFromHex(payload.ProgramID)

	// ✅ ตรวจสอบว่า program ยังมีอยู่ไหม
	var program bson.M
	err := DB.ProgramCollection.FindOne(ctx, bson.M{"_id": id}).Decode(&program)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Println("⚠️ Program not found. Possibly deleted. Skipping task:", id.Hex())
			return nil // ✅ ไม่ถือว่า error
		}
		log.Println("❌ Failed to find program:", err)
		return err
	}

	// ✅ ดำเนินการเปลี่ยนสถานะ
	_, err = DB.ProgramCollection.UpdateOne(ctx,
		bson.M{"_id": id},
		bson.M{"$set": bson.M{"programState": "complete"}},
	)

	if err != nil {
		log.Println("❌ Failed to update program state:", err)
		return err
	}

	log.Println("✅ Program closed:", id.Hex())

	// 📝 ตรวจสอบและให้ชั่วโมงนิสิตที่เข้าร่วมกิจกรรม
	if err := processEnrollmentsForCompletedProgram(ctx, id); err != nil {
		log.Printf("⚠️ Warning: failed to process enrollments for program %s: %v", id.Hex(), err)
		// ไม่ return error เพราะไม่ต้องการให้ task retry
	}

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
	_, err = DB.ProgramCollection.UpdateOne(ctx, filter, update)

	if err == nil {
		log.Println("✅ Program auto-closed after enroll deadline:", payload.ProgramID)
	}

	return err
}

// processEnrollmentsForCompletedProgram ตรวจสอบและให้ชั่วโมงนิสิตเมื่อกิจกรรมเสร็จสิ้น
func processEnrollmentsForCompletedProgram(ctx context.Context, programID primitive.ObjectID) error {
	log.Println("📝 Processing enrollments for completed program:", programID.Hex())

	// 1) หา Program เพื่อดึง totalHours
	var program struct {
		Hour *int `bson:"hour"`
	}
	err := DB.ProgramCollection.FindOne(ctx, bson.M{"_id": programID}).Decode(&program)
	if err != nil {
		return err
	}

	totalHours := 0
	if program.Hour != nil {
		totalHours = *program.Hour
	}

	// 2) หา ProgramItems ทั้งหมดของ program นี้
	cursor, err := DB.ProgramItemCollection.Find(ctx, bson.M{"programId": programID})
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	var programItemIDs []primitive.ObjectID
	for cursor.Next(ctx) {
		var item struct {
			ID primitive.ObjectID `bson:"_id"`
		}
		if err := cursor.Decode(&item); err != nil {
			continue
		}
		programItemIDs = append(programItemIDs, item.ID)
	}

	// 3) หา Enrollments ทั้งหมดที่เกี่ยวข้อง
	enrollCursor, err := DB.EnrollmentCollection.Find(ctx, bson.M{
		"programId":     programID,
		"programItemId": bson.M{"$in": programItemIDs},
	})
	if err != nil {
		return err
	}
	defer enrollCursor.Close(ctx)

	// 4) ประมวลผลแต่ละ enrollment
	successCount := 0
	errorCount := 0

	for enrollCursor.Next(ctx) {
		var enrollment struct {
			ID primitive.ObjectID `bson:"_id"`
		}
		if err := enrollCursor.Decode(&enrollment); err != nil {
			log.Printf("⚠️ Failed to decode enrollment: %v", err)
			errorCount++
			continue
		}

		// เรียกฟังก์ชันตรวจสอบและให้ชั่วโมง
		if err := hourhistory.VerifyAndGrantHours(ctx, enrollment.ID, programID, totalHours); err != nil {
			log.Printf("⚠️ Failed to verify hours for enrollment %s: %v", enrollment.ID.Hex(), err)
			errorCount++
		} else {
			successCount++
		}
	}

	log.Printf("✅ Processed %d enrollments successfully, %d errors", successCount, errorCount)
	return nil
}
