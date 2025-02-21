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

var activityCollection *mongo.Collection

func init() {
	// เชื่อมต่อกับ MongoDB
	if err := database.ConnectMongoDB(); err != nil {
		log.Fatal("MongoDB connection error:", err)
	}

	activityCollection = database.GetCollection("BluelockDB", "activitys")
	if activityCollection == nil {
		log.Fatal("Failed to get the activitys collection")
	}
}

// CreateActivity - เพิ่มข้อมูลผู้ใช้ใน MongoDB
func CreateActivity(activity *models.Activity) error {
	activity.ID = primitive.NewObjectID() // กำหนด ID อัตโนมัติ
	_, err := activityCollection.InsertOne(context.Background(), activity)
	return err
}

// GetAllActivitys - ดึงข้อมูลผู้ใช้ทั้งหมด
func GetAllActivitys() ([]models.Activity, error) {
	var activitys []models.Activity
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := activityCollection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var activity models.Activity
		if err := cursor.Decode(&activity); err != nil {
			return nil, err
		}
		activitys = append(activitys, activity)
	}

	return activitys, nil
}

// GetActivityByID - ดึงข้อมูลผู้ใช้ตาม ID
func GetActivityByID(id string) (*models.Activity, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid activity ID")
	}

	var activity models.Activity
	err = activityCollection.FindOne(context.Background(), bson.M{"_id": objID}).Decode(&activity)
	if err != nil {
		return nil, err
	}

	return &activity, nil
}

// UpdateActivity - อัปเดตข้อมูลผู้ใช้
func UpdateActivity(id string, activity *models.Activity) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid activity ID")
	}

	filter := bson.M{"_id": objID}
	update := bson.M{"$set": activity}

	_, err = activityCollection.UpdateOne(context.Background(), filter, update)
	return err
}

// DeleteActivity - ลบข้อมูลผู้ใช้
func DeleteActivity(id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid activity ID")
	}

	_, err = activityCollection.DeleteOne(context.Background(), bson.M{"_id": objID})
	return err
}
