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

// ✅ CreateEnrollment - ลงทะเบียนกิจกรรม พร้อมตรวจสอบข้อมูลก่อนลงทะเบียน
func RegisterActivityItem(foodVoteID, activityItemID, studentID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// ค้นหา Enrollment ที่ตรงกับ activityItemID
	var existingEnrollment models.Enrollment
	err := enrollmentCollection.FindOne(ctx, bson.M{"activityItemId": activityItemID}).Decode(&existingEnrollment)

	// ถ้ายังไม่มี ให้สร้างใหม่
	if err == mongo.ErrNoDocuments {
		newEnrollment := models.Enrollment{
			ID:               primitive.NewObjectID(),
			RegistrationDate: time.Now(),
			ActivityItemID:   activityItemID,
			StudentID:        []primitive.ObjectID{studentID},
		}

		_, err := enrollmentCollection.InsertOne(ctx, newEnrollment)
		if err != nil {
			return errors.New("failed to create new enrollment")
		}

		return nil
	} else if err != nil {
		return errors.New("database error while checking enrollment")
	}

	// ตรวจสอบว่า studentId และ foodVoteId มีอยู่ใน Array หรือไม่
	studentExists := false
	foodVoteExists := false

	for _, sID := range existingEnrollment.StudentID {
		if sID == studentID {
			studentExists = true
			break
		}
	}

	// เตรียม Update Fields
	updateFields := bson.M{}
	pushFields := bson.M{}

	if !studentExists {
		pushFields["studentId"] = studentID
	}
	if !foodVoteExists {
		pushFields["foodVoteId"] = foodVoteID
	}

	// ถ้ามีข้อมูลต้องอัปเดต ให้ดำเนินการ
	if len(pushFields) > 0 {
		updateFields["$push"] = pushFields
		_, err = enrollmentCollection.UpdateOne(ctx, bson.M{"activityItemId": activityItemID}, updateFields)
		if err != nil {
			return errors.New("failed to update enrollment")
		}
	}

	return nil
}

// GetEnrollmentsByStudent - ดึงข้อมูลกิจกรรมทั้งหมดที่นิสิตเข้าร่วม
func GetEnrollmentsByStudent(studentID primitive.ObjectID) ([]bson.M, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. ค้นหา Enrollment ทั้งหมดของนิสิต
	cursor, err := enrollmentCollection.Find(ctx, bson.M{"studentId": studentID})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch enrollments: %v", err)
	}
	defer cursor.Close(ctx)

	var results []bson.M

	// 2. วนลูปเพื่อประมวลผลแต่ละ Enrollment
	for cursor.Next(ctx) {
		var enrollment models.Enrollment
		if err := cursor.Decode(&enrollment); err != nil {
			return nil, fmt.Errorf("failed to decode enrollment: %v", err)
		}

		// 3. ดึง ActivityItem จาก activityItemId
		var activityItem models.ActivityItem
		err := activityItemCollection.FindOne(ctx, bson.M{"_id": enrollment.ActivityItemID}).Decode(&activityItem)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				// หากไม่พบ ActivityItem ให้ข้ามไปยัง Enrollment ถัดไป
				continue
			}
			return nil, fmt.Errorf("failed to fetch activity item: %v", err)
		}

		// 4. ดึง Activity และ ActivityItems จาก ActivityItem
		activity, activityItems, err := GetActivityByID(activityItem.ActivityID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch activity: %v", err)
		}

		// 5. สร้างโครงข้อมูลผลลัพธ์
		result := bson.M{
			"_id":              enrollment.ID,
			"registrationDate": enrollment.RegistrationDate,
			"studentId":        enrollment.StudentID[0], // ใช้ StudentID ตัวแรกใน array
			"activity": bson.M{
				"_id":             activity.ID,
				"name":            activity.Name,
				"type":            activity.Type,
				"adminId":         activity.AdminID,
				"activityStateId": activity.ActivityStateID,
				"skillId":         activity.SkillID,
				"majorIds":        activity.MajorIDs,
				"activityItems":   activityItems,
			},
		}

		results = append(results, result)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %v", err)
	}

	// หากไม่พบข้อมูล Enrollment ที่เกี่ยวข้อง
	if len(results) == 0 {
		return nil, fmt.Errorf("no enrollments found for the student")
	}

	return results, nil
}

