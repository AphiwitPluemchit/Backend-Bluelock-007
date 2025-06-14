package services

import (
	"Backend-Bluelock-007/src/database"
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

	checkInOutCollection = database.GetCollection("BluelockDB", "checkInOuts")
	if checkInOutCollection == nil {
		log.Fatal("Failed to get the checkInOuts collection")
	}
}

func GenerateCheckinUUID(activityItemId string, checkType string, userId string) (string, error) {
	id := uuid.NewString()
	key := fmt.Sprintf("checkin:%s", id)

	data := map[string]string{
		"activityItemId": activityItemId,
		"type":           checkType,
		"lockedUserId":   userId,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	err = database.RedisClient.Set(database.RedisCtx, key, jsonData, 1000*time.Second).Err()
	if err != nil {
		return "", err
	}

	return id, nil
}

func Checkin(uuid, userId string) (bool, string) {
	key := fmt.Sprintf("checkin:%s", uuid)

	val, err := database.RedisClient.Get(database.RedisCtx, key).Result()
	if err != nil {
		return false, "QR code หมดอายุหรือไม่ถูกต้อง"
	}

	var data struct {
		ActivityItemId string `json:"activityItemId"`
		Type           string `json:"type"`
		LockedUserId   string `json:"lockedUserId"`
	}
	if err := json.Unmarshal([]byte(val), &data); err != nil {
		return false, "ข้อมูลใน QR ไม่ถูกต้อง"
	}
	println(data.LockedUserId, userId)
	if data.LockedUserId != userId {
		return false, "QR นี้ไม่ใช่ของคุณ หรือถูกแชร์ให้ผู้อื่น"
	}

	// Convert IDs
	uID, err1 := primitive.ObjectIDFromHex(userId)
	aID, err2 := primitive.ObjectIDFromHex(data.ActivityItemId)
	if err1 != nil || err2 != nil {
		return false, "รหัสไม่ถูกต้อง"
	}

	// 🔒 ป้องกันเช็คชื่อซ้ำ
	filter := bson.M{
		"userId":         uID,
		"activityItemId": aID,
		"type":           data.Type,
	}
	count, _ := checkInOutCollection.CountDocuments(context.TODO(), filter)
	if count > 0 {
		return false, fmt.Sprintf("คุณได้ %s แล้ว", data.Type)
	}

	// ✅ บันทึกการเช็คชื่อ
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
