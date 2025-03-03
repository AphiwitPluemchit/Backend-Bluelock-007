package services

import (
	"Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var enrollmentCollection *mongo.Collection

func init() {
	// เชื่อมต่อกับ MongoDB
	if err := database.ConnectMongoDB(); err != nil {
		log.Fatal("MongoDB connection error:", err)
	}

	enrollmentCollection = database.GetCollection("BluelockDB", "enrollments")
	activityItemCollection = database.GetCollection("BluelockDB", "activityItems")
	studentCollection = database.GetCollection("BluelockDB", "students")
	foodVoteCollection = database.GetCollection("BluelockDB", "foodVotes")

	if enrollmentCollection == nil || activityItemCollection == nil || studentCollection == nil {
		log.Fatal("Failed to get necessary collections")
	}
}

// ✅ 1. Student ลงทะเบียนกิจกรรม (ลงซ้ำไม่ได้)
func RegisterStudent(activityItemID, studentID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// ตรวจสอบว่า Student ลงทะเบียนไปแล้วหรือยัง
	count, err := enrollmentCollection.CountDocuments(ctx, bson.M{
		"activityItemId": activityItemID,
		"studentId":      studentID,
	})
	if err != nil {
		return err
	}

	if count > 0 {
		return errors.New("already enrolled in this activity")
	}

	// สร้าง Enrollment ใหม่
	newEnrollment := models.Enrollment{
		ID:               primitive.NewObjectID(),
		StudentID:        studentID,
		ActivityItemID:   activityItemID,
		RegistrationDate: time.Now(),
	}

	_, err = enrollmentCollection.InsertOne(ctx, newEnrollment)
	return err
}

// ✅ 2. ดึงกิจกรรมทั้งหมดที่ Student ลงทะเบียนไปแล้ว
func GetEnrollmentsByStudent(studentID primitive.ObjectID) ([]bson.M, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 🔍 ตรวจสอบว่ามี Enrollment หรือไม่
	count, err := enrollmentCollection.CountDocuments(ctx, bson.M{"studentId": studentID})
	if err != nil {
		return nil, fmt.Errorf("database error: %v", err)
	}
	if count == 0 {
		return []bson.M{}, nil // ✅ คืนค่า `[]` แทน `null`
	}

	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.M{"studentId": studentID}}},
		bson.D{{Key: "$lookup", Value: bson.M{
			"from":         "activityItems",
			"localField":   "activityItemId",
			"foreignField": "_id",
			"as":           "activityItemDetails",
		}}},
		bson.D{{Key: "$unwind", Value: "$activityItemDetails"}},
		bson.D{{Key: "$lookup", Value: bson.M{
			"from":         "activitys",
			"localField":   "activityItemDetails.activityId",
			"foreignField": "_id",
			"as":           "activityDetails",
		}}},
		bson.D{{Key: "$unwind", Value: "$activityDetails"}},
		bson.D{{Key: "$project", Value: bson.M{
			"_id":              0,
			"id":               "$_id",
			"registrationDate": "$registrationDate",
			"studentId":        "$studentId",
			"activity": bson.M{
				"id":              "$activityDetails._id",
				"name":            "$activityDetails.name",
				"type":            "$activityDetails.type",
				"adminId":         "$activityDetails.adminId",
				"activityStateId": "$activityDetails.activityStateId",
				"skillId":         "$activityDetails.skillId",
				"majorIds":        "$activityDetails.majorIds",
				"activityItems": bson.M{
					"id":              "$activityItemDetails._id",
					"activityId":      "$activityItemDetails.activityId",
					"name":            "$activityItemDetails.name",
					"maxParticipants": "$activityItemDetails.maxParticipants",
					"description":     "$activityItemDetails.description",
					"room":            "$activityItemDetails.room",
					"startDate":       "$activityItemDetails.startDate",
					"endDate":         "$activityItemDetails.endDate",
					"duration":        "$activityItemDetails.duration",
					"operator":        "$activityItemDetails.operator",
					"hour":            "$activityItemDetails.hour",
				},
			},
		}}},
	}

	cursor, err := enrollmentCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("aggregation error: %v", err)
	}
	defer cursor.Close(ctx)

	var result []bson.M
	if err := cursor.All(ctx, &result); err != nil {
		return nil, fmt.Errorf("cursor error: %v", err)
	}

	return result, nil
}

