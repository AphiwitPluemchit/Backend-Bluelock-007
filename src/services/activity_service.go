package services

import (
	"Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"context"
	"fmt"
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
func CreateActivity(activity *models.Activity) (*models.Activity, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// ✅ สร้าง ID สำหรับ Activity
	activity.ID = primitive.NewObjectID()

	// ✅ สร้าง Activity ที่ต้องบันทึกลง MongoDB
	activityToInsert := models.Activity{
		ID:            activity.ID,
		Name:          activity.Name,
		Type:          activity.Type,
		ActivityState: activity.ActivityState,
		Skill:         activity.Skill,
		File:          activity.File,
		FoodVotes:     activity.FoodVotes,
	}

	// ✅ บันทึก Activity และรับค่า InsertedID กลับมา
	_, err := activityCollection.InsertOne(ctx, activityToInsert)
	if err != nil {
		return nil, err
	}

	// ✅ บันทึก ActivityItems
	for i := range activity.ActivityItems {
		fmt.Println("ActivityItem:", activity.ActivityItems[i])
		activity.ActivityItems[i].ID = primitive.NewObjectID()
		activity.ActivityItems[i].ActivityID = activity.ID

		_, err := activityItemCollection.InsertOne(ctx, activity.ActivityItems[i])
		if err != nil {
			return nil, err
		}
	}

	log.Println("Activity and ActivityItems created successfully")

	// ✅ ดึงข้อมูล Activity ที่เพิ่งสร้างเสร็จกลับมาให้ Response ✅
	return GetActivityByID(activity.ID.Hex())
}

// GetAllActivities - ดึง Activity พร้อม ActivityItems + Pagination, Search, Sorting
func GetAllActivities(params models.PaginationParams, skills []string, states []string, majors []string, studentYears []int) ([]models.Activity, int64, int, error) {
	var results []models.Activity
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// คำนวณค่า Skip
	skip := int64((params.Page - 1) * params.Limit)

	// กำหนดค่าเริ่มต้นของการ Sort
	sortField := params.SortBy
	if sortField == "" {
		sortField = "name"
	}
	sortOrder := 1
	if strings.ToLower(params.Order) == "desc" {
		sortOrder = -1
	}

	// สร้าง Filter
	filter := bson.M{}

	// 🔍 ค้นหาตามชื่อกิจกรรม (case-insensitive)
	if params.Search != "" {
		filter["name"] = bson.M{"$regex": params.Search, "$options": "i"}
	}

	// 🔍 ค้นหาตาม Skill (ถ้ามี)
	if len(skills) > 0 && skills[0] != "" {
		filter["skill"] = bson.M{"$in": skills}
	}

	// 🔍 ค้นหาตาม ActivityState (ถ้ามี)
	if len(states) > 0 && states[0] != "" {
		filter["activityState"] = bson.M{"$in": states}
	}

	// นับจำนวนเอกสารทั้งหมด
	total, err := activityCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, 0, err
	}

	pipeline := getActivitiesPipeline(filter, sortField, sortOrder, skip, int64(params.Limit), majors, studentYears)

	cursor, err := activityCollection.Aggregate(ctx, pipeline)
	if err != nil {
		log.Println("Error fetching activities:", err)
		return nil, 0, 0, err
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &results); err != nil {
		log.Println("Error decoding activities:", err)
		return nil, 0, 0, err
	}

	// คำนวณจำนวนหน้าทั้งหมด
	totalPages := int(math.Ceil(float64(total) / float64(params.Limit)))

	return results, total, totalPages, nil
}

func GetActivityByID(activityID string) (*models.Activity, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(activityID)
	if err != nil {
		return nil, fmt.Errorf("invalid activity ID format")
	}

	var result models.Activity

	pipeline := GetOneActivityPipeline(objectID)

	cursor, err := activityCollection.Aggregate(ctx, pipeline)
	if err != nil {
		log.Println("Error fetching activity by ID:", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			log.Println("Error decoding activity:", err)
			return nil, err
		}

		return &result, nil
	}

	return nil, fmt.Errorf("activity not found")
}