// GetEnrollmentByStudentAndActivity - ดึงข้อมูลกิจกรรมที่นิสิตเลือก
func GetEnrollmentByStudentAndActivity(studentID, activityItemID primitive.ObjectID) (*bson.M, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. ค้นหา Enrollment ที่ตรงกับ studentId และ activityItemId
	var enrollment models.Enrollment
	err := enrollmentCollection.FindOne(ctx, bson.M{
		"studentId":      studentID,
		"activityItemId": activityItemID,
	}).Decode(&enrollment)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("enrollment not found")
		}
		return nil, fmt.Errorf("failed to fetch enrollment: %v", err)
	}

	// 2. ดึง ActivityItem จาก activityItemId
	var activityItem models.ActivityItem
	err = activityItemCollection.FindOne(ctx, bson.M{"activityItemId": enrollment.ActivityItemID}).Decode(&activityItem)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("activity item not found")
		}
		return nil, fmt.Errorf("failed to fetch activity item: %v", err)
	}

	// 3. ดึง Activity และ ActivityItems จาก ActivityItem
	activity, activityItems, err := GetActivityByID(activityItem.ActivityID)

	log.Println(activity)
	log.Println(activityItems)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch activity: %v", err)
	}

	// 4. สร้างโครงข้อมูลผลลัพธ์
	result := bson.M{
		"_id":              enrollment.ID,
		"registrationDate": enrollment.RegistrationDate,
		"studentId":        enrollment.StudentID[0], // ใช้ StudentID ตัวแรกใน array
		"activity": bson.M{
			"_id":             activity.ID,
			"name":            activity.Name,
			"type":            activity.Type,
			"adminId":         activity.AdminID,
			"activityStateId": activity.ActivityStateID,
			"skillId":         activity.SkillID,
			"majorIds":        activity.MajorIDs,
			"activityItems":   activityItems,
		},
	}

	return &result, nil
}

// GetAllEnrollments - ดึงข้อมูล Enrollment พร้อม ActivityItem และ Activity
func GetAllEnrollments() ([]bson.M, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pipeline := mongo.Pipeline{
		// Lookup ActivityItem (ดึงข้อมูลทั้งหมด)
		bson.D{{
			Key: "$lookup", Value: bson.D{
				{Key: "from", Value: "activityItems"},
				{Key: "localField", Value: "activityItemId"},
				{Key: "foreignField", Value: "_id"},
				{Key: "as", Value: "activityItems"},
			},
		}},

		// Lookup Activity (ดึงข้อมูลทั้งหมด)
		bson.D{{
			Key: "$lookup", Value: bson.D{
				{Key: "from", Value: "activities"},
				{Key: "localField", Value: "activityItemId"},
				{Key: "foreignField", Value: "_id"},
				{Key: "as", Value: "activity"},
			},
		}},
		bson.D{{
			Key: "$unwind", Value: bson.D{
				{Key: "path", Value: "$activity"},
				{Key: "preserveNullAndEmptyArrays", Value: true},
			},
		}},

		// Filter ให้ activityItems แสดงเฉพาะที่ตรงกับ activityItemId
		bson.D{{
			Key: "$set", Value: bson.D{
				{Key: "activity.activityItems", Value: bson.D{
					{Key: "$filter", Value: bson.D{
						{Key: "input", Value: "$activity.activityItems"},
						{Key: "as", Value: "item"},
						{Key: "cond", Value: bson.D{
							{Key: "$eq", Value: bson.A{"$$item._id", "$activityItemId"}},
						}},
					}}}},
			},
		}},

		// Lookup Students (ดึงข้อมูลทั้งหมด)
		bson.D{{
			Key: "$lookup", Value: bson.D{
				{Key: "from", Value: "students"},
				{Key: "localField", Value: "studentId"},
				{Key: "foreignField", Value: "_id"},
				{Key: "as", Value: "student"},
			},
		}},

		// Lookup FoodVote (ดึงข้อมูลทั้งหมด)
		bson.D{{
			Key: "$lookup", Value: bson.D{
				{Key: "from", Value: "foodVotes"},
				{Key: "localField", Value: "foodVoteId"},
				{Key: "foreignField", Value: "_id"},
				{Key: "as", Value: "foodVote"},
			},
		}},

		// เลือกเฉพาะข้อมูลที่ต้องการแสดง
		bson.D{{
			Key: "$project", Value: bson.D{
				{Key: "_id", Value: 1},
				{Key: "registrationDate", Value: 1},
				{Key: "activity", Value: 1}, // แสดงข้อมูลของ Activity
				{Key: "student", Value: 1},  // แสดงข้อมูลของ Student
				{Key: "foodVote", Value: 1}, // แสดงข้อมูลของ FoodVote
			},
		}},
	}

	cursor, err := enrollmentCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var enrollments []bson.M
	if err := cursor.All(ctx, &enrollments); err != nil {
		return nil, err
	}

	return enrollments, nil
}