// ✅ 3. ยกเลิกการลงทะเบียน
func UnregisterStudent(activityItemID, studentID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := enrollmentCollection.DeleteOne(ctx, bson.M{
		"activityItemId": activityItemID,
		"studentId":      studentID,
	})

	if err != nil {
		return err
	}

	if res.DeletedCount == 0 {
		return errors.New("no enrollment found")
	}

	return nil
}

// ✅ 4. Admin ดู Student ที่ลงทะเบียนในกิจกรรม พร้อมรายละเอียด
func GetStudentsByActivity(activityItemID primitive.ObjectID) (bson.M, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pipeline := mongo.Pipeline{
		// 1️⃣ Match เฉพาะ Enrollment ที่มี activityItemId ตรงกัน
		bson.D{{Key: "$match", Value: bson.M{"activityItemId": activityItemID}}},

		// 2️⃣ Lookup เชื่อม Student Collection
		bson.D{{
			Key: "$lookup", Value: bson.M{
				"from":         "students",
				"localField":   "studentId",
				"foreignField": "_id",
				"as":           "studentDetails",
			},
		}},

		// 3️⃣ Unwind Student ออกจาก Array
		bson.D{{Key: "$unwind", Value: "$studentDetails"}},

		// 4️⃣ Lookup เชื่อม Major Collection
		bson.D{{
			Key: "$lookup", Value: bson.M{
				"from":         "majors",
				"localField":   "studentDetails.majorId",
				"foreignField": "_id",
				"as":           "majorDetails",
			},
		}},

		// 5️⃣ Unwind Major ออกจาก Array (ถ้ามี)
		bson.D{{Key: "$unwind", Value: bson.M{"path": "$majorDetails", "preserveNullAndEmptyArrays": true}}},

		// 6️⃣ เปลี่ยนโครงสร้างผลลัพธ์
		bson.D{{
			Key: "$project", Value: bson.M{
				"activityItemId": "$activityItemId",
				"student": bson.M{
					"id":        "$studentDetails._id",
					"code":      "$studentDetails.code",
					"name":      "$studentDetails.name",
					"email":     "$studentDetails.email",
					"status":    "$studentDetails.status",
					"major":     "$majorDetails.majorName", // ✅ เอาชื่อ Major มาแทน majorId
					"softSkill": "$studentDetails.softSkill",
					"hardSkill": "$studentDetails.hardSkill",
				},
			},
		}},

		// 7️⃣ Group ข้อมูล Student เป็น Array
		bson.D{{
			Key: "$group", Value: bson.M{
				"_id":            "$activityItemId", // ✅ ใช้ `_id` เป็น activityItemId
				"activityItemId": bson.M{"$first": "$activityItemId"},
				"student":        bson.M{"$push": "$student"},
			},
		}},

		// 8️⃣ ลบ `_id` ออกจากผลลัพธ์
		bson.D{{Key: "$unset", Value: "_id"}},
	}

	cursor, err := enrollmentCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("aggregation error: %v", err)
	}
	defer cursor.Close(ctx)

	var result []bson.M
	if err := cursor.All(ctx, &result); err != nil {
		return nil, fmt.Errorf("cursor error: %v", err)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no enrollments found for this activity")
	}

	return result[0], nil
}

// ✅ 5. ดึงข้อมูลเฉพาะ Activity ที่ Student ลงทะเบียนไว้ (1 ตัว)
func GetEnrollmentByStudentAndActivity(studentID, activityItemID primitive.ObjectID) (*models.Enrollment, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var enrollment models.Enrollment
	err := enrollmentCollection.FindOne(ctx, bson.M{
		"studentId":      studentID,
		"activityItemId": activityItemID,
	}).Decode(&enrollment)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("enrollment not found")
		}
		return nil, err
	}

	return &enrollment, nil
}