func GetActivityEnrollSummary(activityID string) (models.EnrollmentSummary, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(activityID)
	if err != nil {
		return models.EnrollmentSummary{}, err
	}

	var result models.EnrollmentSummary

	pipeline := GetActivityStatisticsPipeline(objectID)

	cursor, err := activityItemCollection.Aggregate(ctx, pipeline)
	if err != nil {
		log.Println("Error fetching activity by ID:", err)
		return result, err
	}
	defer cursor.Close(ctx)

	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			log.Println("Error decoding activity:", err)
			return result, err
		}
		return result, nil
	}

	return result, err
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

func UpdateActivity(id primitive.ObjectID, activity models.Activity) (*models.Activity, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// ✅ อัปเดต Activity หลัก
	update := bson.M{
		"$set": bson.M{
			"name":          activity.Name,
			"type":          activity.Type,
			"activityState": activity.ActivityState,
			"skill":         activity.Skill,
			"file":          activity.File,
			"foodVotes":     activity.FoodVotes,
		},
	}

	_, err := activityCollection.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		return nil, err
	}

	// ✅ ดึงรายการ `ActivityItems` ที่มีอยู่
	var existingItems []models.ActivityItem
	cursor, err := activityItemCollection.Find(ctx, bson.M{"activityId": id})
	if err != nil {
		return nil, err
	}
	if err := cursor.All(ctx, &existingItems); err != nil {
		return nil, err
	}

	// ✅ สร้าง Map ของ `existingItems` เพื่อเช็คว่าตัวไหนมีอยู่แล้ว
	existingItemMap := make(map[string]models.ActivityItem)
	for _, item := range existingItems {
		existingItemMap[item.ID.Hex()] = item
	}

	// ✅ สร้าง `Set` สำหรับเก็บ `ID` ของรายการใหม่
	newItemIDs := make(map[string]bool)
	for _, newItem := range activity.ActivityItems {
		if newItem.ID.IsZero() {
			// ✅ ถ้าไม่มี `_id` ให้สร้างใหม่
			newItem.ID = primitive.NewObjectID()
			newItem.ActivityID = id
			_, err := activityItemCollection.InsertOne(ctx, newItem)
			if err != nil {
				return nil, err
			}
		} else {
			// ✅ ถ้ามี `_id` → อัปเดต
			newItemIDs[newItem.ID.Hex()] = true

			_, err := activityItemCollection.UpdateOne(ctx,
				bson.M{"_id": newItem.ID},
				bson.M{"$set": bson.M{
					"name":            newItem.Name,
					"description":     newItem.Description,
					"maxParticipants": newItem.MaxParticipants,
					"rooms":           newItem.Rooms,
					"dates":           newItem.Dates,
					"hour":            newItem.Hour,
					"operator":        newItem.Operator,
					"studentYears":    newItem.StudentYears,
					"majors":          newItem.Majors,
				}},
			)
			if err != nil {
				return nil, err
			}
		}
	}

	// ✅ ลบ `ActivityItems` ที่ไม่มีในรายการใหม่
	for existingID := range existingItemMap {
		if !newItemIDs[existingID] {
			objID, err := primitive.ObjectIDFromHex(existingID) // 🔥 แปลง `string` เป็น `ObjectID`
			if err != nil {
				continue
			}
			_, err = activityItemCollection.DeleteOne(ctx, bson.M{"_id": objID})
			if err != nil {
				return nil, err
			}
		}
	}

	// ✅ ดึงข้อมูล Activity ที่เพิ่งสร้างเสร็จกลับมาให้ Response ✅
	return GetActivityByID(id.Hex())
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

