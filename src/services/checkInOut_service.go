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
	"go.mongodb.org/mongo-driver/mongo/options"
)

var checkInOutCollection *mongo.Collection

// var qrTokenCollection *mongo.Collection // ลบการใช้งาน MongoDB QRToken
var qrClaimCollection *mongo.Collection

func InitQRClaimTTLIndex() {
	qrClaimCollection = database.GetCollection("BluelockDB", "qr_claims")
	if qrClaimCollection == nil {
		log.Fatal("Failed to get the qr_claims collection")
	}
	indexModel := mongo.IndexModel{
		Keys:    bson.M{"expireAt": 1},
		Options: options.Index().SetExpireAfterSeconds(0),
	}
	_, err := qrClaimCollection.Indexes().CreateOne(context.TODO(), indexModel)
	if err != nil {
		log.Fatal("Failed to create TTL index for qr_claims:", err)
	}
}

func init() {
	if err := database.ConnectMongoDB(); err != nil {
		log.Fatal("MongoDB connection error:", err)
	}
	database.InitRedis()

	checkInOutCollection = database.GetCollection("BluelockDB", "checkInOuts")
	if checkInOutCollection == nil {
		log.Fatal("Failed to get the checkInOuts collection")
	}
	InitQRClaimTTLIndex()
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
	// meta
	metaKey := "qr_token_meta:" + token
	meta := map[string]string{
		"activityId": activityObjID.Hex(),
		"type":       qrType,
	}
	metaJson, _ := json.Marshal(meta)
	database.RedisClient.Set(database.RedisCtx, metaKey, metaJson, 1*time.Hour)
	// เตรียม key สำหรับ claim (ยังไม่ต้อง set ค่า แต่ reserve TTL 1 ชั่วโมง)
	claimKey := "qr_claimed:" + token
	database.RedisClient.Set(database.RedisCtx, claimKey, "", 1*time.Hour)
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
	var claim models.QRClaim
	err = qrClaimCollection.FindOne(ctx, bson.M{"token": token, "studentId": studentObjID, "expireAt": bson.M{"$gt": time.Now()}}).Decode(&claim)
	if err == nil {
		// เคย claim แล้ว และยังไม่หมดอายุ
		return &models.QRToken{
			Token:              claim.Token,
			ActivityID:         claim.ActivityID,
			Type:               claim.Type,
			ClaimedByStudentID: &studentObjID,
		}, nil
	}
	// 2. ถ้าไม่เจอ → ไป Redis (qr_token:{token})
	redisKey := "qr_token:" + token
	qrData, redisErr := database.RedisClient.Get(database.RedisCtx, redisKey).Result()
	if redisErr != nil {
		return nil, fmt.Errorf("QR token expired or invalid")
	}
	var qrMeta models.QRToken
	err = json.Unmarshal([]byte(qrData), &qrMeta)
	if err != nil {
		return nil, fmt.Errorf("QR token data invalid")
	}
	// เช็ค expiresAt
	if time.Now().Unix() > qrMeta.ExpiresAt {
		return nil, fmt.Errorf("QR token expired or invalid")
	}
	// upsert ลง qr_claims (หมดอายุใน 1 ชม. หลัง claim)
	expireAt := time.Now().Add(1 * time.Hour)
	filter := bson.M{"token": token, "studentId": studentObjID}
	update := bson.M{"$set": bson.M{
		"token":      token,
		"studentId":  studentObjID,
		"activityId": qrMeta.ActivityID,
		"type":       qrMeta.Type,
		"claimedAt":  time.Now(),
		"expireAt":   expireAt,
	}}
	_, err = qrClaimCollection.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	if err != nil {
		return nil, err
	}
	return &models.QRToken{
		Token:              token,
		ActivityID:         qrMeta.ActivityID,
		Type:               qrMeta.Type,
		ClaimedByStudentID: &studentObjID,
	}, nil
}

func ValidateQRToken(token, studentId string) (*models.QRToken, error) {
	ctx := context.TODO()
	studentObjID, err := primitive.ObjectIDFromHex(studentId)
	if err != nil {
		return nil, err
	}
	filter := bson.M{"token": token, "studentId": studentObjID, "expireAt": bson.M{"$gt": time.Now()}}
	var claim models.QRClaim
	err = qrClaimCollection.FindOne(ctx, filter).Decode(&claim)
	if err != nil {
		return nil, fmt.Errorf("QR Code นี้หมดอายุแล้ว Validate")
	}
	qrToken := &models.QRToken{
		Token:              token,
		ActivityID:         claim.ActivityID,
		Type:               claim.Type,
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

	switch checkType {
	case "checkin":
		// กัน checkin ซ้ำในวันเดียวกัน (มีแล้ว)
		filter := bson.M{
			"userId":         uID,
			"activityItemId": aID,
			"type":           "checkin",
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
			return fmt.Errorf("คุณได้เช็คชื่อเข้าแล้วในวันนี้")
		}
	case "checkout":
		// ต้องมี checkin ก่อนถึงจะ checkout ได้
		checkinFilter := bson.M{
			"userId":         uID,
			"activityItemId": aID,
			"type":           "checkin",
			"checkedAt": bson.M{
				"$gte": startOfDay,
				"$lt":  endOfDay,
			},
		}
		checkinCount, err := checkInOutCollection.CountDocuments(context.TODO(), checkinFilter)
		if err != nil {
			return err
		}
		if checkinCount == 0 {
			return fmt.Errorf("คุณต้องเช็คชื่อเข้าก่อนจึงจะเช็คชื่อออกได้")
		}
		// กัน checkout ซ้ำในวันเดียวกัน
		checkoutFilter := bson.M{
			"userId":         uID,
			"activityItemId": aID,
			"type":           "checkout",
			"checkedAt": bson.M{
				"$gte": startOfDay,
				"$lt":  endOfDay,
			},
		}
		checkoutCount, err := checkInOutCollection.CountDocuments(context.TODO(), checkoutFilter)
		if err != nil {
			return err
		}
		if checkoutCount > 0 {
			return fmt.Errorf("คุณได้เช็คชื่อออกแล้วในวันนี้")
		}
	}
	// Insert ใหม่
	_, err := checkInOutCollection.InsertOne(context.TODO(), bson.M{
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
		switch checkType {
		case "checkin":
			return fmt.Errorf("คุณไม่ได้ลงทะเบียนในกิจกรรมนี้ ไม่สามารถเช็คชื่อเข้าได้")
		case "checkout":
			return fmt.Errorf("คุณไม่ได้ลงทะเบียนในกิจกรรมนี้ ไม่สามารถเช็คชื่อออกได้")
		}
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
