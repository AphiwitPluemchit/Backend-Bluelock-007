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

var activityStateCollection *mongo.Collection

func init() {
	// เชื่อมต่อกับ MongoDB
	if err := database.ConnectMongoDB(); err != nil {
		log.Fatal("MongoDB connection error:", err)
	}

	activityStateCollection = database.GetCollection("BluelockDB", "activityStates")
	if activityStateCollection == nil {
		log.Fatal("Failed to get the activityStates collection")
	}
}

// CreateActivityState - เพิ่มข้อมูลผู้ใช้ใน MongoDB
func CreateActivityState(activityState *models.ActivityState) error {
	activityState.ID = primitive.NewObjectID() // กำหนด ID อัตโนมัติ
	_, err := activityStateCollection.InsertOne(context.Background(), activityState)
	return err
}

// GetAllActivityStates - ดึงข้อมูลผู้ใช้ทั้งหมด
func GetAllActivityStates() ([]models.ActivityState, error) {
	var activityStates []models.ActivityState
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := activityStateCollection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var activityState models.ActivityState
		if err := cursor.Decode(&activityState); err != nil {
			return nil, err
		}
		activityStates = append(activityStates, activityState)
	}

	return activityStates, nil
}

// GetActivityStateByID - ดึงข้อมูลผู้ใช้ตาม ID
func GetActivityStateByID(id string) (*models.ActivityState, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid activityState ID")
	}

	var activityState models.ActivityState
	err = activityStateCollection.FindOne(context.Background(), bson.M{"_id": objID}).Decode(&activityState)
	if err != nil {
		return nil, err
	}

	return &activityState, nil
}

// UpdateActivityState - อัปเดตข้อมูลผู้ใช้
func UpdateActivityState(id string, activityState *models.ActivityState) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid activityState ID")
	}

	filter := bson.M{"_id": objID}
	update := bson.M{"$set": activityState}

	_, err = activityStateCollection.UpdateOne(context.Background(), filter, update)
	return err
}

// DeleteActivityState - ลบข้อมูลผู้ใช้
func DeleteActivityState(id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid activityState ID")
	}

	_, err = activityStateCollection.DeleteOne(context.Background(), bson.M{"_id": objID})
	return err
}
