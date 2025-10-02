package enrollments

import (
	DB "Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GetCheckinStatus(studentId, programItemId string) ([]models.CheckinoutRecord, error) {
	uID, err1 := primitive.ObjectIDFromHex(studentId)
	aID, err2 := primitive.ObjectIDFromHex(programItemId)
	if err1 != nil || err2 != nil {
		return nil, fmt.Errorf("รหัสไม่ถูกต้อง")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var enrollment models.Enrollment
	if err := DB.EnrollmentCollection.FindOne(ctx, bson.M{"studentId": uID, "programItemId": aID}).Decode(&enrollment); err != nil {
		return []models.CheckinoutRecord{}, nil
	}
	if enrollment.CheckinoutRecord == nil {
		return []models.CheckinoutRecord{}, nil
	}
	return *enrollment.CheckinoutRecord, nil
}

// FindEnrolledItems คืน programItemIds ทั้งหมดที่นิสิตลงทะเบียนไว้ใน programId นี้
func FindEnrolledItems(userId string, programId string) ([]string, bool) {
	uID, _ := primitive.ObjectIDFromHex(userId)
	aID, _ := primitive.ObjectIDFromHex(programId)

	var enrolledItemIDs []string

	// 1. ดึง enrollments ทั้งหมดของ userId
	cursor, err := DB.EnrollmentCollection.Find(context.TODO(), bson.M{
		"studentId": uID, // หรือ "userId" ถ้าคุณใช้ชื่อนี้
	})
	if err != nil {
		return nil, false
	}
	defer cursor.Close(context.TODO())

	// 2. เช็กแต่ละรายการว่า programItemId → programId ตรงหรือไม่
	for cursor.Next(context.TODO()) {
		var enrollment models.Enrollment
		if err := cursor.Decode(&enrollment); err != nil {
			continue
		}

		var item models.ProgramItem
		err := DB.ProgramItemCollection.FindOne(context.TODO(), bson.M{
			"_id": enrollment.ProgramItemID,
		}).Decode(&item)
		if err == nil && item.ProgramID == aID {
			enrolledItemIDs = append(enrolledItemIDs, enrollment.ProgramItemID.Hex())
		}
	}

	if len(enrolledItemIDs) == 0 {
		return nil, false
	}
	return enrolledItemIDs, true
}

func IsStudentEnrolled(studentId string, programItemId string) bool {
	sID, err1 := primitive.ObjectIDFromHex(studentId)
	aID, err2 := primitive.ObjectIDFromHex(programItemId)

	if err1 != nil || err2 != nil {
		log.Printf("Invalid ObjectID: studentId=%s, programItemId=%s", studentId, programItemId)
		return false
	}

	filter := bson.M{
		"studentId":     sID,
		"programItemId": aID,
	}

	count, err := DB.EnrollmentCollection.CountDocuments(context.TODO(), filter)
	if err != nil {
		log.Printf("MongoDB error when checking enrollment: %v", err)
		return false
	}

	return count > 0
}

// FindEnrolledItem คืน programItemId ที่นิสิตลงทะเบียนไว้ใน programId นี้
func FindEnrolledItem(userId string, programId string) (string, bool) {
	uID, _ := primitive.ObjectIDFromHex(userId)
	aID, _ := primitive.ObjectIDFromHex(programId)

	// 1. ดึง enrollments ทั้งหมดของ userId
	cursor, err := DB.EnrollmentCollection.Find(context.TODO(), bson.M{
		"studentId": uID, // หรือ "userId" ถ้าคุณใช้ชื่อนี้
	})
	if err != nil {
		return "", false
	}
	defer cursor.Close(context.TODO())

	// 2. เช็กแต่ละรายการว่า programItemId → programId ตรงหรือไม่
	for cursor.Next(context.TODO()) {
		var enrollment models.Enrollment
		if err := cursor.Decode(&enrollment); err != nil {
			continue
		}

		var item models.ProgramItem
		err := DB.ProgramItemCollection.FindOne(context.TODO(), bson.M{
			"_id": enrollment.ProgramItemID,
		}).Decode(&item)
		if err == nil && item.ProgramID == aID {
			return enrollment.ProgramItemID.Hex(), true
		}
	}

	return "", false
}

func isTimeOverlap(start1, end1, start2, end2 string) bool {
	// ตัวอย่าง: 09:00 < 10:00 -> true (มีเวลาทับซ้อน)
	return !(end1 <= start2 || end2 <= start1)
}