func getActivitiesPipeline(filter bson.M, sortField string, sortOrder int, skip int64, limit int64, majors []string, studentYears []int) mongo.Pipeline {
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

		// 🔥 Unwind ActivityItems เพื่อให้สามารถกรองได้
		{{Key: "$unwind", Value: bson.D{
			{Key: "path", Value: "$activityItems"},
			{Key: "preserveNullAndEmptyArrays", Value: true},
		}}},
	}

	// ✅ กรองเฉพาะ Major ที่ต้องการ **ถ้ามีค่า majorNames**
	if len(majors) > 0 && majors[0] != "" {
		fmt.Println("Filtering by major:", majors) // Debugging log
		pipeline = append(pipeline, bson.D{
			{Key: "$match", Value: bson.D{
				{Key: "activityItems.majors", Value: bson.D{{Key: "$in", Value: majors}}},
			}},
		})
	} else {
		fmt.Println("Skipping majorName filtering")
	}

	// ✅ กรองเฉพาะ StudentYears ที่ต้องการ **ถ้ามีค่า studentYears**
	if len(studentYears) > 0 {
		pipeline = append(pipeline, bson.D{
			{Key: "$match", Value: bson.D{
				{Key: "activityItems.studentYears", Value: bson.D{{Key: "$in", Value: studentYears}}},
			}},
		})
	}

	// ✅ Group ActivityItems กลับเข้าไปใน Activity
	pipeline = append(pipeline, bson.D{
		{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$_id"},
			{Key: "name", Value: bson.D{{Key: "$first", Value: "$name"}}},
			{Key: "type", Value: bson.D{{Key: "$first", Value: "$type"}}},
			{Key: "activityState", Value: bson.D{{Key: "$first", Value: "$activityState"}}},
			{Key: "skill", Value: bson.D{{Key: "$first", Value: "$skill"}}},
			{Key: "file", Value: bson.D{{Key: "$first", Value: "$file"}}},
			{Key: "activityItems", Value: bson.D{{Key: "$push", Value: "$activityItems"}}}, // เก็บ ActivityItems เป็น Array
		}},
	})

	// ✅ ตรวจสอบและเพิ่ม `$sort` เฉพาะกรณีที่ต้องใช้
	if sortField != "" && (sortOrder == 1 || sortOrder == -1) {
		pipeline = append(pipeline, bson.D{{Key: "$sort", Value: bson.D{{Key: sortField, Value: sortOrder}}}})
	}

	// ✅ ตรวจสอบและเพิ่ม `$skip` และ `$limit` เฉพาะกรณีที่ต้องใช้
	if skip > 0 {
		pipeline = append(pipeline, bson.D{{Key: "$skip", Value: skip}})
	}
	if limit > 0 {
		pipeline = append(pipeline, bson.D{{Key: "$limit", Value: limit}})
	}

	return pipeline
}

func GetOneActivityPipeline(activityID primitive.ObjectID) mongo.Pipeline {
	return mongo.Pipeline{
		// 1️⃣ Match เฉพาะ Activity ที่ต้องการ
		{{
			Key: "$match", Value: bson.D{
				{Key: "_id", Value: activityID},
			},
		}},

		// 🔗 Lookup ActivityItems ที่เกี่ยวข้อง
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "activityItems"},
			{Key: "localField", Value: "_id"},
			{Key: "foreignField", Value: "activityId"},
			{Key: "as", Value: "activityItems"},
		}}},

		// //  Unwind ActivityItems เพื่อให้สามารถใช้ Lookup Enrollments ได้
		// {{Key: "$unwind", Value: bson.D{
		// 	{Key: "path", Value: "$activityItems"},
		// 	{Key: "preserveNullAndEmptyArrays", Value: true}, // กรณีไม่มี ActivityItem ให้เก็บค่า null
		// }}},

		// // 🔗 Lookup Enrollments ที่เกี่ยวข้องกับ ActivityItems
		// {{Key: "$lookup", Value: bson.D{
		// 	{Key: "from", Value: "enrollments"},
		// 	{Key: "localField", Value: "activityItems._id"},
		// 	{Key: "foreignField", Value: "activityItemId"},
		// 	{Key: "as", Value: "activityItems.enrollments"},
		// }}},

		// // 🔥 Group ActivityItems กลับเข้าไปใน Activity  ฟังก์ชัน $mergeObjects ที่สามารถรวม Fields ทั้งหมดของ Document เข้าไป
		// {{Key: "$group", Value: bson.D{
		// 	{Key: "_id", Value: "$_id"},
		// 	{Key: "activityData", Value: bson.D{{Key: "$mergeObjects", Value: "$$ROOT"}}},
		// 	{Key: "activityItems", Value: bson.D{{Key: "$push", Value: "$activityItems"}}},
		// }}},

		// // 🔄 แปลงโครงสร้างกลับให้อยู่ในรูปแบบที่ถูกต้อง
		// {{Key: "$replaceRoot", Value: bson.D{
		// 	{Key: "newRoot", Value: bson.D{
		// 		{Key: "$mergeObjects", Value: bson.A{"$activityData", bson.D{{Key: "activityItems", Value: "$activityItems"}}}},
		// 	}},
		// }}},
	}
}

