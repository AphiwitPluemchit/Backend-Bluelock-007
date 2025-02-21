package services

import (
	"Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"context"
	"errors"
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
	if enrollmentCollection == nil {
		log.Fatal("Failed to get the enrollments collection")
	}
}

// CreateEnrollment - เพิ่มข้อมูลผู้ใช้ใน MongoDB
func CreateEnrollment(enrollment *models.Enrollment) error {
	enrollment.ID = primitive.NewObjectID() // กำหนด ID อัตโนมัติ
	_, err := enrollmentCollection.InsertOne(context.Background(), enrollment)
	return err
}

// GetAllEnrollments - ดึงข้อมูลผู้ใช้ทั้งหมด
func GetAllEnrollments() ([]models.Enrollment, error) {
	var enrollments []models.Enrollment
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := enrollmentCollection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var enrollment models.Enrollment
		if err := cursor.Decode(&enrollment); err != nil {
			return nil, err
		}
		enrollments = append(enrollments, enrollment)
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
