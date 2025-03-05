package services

import (
	"Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"context"
	"errors"
	"log"
	"math"
	"strings"
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
func CreateActivity(activity *models.ActivityDto) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// ✅ แปลง ActivityState ID
	if activity.ActivityState.ID.IsZero() {
		activity.ActivityState.ID = primitive.NilObjectID
	} else {
		_, err := primitive.ObjectIDFromHex(activity.ActivityState.ID.Hex())
		if err != nil {
			return errors.New("invalid activityStateId")
		}
	}

	// ✅ แปลง Skill ID
	if activity.Skill.ID.IsZero() {
		activity.Skill.ID = primitive.NilObjectID
	} else {
		_, err := primitive.ObjectIDFromHex(activity.Skill.ID.Hex())
		if err != nil {
			return errors.New("invalid skillId")
		}
	}

	// ✅ แปลง Majors เป็น ObjectID List
	var majorIDs []primitive.ObjectID
	for _, major := range activity.Majors {
		majorIDs = append(majorIDs, major.ID)
	}

	// ✅ สร้าง ID สำหรับ Activity
	activity.ID = primitive.NewObjectID()

	// ✅ สร้าง Activity ที่ต้องบันทึกลง MongoDB
	activityToInsert := models.Activity{
		ID:              activity.ID,
		Name:            activity.Name,
		Type:            activity.Type,
		ActivityStateID: activity.ActivityState.ID,
		SkillID:         activity.Skill.ID,
		MajorIDs:        majorIDs,
	}

	// ✅ บันทึก Activity และรับค่า InsertedID กลับมา
	res, err := activityCollection.InsertOne(ctx, activityToInsert)
	if err != nil {
		return err
	}

	// ✅ อัปเดต activity.ID จาก MongoDB
	activity.ID = res.InsertedID.(primitive.ObjectID)

	// ✅ บันทึก ActivityItems
	for i := range activity.ActivityItems {
		activity.ActivityItems[i].ID = primitive.NewObjectID()
		activity.ActivityItems[i].ActivityID = activity.ID

		_, err := activityItemCollection.InsertOne(ctx, activity.ActivityItems[i])
		if err != nil {
			return err
		}
	}

	log.Println("Activity and ActivityItems created successfully")
	return nil
}

// GetAllActivities - ดึง Activity พร้อม ActivityItems + Pagination, Search, Sorting
func GetAllActivities(params models.PaginationParams) ([]models.ActivityDto, int64, int, error) {
	var results []models.ActivityDto
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// คำนวณค่า Skip
	skip := int64((params.Page - 1) * params.Limit)

	// กำหนดค่าเริ่มต้นของการ Sort
	sortField := params.SortBy
	if sortField == "" {
		sortField = "name" // ค่าเริ่มต้นเรียงด้วย Name
	}
	sortOrder := 1 // ค่าเริ่มต้นเป็น ascending (1)
	if strings.ToLower(params.Order) == "desc" {
		sortOrder = -1
	}

	// ค้นหาข้อมูลที่ตรงกับ Search
	filter := bson.M{}
	if params.Search != "" {
		filter["name"] = bson.M{"$regex": params.Search, "$options": "i"} // ค้นหาแบบ Case-Insensitive
	}

	// นับจำนวนเอกสารทั้งหมด
	total, err := activityCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, 0, err
	}

	// ใช้ `$lookup` ดึง ActivityItems ที่เชื่อมโยงกับ Activity
	pipeline := mongo.Pipeline{
		// 🔍 Match เฉพาะ Activity ที่ต้องการ
		{{Key: "$match", Value: filter}},

		// 🔗 Lookup ActivityItems ที่เกี่ยวข้อง
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "activityItems"},
			{Key: "localField", Value: "_id"},
			{Key: "foreignField", Value: "activityId"},
			{Key: "as", Value: "activityItems"},
		}}},

		// 🔗 Lookup ActivityState
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "activityStates"},
			{Key: "localField", Value: "activityStateId"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "activityState"},
		}}},
		// ✅ Unwind เพื่อให้ activityState เป็น object เดียว ไม่ใช่ array
		{{Key: "$unwind", Value: bson.D{
			{Key: "path", Value: "$activityState"},
			{Key: "preserveNullAndEmptyArrays", Value: true},
		}}},
		// 🔗 Lookup Skill
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "skills"},
			{Key: "localField", Value: "skillId"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "skill"},
		}}},
		// ✅ Unwind เพื่อให้ skill เป็น object เดียว
		{{Key: "$unwind", Value: bson.D{
			{Key: "path", Value: "$skill"},
			{Key: "preserveNullAndEmptyArrays", Value: true},
		}}},

		// 🔗 Lookup Majors (เนื่องจากเป็นอาร์เรย์ ต้องใช้ `$lookup` + `$unwind` + `$group`)
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "majors"},
			{Key: "localField", Value: "majorIds"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "majors"},
		}}},

		// 🏷 Sorting และ Pagination
		{{Key: "$sort", Value: bson.D{{Key: sortField, Value: sortOrder}}}},
		{{Key: "$skip", Value: skip}},
		{{Key: "$limit", Value: int64(params.Limit)}},
	}

	// ✅ ต้องใช้ activityCollection แทน activityItemCollection
	cursor, err := activityCollection.Aggregate(ctx, pipeline)
	if err != nil {
		log.Println("Error fetching activities:", err)
		return nil, 0, 0, err
	}
	defer cursor.Close(ctx)

	// Decode ข้อมูลลงใน Struct

	if err = cursor.All(ctx, &results); err != nil {
		log.Println("Error decoding activities:", err)
		return nil, 0, 0, err
	}

	// คำนวณจำนวนหน้าทั้งหมด
	totalPages := int(math.Ceil(float64(total) / float64(params.Limit)))

	return results, total, totalPages, nil
}

func GetActivityByID(activityID string) (*models.Activity, error) {
	var activity models.Activity

	// แปลง activityID จาก string เป็น ObjectID
	objID, err := primitive.ObjectIDFromHex(activityID)
	if err != nil {
		return nil, err
	}

	// ใช้ `$match` และ `$lookup` ดึง Activity + ActivityItems ที่ตรงกับ activityID
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "_id", Value: objID}}}},
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "actividtyItems"},
			{Key: "localField", Value: "_id"},
			{Key: "foreignField", Value: "activityId"},
			{Key: "as", Value: "activityItems"},
		}}},
	}

	// Query และ Decode ข้อมูล
	cursor, err := activityCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if cursor.Next(ctx) {
		if err := cursor.Decode(&activity); err != nil {
			return nil, err
		}
	}

	return &activity, nil
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