func GetActivityStatisticsPipeline(activityID primitive.ObjectID) mongo.Pipeline {
	return mongo.Pipeline{
		// 1️⃣ Match เฉพาะ ActivityItems ที่ต้องการ
		{{
			Key: "$match", Value: bson.D{
				{Key: "activityId", Value: activityID},
			},
		}},

		// 2️⃣ Lookup Enrollments จาก collection enrollments
		{{
			Key: "$lookup", Value: bson.D{
				{Key: "from", Value: "enrollments"},
				{Key: "localField", Value: "_id"},
				{Key: "foreignField", Value: "activityItemId"},
				{Key: "as", Value: "enrollments"},
			},
		}},

		// 3️⃣ Unwind Enrollments
		{{
			Key: "$unwind", Value: bson.D{
				{Key: "path", Value: "$enrollments"},
				{Key: "preserveNullAndEmptyArrays", Value: true},
			},
		}},

		// 4️⃣ Lookup Students
		{{
			Key: "$lookup", Value: bson.D{
				{Key: "from", Value: "students"},
				{Key: "localField", Value: "enrollments.studentId"},
				{Key: "foreignField", Value: "_id"},
				{Key: "as", Value: "student"},
			},
		}},

		// 5️⃣ Unwind Students
		{{
			Key: "$unwind", Value: bson.D{
				{Key: "path", Value: "$student"},
				{Key: "preserveNullAndEmptyArrays", Value: true},
			},
		}},

		// 6️⃣ Group ตาม ActivityItem และ Major
		{{
			Key: "$group", Value: bson.D{
				{Key: "_id", Value: bson.D{
					{Key: "activityItemId", Value: "$_id"},
					{Key: "majorName", Value: "$student.major"},
				}},
				{Key: "activityItemName", Value: bson.D{{Key: "$first", Value: "$name"}}},
				{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
				{Key: "maxParticipants", Value: bson.D{{Key: "$first", Value: "$maxParticipants"}}},
			},
		}},

		// 9️⃣ Group ActivityItemSums
		{{
			Key: "$group", Value: bson.D{
				{Key: "_id", Value: "$_id.activityItemId"},
				{Key: "activityItemName", Value: bson.D{{Key: "$first", Value: "$activityItemName"}}},
				{Key: "maxParticipants", Value: bson.D{{Key: "$first", Value: "$maxParticipants"}}},
				{Key: "totalRegistered", Value: bson.D{{Key: "$sum", Value: "$count"}}},
				{Key: "registeredByMajor", Value: bson.D{{
					Key: "$push", Value: bson.D{
						{Key: "majorName", Value: "$_id.majorName"},
						{Key: "count", Value: "$count"},
					},
				}}},
			},
		}},

		// 🔟 Group Final Result
		{{
			Key: "$group", Value: bson.D{
				{Key: "_id", Value: nil},
				{Key: "maxParticipants", Value: bson.D{{Key: "$sum", Value: "$maxParticipants"}}},
				{Key: "totalRegistered", Value: bson.D{{Key: "$sum", Value: "$totalRegistered"}}},
				{Key: "activityItemSums", Value: bson.D{{Key: "$push", Value: bson.D{
					{Key: "activityItemName", Value: "$activityItemName"},
					{Key: "registeredByMajor", Value: "$registeredByMajor"},
				}}}},
			},
		}},

		// 11️⃣ Add field remainingSlots
		{{
			Key: "$addFields", Value: bson.D{
				{Key: "remainingSlots", Value: bson.D{{Key: "$subtract", Value: bson.A{"$maxParticipants", "$totalRegistered"}}}},
			},
		}},

		// 12️⃣ Project Final Output
		{{
			Key: "$project", Value: bson.D{
				{Key: "_id", Value: 0},
				{Key: "maxParticipants", Value: 1},
				{Key: "totalRegistered", Value: 1},
				{Key: "remainingSlots", Value: 1},
				{Key: "activityItemSums", Value: 1},
			},
		}},
	}
}

func GetEnrollmentByActivityID(activityID string, pagination models.PaginationParams, majors []string, status []int) ([]models.Enrollment, int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(activityID)
	if err != nil {
		return nil, 0, err
	}

	pipeline := GetEnrollmentByActivityIDPipeline(objectID, pagination, majors, status)
	cursor, err := activityItemCollection.Aggregate(ctx, pipeline)
	if err != nil {
		log.Println("Error fetching enrollments:", err)
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var results []models.Enrollment
	if err = cursor.All(ctx, &results); err != nil {
		log.Println("Error decoding enrollments:", err)
		return nil, 0, err
	}

	total, err := activityItemCollection.CountDocuments(ctx, bson.M{"activityId": objectID})
	if err != nil {
		return nil, 0, err
	}

	return results, total, nil
}

func GetActivityItemIDsByActivityID(ctx context.Context, activityID primitive.ObjectID) ([]primitive.ObjectID, error) {
	var activityItems []models.ActivityItem
	filter := bson.M{"activityId": activityID}
	cursor, err := activityItemCollection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &activityItems); err != nil {
		return nil, err
	}

	var activityItemIDs []primitive.ObjectID
	for _, item := range activityItems {
		activityItemIDs = append(activityItemIDs, item.ID)
	}

	fmt.Println(activityItemIDs)
	return activityItemIDs, nil
}

func GetEnrollmentByActivityIDPipeline(activityID primitive.ObjectID, pagination models.PaginationParams, majors []string, status []int) mongo.Pipeline {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "activityId", Value: activityID}}}},
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "enrollments"},
			{Key: "localField", Value: "_id"},
			{Key: "foreignField", Value: "activityItemId"},
			{Key: "as", Value: "enrollments"},
		}}},
		{{Key: "$unwind", Value: bson.D{
			{Key: "path", Value: "$enrollments"},
			{Key: "preserveNullAndEmptyArrays", Value: true},
		}}},
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "students"},
			{Key: "localField", Value: "enrollments.studentId"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "enrollments.student"},
		}}},
		{{Key: "$unwind", Value: bson.D{
			{Key: "path", Value: "$enrollments.student"},
			{Key: "preserveNullAndEmptyArrays", Value: true},
		}}},

		// เพิ่ม `$addFields` เพื่อแยก `major` ออกมาก่อนทำ `$match`
		{{Key: "$addFields", Value: bson.D{
			{Key: "studentMajor", Value: "$enrollments.student.major"},
		}}},
	}

	// Apply filter for student majors if provided
	if len(majors) > 0 {
		pipeline = append(pipeline, bson.D{{Key: "$match", Value: bson.D{{Key: "studentMajor", Value: bson.M{"$in": majors}}}}})
	}

	// Apply filter for student status if provided
	if len(status) > 0 {
		pipeline = append(pipeline, bson.D{{Key: "$match", Value: bson.D{{Key: "enrollments.student.status", Value: bson.M{"$in": status}}}}})
	}

	// Apply search filter if provided
	if pagination.Search != "" {
		searchRegex := bson.M{"$regex": pagination.Search, "$options": "i"} // Case-insensitive search
		pipeline = append(pipeline, bson.D{{Key: "$match", Value: bson.D{
			{Key: "$or", Value: bson.A{
				bson.D{{Key: "enrollments.student.name", Value: searchRegex}},
				bson.D{{Key: "enrollments.student.code", Value: searchRegex}},
			}},
		}}})
	}

	pipeline = append(pipeline,
		bson.D{{Key: "$project", Value: bson.D{
			{Key: "_id", Value: "$enrollments._id"},
			{Key: "registrationDate", Value: "$enrollments.registrationDate"},
			{Key: "activityItemId", Value: "$enrollments.activityItemId"},
			{Key: "studentId", Value: "$enrollments.studentId"},
			{Key: "student", Value: "$enrollments.student"},
		}}},
		bson.D{{Key: "$skip", Value: (pagination.Page - 1) * pagination.Limit}},
		bson.D{{Key: "$limit", Value: pagination.Limit}},
	)

	return pipeline
}
