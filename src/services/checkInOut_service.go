package services

import (
	"Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/services/enrollments"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var checkInOutCollection *mongo.Collection

func init() {
	if err := database.ConnectMongoDB(); err != nil {
		log.Fatal("MongoDB connection error:", err)
	}
	database.InitRedis()

	checkInOutCollection = database.GetCollection("BluelockDB", "checkInOuts")
	if checkInOutCollection == nil {
		log.Fatal("Failed to get the checkInOuts collection")
	}
}

func GenerateCheckinUUID(activityId string, checkType string) (string, error) {
	id := uuid.NewString()
	key := fmt.Sprintf("checkin:%s", id)

	data := map[string]string{
		"activityId": activityId, // ✅ เปลี่ยนตรงนี้
		"type":       checkType,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	err = database.RedisClient.Set(database.RedisCtx, key, jsonData, 10*time.Second).Err()
	if err != nil {
		return "", err
	}

	return id, nil
}
func Checkin(uuid, userId string) (bool, string) {
	key := fmt.Sprintf("checkin:%s", uuid)

	val, err := database.RedisClient.Get(database.RedisCtx, key).Result()
	fmt.Println("Redis Value:", val)

	if err != nil {
		return false, "QR code หมดอายุหรือไม่ถูกต้อง"
	}

	var data struct {
		ActivityId string `json:"activityId"` // 🔄 เปลี่ยนจาก ActivityItemId
		Type       string `json:"type"`
	}
	if err := json.Unmarshal([]byte(val), &data); err != nil {
		return false, "ข้อมูลใน QR ไม่ถูกต้อง"
	}
	fmt.Println("data.ActivityId:", data.ActivityId)
	fmt.Println("userId:", userId)

	// ✅ ดึง activityItemId ที่นิสิตลงทะเบียนไว้ โดย matching กับ activityId
	enrolledItemID, found := enrollments.FindEnrolledItem(userId, data.ActivityId)
	if !found {
		return false, "คุณยังไม่ได้ลงทะเบียนกิจกรรมนี้"
	}

	// ✅ แปลง ObjectID
	uID, err1 := primitive.ObjectIDFromHex(userId)
	aID, err2 := primitive.ObjectIDFromHex(enrolledItemID)
	if err1 != nil || err2 != nil {
		return false, "รหัสไม่ถูกต้อง"
	}

	// 🔁 ป้องกันการเช็คชื่อซ้ำใน type เดียวกัน
	filter := bson.M{
		"userId":         uID,
		"activityItemId": aID,
		"type":           data.Type,
	}
	count, _ := checkInOutCollection.CountDocuments(context.TODO(), filter)
	if count > 0 {
		return false, fmt.Sprintf("คุณได้ %s แล้ว", data.Type)
	}

	// ✅ บันทึกเวลาที่เช็คชื่อ
	_, err = checkInOutCollection.InsertOne(context.TODO(), bson.M{
		"userId":         uID,
		"activityItemId": aID,
		"type":           data.Type,
		"checkedAt":      time.Now(),
	})
	if err != nil {
		return false, "ไม่สามารถบันทึกข้อมูลได้"
	}

	return true, fmt.Sprintf("%s สำเร็จ", data.Type)
}
