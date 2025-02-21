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

var activityItemCollection *mongo.Collection

func init() {
	// เชื่อมต่อกับ MongoDB
	if err := database.ConnectMongoDB(); err != nil {
		log.Fatal("MongoDB connection error:", err)
	}

	activityItemCollection = database.GetCollection("BluelockDB", "activityItems")
	if activityItemCollection == nil {
		log.Fatal("Failed to get the activityItems collection")
	}
}

// CreateActivityItem - เพิ่มข้อมูลผู้ใช้ใน MongoDB
func CreateActivityItem(activityItem *models.ActivityItem) error {
	activityItem.ID = primitive.NewObjectID() // กำหนด ID อัตโนมัติ
	_, err := activityItemCollection.InsertOne(context.Background(), activityItem)
	return err
}

// GetAllActivityItems - ดึงข้อมูลผู้ใช้ทั้งหมด
func GetAllActivityItems() ([]models.ActivityItem, error) {
	var activityItems []models.ActivityItem
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := activityItemCollection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var activityItem models.ActivityItem
		if err := cursor.Decode(&activityItem); err != nil {
			return nil, err
		}
		activityItems = append(activityItems, activityItem)
	}

	return activityItems, nil
}

// GetActivityItemByID - ดึงข้อมูลผู้ใช้ตาม ID
func GetActivityItemByID(id string) (*models.ActivityItem, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid activityItem ID")
	}

	var activityItem models.ActivityItem
	err = activityItemCollection.FindOne(context.Background(), bson.M{"_id": objID}).Decode(&activityItem)
	if err != nil {
		return nil, err
	}

	return &activityItem, nil
}

// UpdateActivityItem - อัปเดตข้อมูลผู้ใช้
func UpdateActivityItem(id string, activityItem *models.ActivityItem) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid activityItem ID")
	}

	filter := bson.M{"_id": objID}
	update := bson.M{"$set": activityItem}

	_, err = activityItemCollection.UpdateOne(context.Background(), filter, update)
	return err
}

// DeleteActivityItem - ลบข้อมูลผู้ใช้
func DeleteActivityItem(id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid activityItem ID")
	}

	_, err = activityItemCollection.DeleteOne(context.Background(), bson.M{"_id": objID})
	return err
}
