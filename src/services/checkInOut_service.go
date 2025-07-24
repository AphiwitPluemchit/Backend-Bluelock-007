package services

import (
	"Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/services/enrollments"
	"context"
	"fmt"
	"log"
	"time"

	"Backend-Bluelock-007/src/models"

	"encoding/json"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var checkInOutCollection *mongo.Collection

// var qrTokenCollection *mongo.Collection // ลบการใช้งาน MongoDB QRToken

func init() {
	if err := database.ConnectMongoDB(); err != nil {
		log.Fatal("MongoDB connection error:", err)
	}
	database.InitRedis()

	checkInOutCollection = database.GetCollection("BluelockDB", "checkInOuts")
	if checkInOutCollection == nil {
		log.Fatal("Failed to get the checkInOuts collection")
	}
	// qrTokenCollection = database.GetCollection("BluelockDB", "qr_tokens") // ไม่ใช้แล้ว
}

// GetCheckinStatus returns all check-in/out records for a student and activityItemId
func GetCheckinStatus(studentId, activityItemId string) ([]map[string]interface{}, error) {
	uID, err1 := primitive.ObjectIDFromHex(studentId)
	aID, err2 := primitive.ObjectIDFromHex(activityItemId)
	if err1 != nil || err2 != nil {
		return nil, fmt.Errorf("รหัสไม่ถูกต้อง")
	}

	filter := bson.M{
		"userId":         uID,
		"activityItemId": aID,
	}

	cursor, err := checkInOutCollection.Find(context.TODO(), filter)
	if err != nil {
		return nil, fmt.Errorf("ไม่สามารถค้นหาข้อมูลเช็คชื่อได้")
	}
	defer cursor.Close(context.TODO())

	loc, _ := time.LoadLocation("Asia/Bangkok")

	// แยก checkin/checkout ตามวัน
	type rec struct {
		Type      string    `bson:"type"`
		CheckedAt time.Time `bson:"checkedAt"`
	}
	var checkins, checkouts []time.Time
	for cursor.Next(context.TODO()) {
		var r rec
		if err := cursor.Decode(&r); err != nil {
			continue
		}
		t := r.CheckedAt.In(loc)
		switch r.Type {
		case "checkin":
			checkins = append(checkins, t)
		case "checkout":
			checkouts = append(checkouts, t)
		}
	}

	// จับคู่ checkin/checkout ตามลำดับเวลา
	var results []map[string]interface{}
	usedCheckout := make([]bool, len(checkouts))
	for _, ci := range checkins {
		// หา checkout ที่เร็วที่สุดหลัง checkin นี้
		var co *time.Time
		for i, c := range checkouts {
			if !usedCheckout[i] && c.After(ci) {
				co = &c
				usedCheckout[i] = true
				break
			}
		}
		result := map[string]interface{}{
			"checkin": ci,
		}
		if co != nil {
			result["checkout"] = *co
		}
		results = append(results, result)
	}
	return results, nil
}

// CreateQRToken creates a new QR token for an activityId, valid for 5 seconds
func CreateQRToken(activityId string, qrType string) (string, int64, error) {
	token := uuid.NewString()
	activityObjID, err := primitive.ObjectIDFromHex(activityId)
	if err != nil {
		return "", 0, err
	}
	now := time.Now().Unix()
	expiresAt := now + 8 // 8 วินาที
	qrToken := models.QRToken{
		Token:      token,
		ActivityID: activityObjID,
		Type:       qrType,
		CreatedAt:  now,
		ExpiresAt:  expiresAt,
	}
	jsonData, err := json.Marshal(qrToken)
	if err != nil {
		return "", 0, err
	}
	key := "qr_token:" + token
	err = database.RedisClient.Set(database.RedisCtx, key, jsonData, 5*time.Second).Err()
	if err != nil {
		return "", 0, err
	}
	// เตรียม key สำหรับ claim (ยังไม่ต้อง set ค่า แต่ reserve TTL 1 ชั่วโมง)
	claimKey := "qr_claimed:" + token
	database.RedisClient.Set(database.RedisCtx, claimKey, "", 1*time.Hour)
	return token, expiresAt, nil
}

// ClaimQRToken allows a student to claim a QR token if not expired and not already claimed
func ClaimQRToken(token, studentId string) (*models.QRToken, error) {
	studentObjID, err := primitive.ObjectIDFromHex(studentId)
	if err != nil {
		return nil, err
	}
	key := "qr_token:" + token
	val, err := database.RedisClient.Get(database.RedisCtx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("QR token expired or invalid")
	}
	var qrToken models.QRToken
	if err := json.Unmarshal([]byte(val), &qrToken); err != nil {
		return nil, err
	}
	// ตรวจสอบว่าเคย claim หรือยัง
	claimKey := "qr_claimed:" + token
	claimVal, _ := database.RedisClient.Get(database.RedisCtx, claimKey).Result()
	if claimVal != "" {
		// มีคน claim ไปแล้ว
		var claimed struct {
			StudentID  string `json:"studentId"`
			ActivityID string `json:"activityId"`
			Type       string `json:"type"`
		}
		_ = json.Unmarshal([]byte(claimVal), &claimed)
		if claimed.StudentID != studentObjID.Hex() {
			return nil, fmt.Errorf("QR token already claimed by another student")
		}
		// ถ้าเป็นคนเดิม ให้คืนข้อมูลเดิม
		qrToken.ClaimedByStudentID = &studentObjID
		return &qrToken, nil
	}
	// ยังไม่เคย claim ให้บันทึกลง qr_claimed:<token>
	claimData := struct {
		StudentID  string `json:"studentId"`
		ActivityID string `json:"activityId"`
		Type       string `json:"type"`
	}{
		StudentID:  studentObjID.Hex(),
		ActivityID: qrToken.ActivityID.Hex(),
		Type:       qrToken.Type,
	}
	claimJson, _ := json.Marshal(claimData)
	database.RedisClient.Set(database.RedisCtx, claimKey, claimJson, 1*time.Hour)
	qrToken.ClaimedByStudentID = &studentObjID
	return &qrToken, nil
}

// ValidateQRToken checks if the token is valid for the student (claimed or claimable)
func ValidateQRToken(token, studentId string) (*models.QRToken, error) {
	studentObjID, err := primitive.ObjectIDFromHex(studentId)
	if err != nil {
		return nil, err
	}
	claimKey := "qr_claimed:" + token
	claimVal, err := database.RedisClient.Get(database.RedisCtx, claimKey).Result()
	if err != nil || claimVal == "" {
		return nil, fmt.Errorf("QR token not claimed or expired")
	}
	var claimed struct {
		StudentID  string `json:"studentId"`
		ActivityID string `json:"activityId"`
		Type       string `json:"type"`
	}
	if err := json.Unmarshal([]byte(claimVal), &claimed); err != nil {
		return nil, err
	}
	if claimed.StudentID != studentObjID.Hex() {
		return nil, fmt.Errorf("QR token claimed by another student")
	}
	// สร้าง QRToken struct สำหรับคืนค่า (ข้อมูลจาก claim)
	activityObjID, _ := primitive.ObjectIDFromHex(claimed.ActivityID)
	qrToken := &models.QRToken{
		Token:              token,
		ActivityID:         activityObjID,
		Type:               claimed.Type,
		ClaimedByStudentID: &studentObjID,
	}
	return qrToken, nil
}

// SaveCheckInOut saves a check-in/out for a specific activityItemId, prevents duplicate in the same day
func SaveCheckInOut(userId, activityItemId, checkType string) error {
	uID, err1 := primitive.ObjectIDFromHex(userId)
	aID, err2 := primitive.ObjectIDFromHex(activityItemId)
	if err1 != nil || err2 != nil {
		return fmt.Errorf("รหัสไม่ถูกต้อง")
	}
	// หาวันนี้ (ตัดเวลา)
	now := time.Now()
	y, m, d := now.Date()
	loc := now.Location()
	startOfDay := time.Date(y, m, d, 0, 0, 0, 0, loc)
	endOfDay := startOfDay.Add(24 * time.Hour)
	// เช็คว่ามี record ซ้ำในวันเดียวกันหรือยัง
	filter := bson.M{
		"userId":         uID,
		"activityItemId": aID,
		"type":           checkType,
		"checkedAt": bson.M{
			"$gte": startOfDay,
			"$lt":  endOfDay,
		},
	}
	count, err := checkInOutCollection.CountDocuments(context.TODO(), filter)
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("คุณได้เช็คชื่อ %s แล้วในวันนี้", checkType)
	}
	// Insert ใหม่
	_, err = checkInOutCollection.InsertOne(context.TODO(), bson.M{
		"userId":         uID,
		"activityItemId": aID,
		"type":           checkType,
		"checkedAt":      now,
	})
	return err
}

// RecordCheckin records a check-in or check-out for a student for all enrolled items in an activity
func RecordCheckin(studentId, activityId, checkType string) error {
	// ดึง activityItemIds ทั้งหมดที่นิสิตลงทะเบียนใน activity นี้
	itemIDs, found := enrollments.FindEnrolledItems(studentId, activityId)
	if !found || len(itemIDs) == 0 {
		return fmt.Errorf("not enrolled in this activity")
	}
	for _, itemID := range itemIDs {
		err := SaveCheckInOut(studentId, itemID, checkType)
		if err != nil {
			return err
		}
	}
	return nil
}