// GetEnrollmentByID - ดึงข้อมูลผู้ใช้ตาม ID
func GetEnrollmentByID(id string) (*models.Enrollment, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid enrollment ID")
	}

	var enrollment models.Enrollment
	err = enrollmentCollection.FindOne(context.Background(), bson.M{"_id": objID}).Decode(&enrollment)
	if err != nil {
		return nil, err
	}
	fmt.Println(enrollment)
	return &enrollment, nil
}

// UpdateEnrollment - อัปเดตข้อมูลผู้ใช้
func UpdateEnrollment(id string, enrollment *models.Enrollment) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid enrollment ID")
	}

	filter := bson.M{"_id": objID}
	update := bson.M{"$set": enrollment}

	_, err = enrollmentCollection.UpdateOne(context.Background(), filter, update)
	return err
}

// DeleteEnrollment - ลบข้อมูลผู้ใช้
func DeleteEnrollment(id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid enrollment ID")
	}

	_, err = enrollmentCollection.DeleteOne(context.Background(), bson.M{"_id": objID})
	return err
}

// ✅ ฟังก์ชันตรวจสอบ Object ใน Database
func IsValidActivityItem(activityItemID primitive.ObjectID) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	count, err := activityItemCollection.CountDocuments(ctx, bson.M{"_id": activityItemID})
	return err == nil && count > 0
}

func IsValidStudent(studentID primitive.ObjectID) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	count, err := studentCollection.CountDocuments(ctx, bson.M{"_id": studentID})
	return err == nil && count > 0
}

func IsValidFoodVote(foodVoteID primitive.ObjectID) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	count, err := foodVoteCollection.CountDocuments(ctx, bson.M{"_id": foodVoteID})
	return err == nil && count > 0
}

// GetActivityByID - ดึงข้อมูล Activity พร้อม ActivityItems
func ActivityByID(activityID primitive.ObjectID) (models.Activity, []models.ActivityItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var activity models.Activity

	// ค้นหา Activity จาก ID
	err := activityCollection.FindOne(ctx, bson.M{"_id": activityID}).Decode(&activity)
	if err != nil {
		return models.Activity{}, nil, fmt.Errorf("activity not found")
	}

	// ดึง ActivityItems ที่เชื่อมโยง
	activityItems, err := ActivityItemsByActivityID(activityID)
	if err != nil {
		return models.Activity{}, nil, fmt.Errorf("failed to fetch activity items")
	}

	return activity, activityItems, nil
}

// GetActivityItemsByActivityID - ดึง ActivityItems ตาม ActivityID
func ActivityItemsByActivityID(activityID primitive.ObjectID) ([]models.ActivityItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var activityItems []models.ActivityItem

	// ค้นหา ActivityItems ทั้งหมดที่เชื่อมโยงกับ Activity
	cursor, err := activityItemCollection.Find(ctx, bson.M{"activityId": activityID})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch activity items")
	}
	defer cursor.Close(ctx)

	// อ่านผลลัพธ์
	if err := cursor.All(ctx, &activityItems); err != nil {
		return nil, fmt.Errorf("failed to decode activity items")
	}

	return activityItems, nil
}
