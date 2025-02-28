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
var activityItemCollection *mongo.Collection

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
func CreateEnrollment(enrollment models.Enrollment) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// ตรวจสอบว่า ActivityItem มีอยู่จริง
	if !IsValidActivityItem(enrollment.ActivityItemID) {
		return errors.New("invalid activityItemId: not found in database")
	}

	// ตรวจสอบว่า Student มีอยู่จริง
	if !IsValidStudent(enrollment.StudentID) {
		return errors.New("invalid studentId: not found in database")
	}

	// ตรวจสอบว่า FoodVote มีอยู่จริง (ถ้ามี)
	if enrollment.FoodVoteID != nil && *enrollment.FoodVoteID != primitive.NilObjectID && !IsValidFoodVote(*enrollment.FoodVoteID) {
		return errors.New("invalid foodVoteId: not found in database")
	}

	enrollment.ID = primitive.NewObjectID()
	enrollment.RegistrationDate = time.Now()

	_, err := enrollmentCollection.InsertOne(ctx, enrollment)
	if err != nil {
		return errors.New("failed to create enrollment")
	}

	return nil
}

// ✅ GetAllEnrollments - ดึงข้อมูล Enrollment พร้อม ActivityItem และ Activity
func GetAllEnrollments() ([]bson.M, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pipeline := mongo.Pipeline{
		// Lookup ActivityItem
		bson.D{{
			Key: "$lookup", Value: bson.D{
				{Key: "from", Value: "activityItems"},
				{Key: "localField", Value: "activityItemId"},
				{Key: "foreignField", Value: "_id"},
				{Key: "as", Value: "activityItems"},
			},
		}},

		// Lookup Activity ทั้งหมด
		bson.D{{
			Key: "$lookup", Value: bson.D{
				{Key: "from", Value: "activities"},
				{Key: "localField", Value: "activityItems.activityId"},
				{Key: "foreignField", Value: "_id"},
				{Key: "as", Value: "activity"},
			},
		}},
		bson.D{{Key: "$unwind", Value: bson.D{{Key: "path", Value: "$activity"}, {Key: "preserveNullAndEmptyArrays", Value: true}}}},

		// Filter ให้ activityItems มีเฉพาะตัวที่ตรงกับ activityItemId
		bson.D{{
			Key: "$set", Value: bson.D{
				{Key: "activity.activityItems", Value: bson.D{
					{Key: "$filter", Value: bson.D{
						{Key: "input", Value: "$activity.activityItems"},
						{Key: "as", Value: "item"},
						{Key: "cond", Value: bson.D{
							{Key: "$eq", Value: bson.A{"$$item._id", "$activityItemId"}},
						}},
					}},
				}},
			},
		}},

		// Lookup Students (ดึงมาทั้งหมด)
		bson.D{{
			Key: "$lookup", Value: bson.D{
				{Key: "from", Value: "students"},
				{Key: "localField", Value: "studentId"},
				{Key: "foreignField", Value: "_id"},
				{Key: "as", Value: "student"},
			},
		}},

		// Lookup FoodVote (ดึงมาทั้งหมด)
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
				{Key: "activity", Value: 1}, // แสดง Activity ทั้งหมด
				{Key: "student", Value: 1},  // แสดง Student ทั้งหมด
				{Key: "foodVote", Value: 1}, // แสดง FoodVote ทั้งหมด
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
