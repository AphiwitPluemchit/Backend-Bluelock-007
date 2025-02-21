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

var checkInOutCollection *mongo.Collection

func init() {
	// เชื่อมต่อกับ MongoDB
	if err := database.ConnectMongoDB(); err != nil {
		log.Fatal("MongoDB connection error:", err)
	}

	checkInOutCollection = database.GetCollection("BluelockDB", "checkInOuts")
	if checkInOutCollection == nil {
		log.Fatal("Failed to get the checkInOuts collection")
	}
}

// CreateCheckInOut - เพิ่มข้อมูลผู้ใช้ใน MongoDB
func CreateCheckInOut(checkInOut *models.CheckInOut) error {
	checkInOut.ID = primitive.NewObjectID() // กำหนด ID อัตโนมัติ
	_, err := checkInOutCollection.InsertOne(context.Background(), checkInOut)
	return err
}

// GetAllCheckInOuts - ดึงข้อมูลผู้ใช้ทั้งหมด
func GetAllCheckInOuts() ([]models.CheckInOut, error) {
	var checkInOuts []models.CheckInOut
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := checkInOutCollection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var checkInOut models.CheckInOut
		if err := cursor.Decode(&checkInOut); err != nil {
			return nil, err
		}
		checkInOuts = append(checkInOuts, checkInOut)
	}

	return checkInOuts, nil
}

// GetCheckInOutByID - ดึงข้อมูลผู้ใช้ตาม ID
func GetCheckInOutByID(id string) (*models.CheckInOut, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid checkInOut ID")
	}

	var checkInOut models.CheckInOut
	err = checkInOutCollection.FindOne(context.Background(), bson.M{"_id": objID}).Decode(&checkInOut)
	if err != nil {
		return nil, err
	}

	return &checkInOut, nil
}

// UpdateCheckInOut - อัปเดตข้อมูลผู้ใช้
func UpdateCheckInOut(id string, checkInOut *models.CheckInOut) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid checkInOut ID")
	}

	filter := bson.M{"_id": objID}
	update := bson.M{"$set": checkInOut}

	_, err = checkInOutCollection.UpdateOne(context.Background(), filter, update)
	return err
}

// DeleteCheckInOut - ลบข้อมูลผู้ใช้
func DeleteCheckInOut(id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid checkInOut ID")
	}

	_, err = checkInOutCollection.DeleteOne(context.Background(), bson.M{"_id": objID})
	return err
}
