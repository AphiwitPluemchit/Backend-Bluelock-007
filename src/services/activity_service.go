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

var ctx = context.Background()

var activityCollection *mongo.Collection
var activityItemCollection *mongo.Collection

func init() {
	if err := database.ConnectMongoDB(); err != nil {
		log.Fatal("MongoDB connection error:", err)
	}

	activityCollection = database.GetCollection("BluelockDB", "activitys")
	activityItemCollection = database.GetCollection("BluelockDB", "activityItems")

	if activityCollection == nil || activityItemCollection == nil {
		log.Fatal("Failed to get the required collections")
	}
}

// CreateActivity - สร้าง Activity และ ActivityItems
func CreateActivity(activity models.Activity, activityItems []models.ActivityItem) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// สร้าง ID สำหรับ Activity ก่อน
	activity.ID = primitive.NewObjectID()

	// แปลงค่า ObjectID ที่เกี่ยวข้อง
	adminID, err := primitive.ObjectIDFromHex(activity.AdminID.Hex())
	if err != nil {
		return errors.New("invalid adminId")
	}
	activityStateID, err := primitive.ObjectIDFromHex(activity.ActivityStateID.Hex())
	if err != nil {
		return errors.New("invalid activityStateId")
	}
	skillID, err := primitive.ObjectIDFromHex(activity.SkillID.Hex())
	if err != nil {
		return errors.New("invalid skillId")
	}

	var majorIDs []primitive.ObjectID
	for _, id := range activity.MajorIDs {
		objID, err := primitive.ObjectIDFromHex(id.Hex())
		if err != nil {
			return errors.New("invalid majorId")
		}
		majorIDs = append(majorIDs, objID)
	}

	// อัปเดตค่าก่อนบันทึก
	activity.AdminID = adminID
	activity.ActivityStateID = activityStateID
	activity.SkillID = skillID
	activity.MajorIDs = majorIDs

	// บันทึก Activity ก่อน
	_, err = activityCollection.InsertOne(ctx, activity)
	if err != nil {
		return err
	}

	// บันทึก ActivityItems และตั้งค่า ActivityID
	for i := range activityItems {
		activityItems[i].ID = primitive.NewObjectID()
		activityItems[i].ActivityID = activity.ID

		_, err := activityItemCollection.InsertOne(ctx, activityItems[i])
		if err != nil {
			return err
		}
	}

	log.Println("Activity and ActivityItems created successfully")
	return nil
}

// GetAllActivities - ดึงข้อมูลกิจกรรมทั้งหมดพร้อม ActivityItems
func GetAllActivities() ([]models.Activity, error) {
	var activities []models.Activity

	// ค้นหากิจกรรมทั้งหมด
	cursor, err := activityCollection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	// อ่านผลลัพธ์จาก cursor
	for cursor.Next(ctx) {
		var activity models.Activity
		if err := cursor.Decode(&activity); err != nil {
			return nil, err
		}

		// ดึงข้อมูล ActivityItems ที่เชื่อมโยงกับ Activity
		activityItems, err := GetActivityItemsByActivityID(activity.ID)
		if err != nil {
			return nil, err
		}
		// เพิ่ม ActivityItems ลงในข้อมูลกิจกรรม
		activity.ActivityItems = &activityItems
		activities = append(activities, activity)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return activities, nil
}

// GetActivityByID - ดึงข้อมูลกิจกรรมตาม ID พร้อม ActivityItems
func GetActivityByID(id primitive.ObjectID) (models.Activity, []models.ActivityItem, error) {
	var activity models.Activity
	// ค้นหากิจกรรมตาม ID
	err := activityCollection.FindOne(ctx, bson.M{"_id": id}).Decode(&activity)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return models.Activity{}, nil, nil
		}
		return models.Activity{}, nil, err
	}

	// ดึง ActivityItems ที่เชื่อมโยงกับ ActivityID
	activityItems, err := GetActivityItemsByActivityID(activity.ID)
	if err != nil {
		return models.Activity{}, nil, err
	}

	return activity, activityItems, nil
}

// GetActivityItemsByActivityID - ดึง ActivityItems ตาม ActivityID
func GetActivityItemsByActivityID(activityID primitive.ObjectID) ([]models.ActivityItem, error) {
	var activityItems []models.ActivityItem
	cursor, err := activityItemCollection.Find(ctx, bson.M{"activityId": activityID})
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

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return activityItems, nil
}

// UpdateActivity - อัปเดตกิจกรรมและ ActivityItems
func UpdateActivity(id primitive.ObjectID, activity models.Activity, activityItems []models.ActivityItem) (models.Activity, []models.ActivityItem, error) {
	// อัปเดต Activity
	update := bson.M{
		"$set": activity,
	}
	_, err := activityCollection.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		return models.Activity{}, nil, err
	}

	// อัปเดต ActivityItems (ถ้ามีการเปลี่ยนแปลง)
	var updatedActivityItems []models.ActivityItem
	for _, item := range activityItems {
		item.ActivityID = activity.ID // ตั้งค่า ActivityID ใหม่
		item.ID = primitive.NewObjectID()

		// บันทึก ActivityItem ลง MongoDB
		_, err := activityItemCollection.InsertOne(ctx, item)
		if err != nil {
			return models.Activity{}, nil, err
		}
		updatedActivityItems = append(updatedActivityItems, item)
	}

	// คืนค่าข้อมูลที่อัปเดต
	return activity, updatedActivityItems, nil
}

// DeleteActivity - ลบกิจกรรมและ ActivityItems ที่เกี่ยวข้อง
func DeleteActivity(id primitive.ObjectID) error {
	// ลบ ActivityItems ที่เชื่อมโยงกับ Activity
	_, err := activityItemCollection.DeleteMany(ctx, bson.M{"activityId": id})
	if err != nil {
		return err
	}

	// ลบ Activity
	_, err = activityCollection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}
