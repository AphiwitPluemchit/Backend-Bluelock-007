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

var ctx = context.Background() // กำหนด context สำหรับ MongoDB

// ตัวแปรสำหรับเชื่อมต่อกับ MongoDB collection
var activityCollection *mongo.Collection

func init() {
	// เชื่อมต่อกับ MongoDB
	if err := database.ConnectMongoDB(); err != nil {
		log.Fatal("MongoDB connection error:", err)
	}

	// รับ MongoDB collection ที่ต้องการใช้งาน
	activityCollection = database.GetCollection("BluelockDB", "activitys")
	if activityCollection == nil {
		log.Fatal("Failed to get the activity collection")
	}
}

// CreateActivity - สร้างกิจกรรมใหม่
func CreateActivity(activity models.Activity) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// แปลง string เป็น ObjectID
	adminID, err := primitive.ObjectIDFromHex(activity.AdminID)
	if err != nil {
		return errors.New("invalid adminId")
	}
	activityStateID, err := primitive.ObjectIDFromHex(activity.ActivityStateID)
	if err != nil {
		return errors.New("invalid activityStateId")
	}
	skillID, err := primitive.ObjectIDFromHex(activity.SkillID)
	if err != nil {
		return errors.New("invalid skillId")
	}

	var majorIDs []primitive.ObjectID
	for _, id := range activity.MajorIDs {
		objID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return errors.New("invalid majorId")
		}
		majorIDs = append(majorIDs, objID)
	}

	// กำหนดค่าใหม่ให้ activity
	activity.ID = primitive.NewObjectID()
	activity.AdminID = adminID.Hex()
	activity.ActivityStateID = activityStateID.Hex()
	activity.SkillID = skillID.Hex()
	activity.MajorIDs = nil // ล้างค่า string ก่อนบันทึก
	for _, id := range majorIDs {
		activity.MajorIDs = append(activity.MajorIDs, id.Hex())
	}

	// บันทึกกิจกรรม
	_, err = activityCollection.InsertOne(ctx, activity)
	if err != nil {
		return err
	}

	log.Println("Activity created successfully")
	return nil
}

// GetAllActivities - ดึงข้อมูลกิจกรรมทั้งหมด
func GetAllActivities() ([]models.Activity, error) {
	var activities []models.Activity
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
		activities = append(activities, activity)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return activities, nil
}

// GetActivityByID - ดึงข้อมูลกิจกรรมตาม ID
func GetActivityByID(id primitive.ObjectID) (models.Activity, error) {
	var activity models.Activity
	err := activityCollection.FindOne(ctx, bson.M{"_id": id}).Decode(&activity)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return models.Activity{}, nil
		}
		return models.Activity{}, err
	}
	return activity, nil
}

// UpdateActivity - อัพเดตข้อมูลกิจกรรม
func UpdateActivity(id primitive.ObjectID, activity models.Activity) (models.Activity, error) {
	update := bson.M{
		"$set": activity,
	}

	_, err := activityCollection.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		return models.Activity{}, err
	}

	activity.ID = id
	return activity, nil
}

// DeleteActivity - ลบกิจกรรม
func DeleteActivity(id primitive.ObjectID) error {
	_, err := activityCollection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}
