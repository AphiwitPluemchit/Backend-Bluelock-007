package services

import (
	"Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/services/enrollments"
	"context"
	"fmt"
	"log"
	"time"

	"Backend-Bluelock-007/src/models"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var checkInOutCollection *mongo.Collection
var qrTokenCollection *mongo.Collection

func init() {
	if err := database.ConnectMongoDB(); err != nil {
		log.Fatal("MongoDB connection error:", err)
	}
	database.InitRedis()

	checkInOutCollection = database.GetCollection("BluelockDB", "checkInOuts")
	if checkInOutCollection == nil {
		log.Fatal("Failed to get the checkInOuts collection")
	}
	// New collections for QR system
	qrTokenCollection = database.GetCollection("BluelockDB", "qr_tokens")
}

// func GenerateCheckinUUID(activityId string, checkType string) (string, error) {
// 	id := uuid.NewString()
// 	key := fmt.Sprintf("checkin:%s", id)

// 	data := map[string]string{
// 		"activityId": activityId, // ✅ เปลี่ยนตรงนี้
// 		"type":       checkType,
// 	}

// 	jsonData, err := json.Marshal(data)
// 	if err != nil {
// 		return "", err
// 	}

// 	err = database.RedisClient.Set(database.RedisCtx, key, jsonData, 1000*time.Second).Err()
// 	if err != nil {
// 		return "", err
// 	}

// 	return id, nil
// }
// func Checkin(uuid, userId string) (bool, string) {
// 	key := fmt.Sprintf("checkin:%s", uuid)
// 	val, err := database.RedisClient.Get(database.RedisCtx, key).Result()
// 	fmt.Println("Redis Value:", val)

// 	if err != nil {
// 		return false, "QR code หมดอายุหรือไม่ถูกต้อง"
// 	}

// 	var data struct {
// 		ActivityId string `json:"activityId"`
// 		Type       string `json:"type"` // checkin หรือ checkout
// 	}
// 	if err := json.Unmarshal([]byte(val), &data); err != nil {
// 		return false, "ข้อมูล QR ไม่ถูกต้อง"
// 	}

// 	enrolledItemID, found := enrollments.FindEnrolledItem(userId, data.ActivityId)
// 	if !found {
// 		return false, "คุณยังไม่ได้ลงทะเบียนกิจกรรมนี้"
// 	}

// 	// Convert ObjectID
// 	uID, err1 := primitive.ObjectIDFromHex(userId)
// 	aID, err2 := primitive.ObjectIDFromHex(enrolledItemID)
// 	if err1 != nil || err2 != nil {
// 		return false, "รหัสไม่ถูกต้อง"
// 	}

// 	// ป้องกันเช็คชื่อซ้ำ
// 	filter := bson.M{
// 		"userId":         uID,
// 		"activityItemId": aID,
// 		"type":           data.Type,
// 	}
// 	count, _ := checkInOutCollection.CountDocuments(context.TODO(), filter)
// 	if count > 0 {
// 		return false, fmt.Sprintf("คุณได้ %s แล้ว", data.Type)
// 	}

// 	// ✅ Insert
// 	_, err = checkInOutCollection.InsertOne(context.TODO(), bson.M{
// 		"userId":         uID,
// 		"activityItemId": aID,
// 		"type":           data.Type,
// 		"checkedAt":      time.Now(),
// 	})
// 	if err != nil {
// 		return false, "ไม่สามารถบันทึกข้อมูลได้"
// 	}

// 	return true, fmt.Sprintf("%s สำเร็จ", data.Type)
// }

// func Checkout(uuid, userId, evaluationId string) (bool, string) {
// 	key := fmt.Sprintf("checkin:%s", uuid)

// 	val, err := database.RedisClient.Get(database.RedisCtx, key).Result()
// 	fmt.Println("Redis Value:", val)

// 	if err != nil {
// 		return false, "QR code หมดอายุหรือไม่ถูกต้อง"
// 	}

// 	var data struct {
// 		ActivityId string `json:"activityId"` // 🔄 เปลี่ยนจาก ActivityItemId
// 		Type       string `json:"type"`
// 	}
// 	if err := json.Unmarshal([]byte(val), &data); err != nil {
// 		return false, "ข้อมูลใน QR ไม่ถูกต้อง"
// 	}
// 	fmt.Println("data.ActivityId:", data.ActivityId)
// 	fmt.Println("userId:", userId)

// 	// ✅ ดึง activityItemId ที่นิสิตลงทะเบียนไว้ โดย matching กับ activityId
// 	enrolledItemID, found := enrollments.FindEnrolledItem(userId, data.ActivityId)
// 	if !found {
// 		return false, "คุณยังไม่ได้ลงทะเบียนกิจกรรมนี้"
// 	}

// 	// ✅ แปลง ObjectID
// 	uID, err1 := primitive.ObjectIDFromHex(userId)
// 	aID, err2 := primitive.ObjectIDFromHex(enrolledItemID)
// 	if err1 != nil || err2 != nil {
// 		return false, "รหัสไม่ถูกต้อง"
// 	}

// 	// 🔁 ป้องกันการเช็คชื่อซ้ำใน type เดียวกัน
// 	filter := bson.M{
// 		"userId":         uID,
// 		"activityItemId": aID,
// 		"type":           data.Type,
// 	}
// 	count, _ := checkInOutCollection.CountDocuments(context.TODO(), filter)
// 	if count > 0 {
// 		return false, fmt.Sprintf("คุณได้ %s แล้ว", data.Type)
// 	}

// 	// ✅ บันทึกเวลาที่เช็คชื่อ
// 	_, err = checkInOutCollection.InsertOne(context.TODO(), bson.M{
// 		"userId":         uID,
// 		"activityItemId": aID,
// 		"type":           data.Type,
// 		"checkedAt":      time.Now(),
// 		"evaluationId":   evaluationId, // ✅ เพิ่มตรงนี้เท่านั้น
// 	})

// 	if err != nil {
// 		return false, "ไม่สามารถบันทึกข้อมูลได้"
// 	}

//		return true, fmt.Sprintf("%s สำเร็จ", data.Type)
//	}
//
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
		if r.Type == "checkin" {
			checkins = append(checkins, t)
		} else if r.Type == "checkout" {
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
	expiresAt := now + 5
	qrToken := models.QRToken{
		Token:      token,
		ActivityID: activityObjID,
		Type:       qrType,
		CreatedAt:  now,
		ExpiresAt:  expiresAt,
	}
	_, err = qrTokenCollection.InsertOne(context.TODO(), qrToken)
	if err != nil {
		return "", 0, err
	}
	return token, expiresAt, nil
}

// ClaimQRToken allows a student to claim a QR token if not expired and not already claimed
func ClaimQRToken(token, studentId string) (*models.QRToken, error) {
	studentObjID, err := primitive.ObjectIDFromHex(studentId)
	if err != nil {
		return nil, err
	}
	var qrToken models.QRToken
	err = qrTokenCollection.FindOne(context.TODO(), bson.M{"token": token}).Decode(&qrToken)
	if err != nil {
		return nil, err
	}
	now := time.Now().Unix()
	if now > qrToken.ExpiresAt && qrToken.ClaimedByStudentID == nil {
		return nil, fmt.Errorf("QR token expired")
	}
	if qrToken.ClaimedByStudentID == nil {
		// Claim it
		_, err := qrTokenCollection.UpdateOne(context.TODO(), bson.M{"token": token}, bson.M{"$set": bson.M{"claimedByStudentId": studentObjID}})
		if err != nil {
			return nil, err
		}
		qrToken.ClaimedByStudentID = &studentObjID
	} else if qrToken.ClaimedByStudentID.Hex() != studentObjID.Hex() {
		return nil, fmt.Errorf("QR token already claimed by another student")
	}
	return &qrToken, nil
}

// ValidateQRToken checks if the token is valid for the student (claimed or claimable)
func ValidateQRToken(token, studentId string) (*models.QRToken, error) {
	studentObjID, err := primitive.ObjectIDFromHex(studentId)
	if err != nil {
		return nil, err
	}
	var qrToken models.QRToken
	err = qrTokenCollection.FindOne(context.TODO(), bson.M{"token": token}).Decode(&qrToken)
	if err != nil {
		return nil, err
	}
	if qrToken.ClaimedByStudentID == nil {
		return nil, fmt.Errorf("QR token not claimed yet")
	}
	if qrToken.ClaimedByStudentID.Hex() != studentObjID.Hex() {
		return nil, fmt.Errorf("QR token claimed by another student")
	}
	return &qrToken, nil
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
