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
var activityItemCollection *mongo.Collection

func init() {
	// เชื่อมต่อ MongoDB
	if err := database.ConnectMongoDB(); err != nil {
		log.Fatal("MongoDB connection error:", err)
	}

	activityCollection = database.GetCollection("BluelockDB", "activitys")
	activityItemCollection = database.GetCollection("BluelockDB", "activityItems")

	if activityCollection == nil || activityItemCollection == nil {
		log.Fatal("Failed to get collections")
	}
}

// CreateActivity - เพิ่มกิจกรรมใหม่
func CreateActivity(activity models.Activity) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	activity.ID = primitive.NewObjectID()
	_, err := activityCollection.InsertOne(ctx, activity)
	if err != nil {
		return errors.New("failed to create activity")
	}

	if len(activity.ActivityItem) > 0 {
		var activityItems []interface{}
		for _, item := range activity.ActivityItem {
			item.ID = primitive.NewObjectID()
			item.ActivityID = activity.ID
			activityItems = append(activityItems, item)
		}

		_, err := activityItemCollection.InsertMany(ctx, activityItems)
		if err != nil {
			return errors.New("failed to create activity items")
		}
	}

	return nil
}

// GetAllActivitys - ดึงข้อมูลกิจกรรมทั้งหมด
func GetAllActivitys() ([]bson.M, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pipeline := mongo.Pipeline{
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "activityItems"},
			{Key: "localField", Value: "_id"},
			{Key: "foreignField", Value: "activityId"},
			{Key: "as", Value: "activityItems"},
		}}},
	}

	cursor, err := activityCollection.Aggregate(ctx, pipeline)
	if err != nil {
		log.Println("Error fetching activities:", err)
		return nil, err
	}

	var activities []bson.M
	if err := cursor.All(ctx, &activities); err != nil {
		log.Println("Error decoding activities:", err)
		return nil, err
	}

	return activities, nil
}

// GetActivityByID - ดึงข้อมูลกิจกรรมตาม ID
func GetActivityByID(id string) (bson.M, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid activity ID")
	}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "_id", Value: objID}}}},
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "activityItems"},
			{Key: "localField", Value: "_id"},
			{Key: "foreignField", Value: "activityId"},
			{Key: "as", Value: "activityItems"},
		}}},
	}

	cursor, err := activityCollection.Aggregate(ctx, pipeline)
	if err != nil {
		log.Println("Error fetching activity:", err)
		return nil, err
	}

	var activities []bson.M
	if err := cursor.All(ctx, &activities); err != nil {
		log.Println("Error decoding activity:", err)
		return nil, err
	}

	if len(activities) == 0 {
		return nil, errors.New("activity not found")
	}

	return activities[0], nil
}

// UpdateActivity - อัปเดตข้อมูลกิจกรรม
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

// DeleteActivity - ลบกิจกรรม
func DeleteActivity(id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid activity ID")
	}

	_, err = activityCollection.DeleteOne(context.Background(), bson.M{"_id": objID})
	if err != nil {
		return err
	}

	_, err = activityItemCollection.DeleteMany(context.Background(), bson.M{"activityId": objID})
	if err != nil {
		return err
	}

	return nil
}
