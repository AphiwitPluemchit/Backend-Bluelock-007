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

var majorCollection *mongo.Collection

func init() {
	// เชื่อมต่อกับ MongoDB
	if err := database.ConnectMongoDB(); err != nil {
		log.Fatal("MongoDB connection error:", err)
	}

	majorCollection = database.GetCollection("BluelockDB", "majors")
	if majorCollection == nil {
		log.Fatal("Failed to get the majors collection")
	}
}

// CreateMajor - เพิ่มข้อมูลผู้ใช้ใน MongoDB
func CreateMajor(major *models.Major) error {
	major.ID = primitive.NewObjectID() // กำหนด ID อัตโนมัติ
	_, err := majorCollection.InsertOne(context.Background(), major)
	return err
}

// GetAllMajors - ดึงข้อมูลผู้ใช้ทั้งหมด
func GetAllMajors() ([]models.Major, error) {
	var majors []models.Major
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := majorCollection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var major models.Major
		if err := cursor.Decode(&major); err != nil {
			return nil, err
		}
		majors = append(majors, major)
	}

	return majors, nil
}

// GetMajorByID - ดึงข้อมูลผู้ใช้ตาม ID
func GetMajorByID(id string) (*models.Major, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid major ID")
	}

	var major models.Major
	err = majorCollection.FindOne(context.Background(), bson.M{"_id": objID}).Decode(&major)
	if err != nil {
		return nil, err
	}

	return &major, nil
}

// UpdateMajor - อัปเดตข้อมูลผู้ใช้
func UpdateMajor(id string, major *models.Major) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid major ID")
	}

	filter := bson.M{"_id": objID}
	update := bson.M{"$set": major}

	_, err = majorCollection.UpdateOne(context.Background(), filter, update)
	return err
}

// DeleteMajor - ลบข้อมูลผู้ใช้
func DeleteMajor(id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid major ID")
	}

	_, err = majorCollection.DeleteOne(context.Background(), bson.M{"_id": objID})
	return err
}
