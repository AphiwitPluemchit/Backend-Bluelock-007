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
	"go.mongodb.org/mongo-driver/mongo/options"
)

var checkInOutCollection *mongo.Collection
var qrTokenCollection *mongo.Collection
var qrClaimCollection *mongo.Collection

func init() {
	if err := database.ConnectMongoDB(); err != nil {
		log.Fatal("MongoDB connection error:", err)
	}
	database.InitRedis()

	checkInOutCollection = database.GetCollection("BluelockDB", "checkInOuts")
	qrTokenCollection = database.GetCollection("BluelockDB", "qr_tokens")
	qrClaimCollection = database.GetCollection("BluelockDB", "qr_claims")
	if checkInOutCollection == nil || qrTokenCollection == nil || qrClaimCollection == nil {
		log.Fatal("Failed to get the required collections")
	}
	// Ensure TTL index for qr_tokens (expiresAt)
	_, _ = qrTokenCollection.Indexes().CreateOne(context.TODO(), mongo.IndexModel{
		Keys:    bson.M{"expiresAt": 1},
		Options: options.Index().SetExpireAfterSeconds(0),
	})
	// Ensure TTL index for qr_claims (expireAt)
	_, _ = qrClaimCollection.Indexes().CreateOne(context.TODO(), mongo.IndexModel{
		Keys:    bson.M{"expireAt": 1},
		Options: options.Index().SetExpireAfterSeconds(0),
	})
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

// CreateQRToken creates a new QR token for an activityId, valid for 8 seconds
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
	_, err = qrTokenCollection.InsertOne(context.TODO(), qrToken)
	if err != nil {
		return "", 0, err
	}
	return token, expiresAt, nil
}

// ClaimQRToken allows a student to claim a QR token if not expired and not already claimed
func ClaimQRToken(token, studentId string) (*models.QRToken, error) {
	ctx := context.TODO()
	studentObjID, err := primitive.ObjectIDFromHex(studentId)
	if err != nil {
		return nil, err
	}
	// 1. หาใน qr_claims ก่อน (token+studentId+expireAt>now)
	var claim struct {
		Token      string             `bson:"token"`
		StudentID  primitive.ObjectID `bson:"studentId"`
		ActivityID primitive.ObjectID `bson:"activityId"`
		Type       string             `bson:"type"`
		ClaimedAt  time.Time          `bson:"claimedAt"`
		ExpireAt   time.Time          `bson:"expireAt"`
	}
	err = qrClaimCollection.FindOne(ctx, bson.M{"token": token, "studentId": studentObjID, "expireAt": bson.M{"$gt": time.Now()}}).Decode(&claim)
	if err == nil {
		return &models.QRToken{
			Token:              claim.Token,
			ActivityID:         claim.ActivityID,
			Type:               claim.Type,
			ClaimedByStudentID: &studentObjID,
		}, nil
	}
	// 2. ถ้าไม่เจอ → ไปหาใน qr_tokens (token+expiresAt>now)
	var qrToken models.QRToken
	err = qrTokenCollection.FindOne(ctx, bson.M{"token": token, "expiresAt": bson.M{"$gt": time.Now().Unix()}}).Decode(&qrToken)
	if err != nil {
		return nil, fmt.Errorf("QR token expired or invalid")
	}
	// upsert ลง qr_claims (หมดอายุใน 1 ชม. หลัง claim)
	expireAt := time.Now().Add(1 * time.Hour)
	claimDoc := bson.M{
		"token":      token,
		"studentId":  studentObjID,
		"activityId": qrToken.ActivityID,
		"type":       qrToken.Type,
		"claimedAt":  time.Now(),
		"expireAt":   expireAt,
	}
	_, err = qrClaimCollection.UpdateOne(ctx, bson.M{"token": token, "studentId": studentObjID}, bson.M{"$set": claimDoc}, options.Update().SetUpsert(true))
	if err != nil {
		return nil, err
	}
	qrToken.ClaimedByStudentID = &studentObjID
	return &qrToken, nil
}

// ValidateQRToken checks if the token is valid for the student (claimed and not expired)
func ValidateQRToken(token, studentId string) (*models.QRToken, error) {
	ctx := context.TODO()
	studentObjID, err := primitive.ObjectIDFromHex(studentId)
	if err != nil {
		return nil, err
	}
	var claim struct {
		Token      string             `bson:"token"`
		StudentID  primitive.ObjectID `bson:"studentId"`
		ActivityID primitive.ObjectID `bson:"activityId"`
		Type       string             `bson:"type"`
		ClaimedAt  time.Time          `bson:"claimedAt"`
		ExpireAt   time.Time          `bson:"expireAt"`
	}
	err = qrClaimCollection.FindOne(ctx, bson.M{"token": token, "studentId": studentObjID, "expireAt": bson.M{"$gt": time.Now()}}).Decode(&claim)
	if err != nil {
		return nil, fmt.Errorf("QR token not claimed or expired")
	}
	return &models.QRToken{
		Token:              claim.Token,
		ActivityID:         claim.ActivityID,
		Type:               claim.Type,
		ClaimedByStudentID: &studentObjID,
	}, nil
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
