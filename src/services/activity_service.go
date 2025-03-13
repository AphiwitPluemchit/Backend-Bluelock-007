package services

import (
	"Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"context"
	"fmt"
	"log"
	"math"
	"strconv"
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
func CreateActivity(activity *models.ActivityDto) (*models.ActivityDto, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// ✅ แปลง Majors เป็น ObjectID List
	var majorIDs []primitive.ObjectID
	for _, major := range activity.Majors {
		majorIDs = append(majorIDs, major.ID)
	}

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
		StudentYears:  activity.StudentYears,
		MajorIDs:      majorIDs,
	}

	// ✅ บันทึก Activity และรับค่า InsertedID กลับมา
	_, err := activityCollection.InsertOne(ctx, activityToInsert)
	if err != nil {
		return nil, err
	}

	// ✅ บันทึก ActivityItems
	for i := range activity.ActivityItems {
		activity.ActivityItems[i].ID = primitive.NewObjectID()
		activity.ActivityItems[i].ActivityID = activity.ID

		_, err := activityItemCollection.InsertOne(ctx, activity.ActivityItems[i])
		if err != nil {
			return nil, err
		}
	}

	// ✅ บันทึก FoodVotes
	for i := range activity.FoodVotes {
		activity.FoodVotes[i].ID = primitive.NewObjectID()
		activity.FoodVotes[i].ActivityID = activity.ID
		activity.FoodVotes[i].FoodID = activity.FoodVotes[i].Food.ID

		_, err := foodVoteCollection.InsertOne(ctx, activity.FoodVotes[i])
		if err != nil {
			return nil, err
		}
	}

	log.Println("Activity and ActivityItems created successfully")

	// ✅ ดึงข้อมูล Activity ที่เพิ่งสร้างเสร็จกลับมาให้ Response ✅
	return GetActivityByID(activity.ID.Hex())
}

// GetAllActivities - ดึง Activity พร้อม ActivityItems + Pagination, Search, Sorting
func GetAllActivities(params models.PaginationParams, skills []string, states []string, majorNames []string, studentYears []string) ([]models.ActivityDto, int64, int, error) {
	var results []models.ActivityDto
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

	// 🔍 ค้นหาตาม StudentYear (ถ้ามี)
	if len(studentYears) > 0 && studentYears[0] != "" {
		var years []int
		for _, year := range studentYears {
			y, err := strconv.Atoi(year)
			if err == nil {
				years = append(years, y)
			}
		}
		if len(years) > 0 {
			filter["studentYears"] = bson.M{"$in": years}
		}
	}

	// นับจำนวนเอกสารทั้งหมด
	total, err := activityCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, 0, err
	}

	pipeline := getActivityPipeline(filter, sortField, sortOrder, skip, int64(params.Limit), majorNames)

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

func GetActivityByID(activityID string) (*models.ActivityDto, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(activityID)
	if err != nil {
		return nil, fmt.Errorf("invalid activity ID format")
	}

	var result models.ActivityDto

	pipeline := GetOneActivityPipeline(bson.M{"_id": objectID})

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

type EnrollmentSummary struct {
	MaxParticipants   int               `json:"maxParticipants"`
	TotalRegistered   int               `json:"totalRegistered"`
	RemainingSlots    int               `json:"remainingSlots"`
	RegisteredByMajor []MajorEnrollment `json:"registeredByMajor"`
}

// โครงสร้างสำหรับแยกจำนวนลงทะเบียนตามสาขา
type MajorEnrollment struct {
	MajorName string `json:"majorName"`
	Count     int    `json:"count"`
}

func GetActivityEnrollSummary(activityID string) (EnrollmentSummary, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(activityID)
	if err != nil {
		return EnrollmentSummary{}, err
	}

	var result EnrollmentSummary

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

func UpdateActivity(id primitive.ObjectID, activity models.ActivityDto) (models.ActivityDto, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// ✅ แปลง Majors เป็น ObjectID List
	var majorIDs []primitive.ObjectID
	for _, major := range activity.Majors {
		majorIDs = append(majorIDs, major.ID)
	}

	// ✅ อัปเดต Activity หลัก
	update := bson.M{
		"$set": bson.M{
			"name":          activity.Name,
			"type":          activity.Type,
			"activityState": activity.ActivityState,
			"skill":         activity.Skill,
			"file":          activity.File,
			"studentYears":  activity.StudentYears,
			"majorIds":      majorIDs,
		},
	}

	_, err := activityCollection.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		return models.ActivityDto{}, err
	}

	// ✅ ดึงรายการ `ActivityItems` ที่มีอยู่
	var existingItems []models.ActivityItem
	cursor, err := activityItemCollection.Find(ctx, bson.M{"activityId": id})
	if err != nil {
		return models.ActivityDto{}, err
	}
	if err := cursor.All(ctx, &existingItems); err != nil {
		return models.ActivityDto{}, err
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
				return models.ActivityDto{}, err
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
					"room":            newItem.Room,
					"dates":           newItem.Dates,
					"hour":            newItem.Hour,
					"operator":        newItem.Operator,
				}},
			)
			if err != nil {
				return models.ActivityDto{}, err
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
				return models.ActivityDto{}, err
			}
		}
	}

	// ดึงรายการ FoodVote ที่มีอยู่
	var existingFoodVotes []models.FoodVote
	cursor, err = foodVoteCollection.Find(ctx, bson.M{"activityId": id})
	if err != nil {
		return activity, err
	}
	if err := cursor.All(ctx, &existingFoodVotes); err != nil {
		return activity, err
	}

	// สร้าง Map ของ `existingFoodVotes` เพื่อเช็คว่าตัวไหนมีอยู่แล้ว
	existingFoodVoteMap := make(map[string]models.FoodVote)
	for _, foodVote := range existingFoodVotes {
		existingFoodVoteMap[foodVote.ID.Hex()] = foodVote
	}

	// สร้าง `Set` สำหรับเก็บ `ID` ของรายการใหม่
	newFoodVoteIDs := make(map[string]bool)
	for _, newFoodVote := range activity.FoodVotes {
		if newFoodVote.ID.IsZero() {
			// ถ้าไม่มี `_id` ให้สร้างใหม่
			newFoodVote.ID = primitive.NewObjectID()
			newFoodVote.ActivityID = id
			_, err := foodVoteCollection.InsertOne(ctx, newFoodVote)
			if err != nil {
				return activity, err
			}
		} else {
			// ถ้ามี `_id` → อัปเดต
			newFoodVoteIDs[newFoodVote.ID.Hex()] = true

			_, err := foodVoteCollection.UpdateOne(ctx,
				bson.M{"_id": newFoodVote.ID},
				bson.M{"$set": bson.M{
					"foodId": newFoodVote.FoodID,
					"food":   newFoodVote.Food,
					"vote":   newFoodVote.Vote,
				}},
			)
			if err != nil {
				return activity, err
			}
		}
	}

	// ลบ `FoodVotes` ที่ไม่มีในรายการใหม่
	for existingID := range existingFoodVoteMap {
		if !newFoodVoteIDs[existingID] {
			objID, err := primitive.ObjectIDFromHex(existingID) // 🔥 แปลง `string` เป็น `ObjectID`
			if err != nil {
				continue
			}
			_, err = foodVoteCollection.DeleteOne(ctx, bson.M{"_id": objID})
			if err != nil {
				return activity, err
			}
		}

	}

	return activity, nil
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

func getActivityPipeline(filter bson.M, sortField string, sortOrder int, skip int64, limit int64, majorNames []string) mongo.Pipeline {
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

		// 🔗 Lookup Majors
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "majors"},
			{Key: "localField", Value: "majorIds"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "majors"},
		}}},
	}

	// ✅ กรองเฉพาะ Major ที่ต้องการ **ถ้ามีค่า majorNames**
	if majorNames[0] != "" {
		fmt.Println("Filtering by majorNames:", majorNames) // Debugging log
		pipeline = append(pipeline, bson.D{
			{Key: "$match", Value: bson.D{
				{Key: "majors.majorName", Value: bson.D{{Key: "$in", Value: majorNames}}},
			}},
		})
	} else {
		fmt.Println("Skipping majorName filtering")
	}

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

func GetOneActivityPipeline(filter bson.M) mongo.Pipeline {
	return mongo.Pipeline{
		// 🔍 Match เฉพาะ Activity ที่ต้องการ
		{{Key: "$match", Value: filter}},

		// 🔗 Lookup ActivityItems ที่เกี่ยวข้อง
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "activityItems"},
			{Key: "localField", Value: "_id"},
			{Key: "foreignField", Value: "activityId"},
			{Key: "as", Value: "activityItems"},
		}}},

		// 🔥 Unwind ActivityItems เพื่อให้สามารถใช้ Lookup Enrollments ได้
		{{Key: "$unwind", Value: bson.D{
			{Key: "path", Value: "$activityItems"},
			{Key: "preserveNullAndEmptyArrays", Value: true}, // กรณีไม่มี ActivityItem ให้เก็บค่า null
		}}},

		// 🔗 Lookup Enrollments ที่เกี่ยวข้องกับ ActivityItems
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "enrollments"},
			{Key: "localField", Value: "activityItems._id"},
			{Key: "foreignField", Value: "activityItemId"},
			{Key: "as", Value: "activityItems.enrollments"},
		}}},

		// 🔗 Lookup Majors
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "majors"},
			{Key: "localField", Value: "majorIds"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "majors"},
		}}},

		// Lookup FoodVote
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "foodVotes"},
			{Key: "localField", Value: "_id"},
			{Key: "foreignField", Value: "activityId"},
			{Key: "as", Value: "foodVotes"},
		}}},

		// 🔥 Group ActivityItems กลับเข้าไปใน Activity  ฟังก์ชัน $mergeObjects ที่สามารถรวม Fields ทั้งหมดของ Document เข้าไป
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$_id"},
			{Key: "activityData", Value: bson.D{{Key: "$mergeObjects", Value: "$$ROOT"}}},
			{Key: "activityItems", Value: bson.D{{Key: "$push", Value: "$activityItems"}}},
		}}},

		// 🔄 แปลงโครงสร้างกลับให้อยู่ในรูปแบบที่ถูกต้อง
		{{Key: "$replaceRoot", Value: bson.D{
			{Key: "newRoot", Value: bson.D{
				{Key: "$mergeObjects", Value: bson.A{"$activityData", bson.D{{Key: "activityItems", Value: "$activityItems"}}}},
			}},
		}}},
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

		// 2️⃣ Group แรก: รวมค่า maxParticipants ของ activityItem ทั้งหมด
		//    และรวม enrollment ทั้งหมดไว้ใน fields ชื่อ "allEnrollments"
		{{
			Key: "$group", Value: bson.D{
				{Key: "_id", Value: "$activityId"},
				// รวมค่า maxParticipants จาก activityItem หลายตัว
				{Key: "maxParticipants", Value: bson.D{{Key: "$sum", Value: "$maxParticipants"}}},
				// สะสม ID ของ activityItem ไว้ (ถ้าต้อง Lookup ต่อ)
				{Key: "itemIds", Value: bson.D{{Key: "$push", Value: "$_id"}}},
			},
		}},

		// 3️⃣ Lookup Enrollments จาก collection enrollments
		//    โดยจับคู่ field "itemIds" กับ "activityItemId"
		{{
			Key: "$lookup", Value: bson.D{
				{Key: "from", Value: "enrollments"},
				{Key: "localField", Value: "itemIds"},
				{Key: "foreignField", Value: "activityItemId"},
				{Key: "as", Value: "enrollments"},
			},
		}},

		// 4️⃣ Unwind Enrollments (ถ้าไม่มี Enrollments จะไม่ดรอป document ทิ้ง)
		{{
			Key: "$unwind", Value: bson.D{
				{Key: "path", Value: "$enrollments"},
				{Key: "preserveNullAndEmptyArrays", Value: true},
			},
		}},

		// 5️⃣ Lookup Students
		{{
			Key: "$lookup", Value: bson.D{
				{Key: "from", Value: "students"},
				{Key: "localField", Value: "enrollments.studentId"},
				{Key: "foreignField", Value: "_id"},
				{Key: "as", Value: "students"},
			},
		}},

		// 6️⃣ Unwind Students
		{{
			Key: "$unwind", Value: bson.D{
				{Key: "path", Value: "$students"},
				{Key: "preserveNullAndEmptyArrays", Value: true},
			},
		}},

		// 7️⃣ Group ตาม MajorID และนับจำนวน student ต่อ major
		//    พร้อมกับ "ยก" maxParticipants จากขั้นตอนก่อนหน้ามาด้วย
		{{
			Key: "$group", Value: bson.D{
				{Key: "_id", Value: "$students.majorId"},
				{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
				// สำคัญ: เก็บค่า maxParticipants ที่ได้มาจาก group แรก
				{Key: "maxParticipants", Value: bson.D{{Key: "$first", Value: "$maxParticipants"}}},
			},
		}},

		// 8) Lookup "majors" เพื่อดึง majorName จาก _id ของ major
		{{
			Key: "$lookup", Value: bson.D{
				{Key: "from", Value: "majors"},
				{Key: "localField", Value: "_id"},   // ตอนนี้ _id = majorId
				{Key: "foreignField", Value: "_id"}, // ใน majors ก็เก็บ _id
				{Key: "as", Value: "majorData"},
			},
		}},

		// 9) Unwind majorData (ถ้าไม่พบ จะได้ null)
		{{
			Key: "$unwind", Value: bson.D{
				{Key: "path", Value: "$majorData"},
				{Key: "preserveNullAndEmptyArrays", Value: true},
			},
		}},

		// 10) Group สุดท้าย → รวมเป็น Document เดียว
		//     เอา maxParticipants, totalRegistered, และ registeredByMajor (majorName + count)
		{{
			Key: "$group", Value: bson.D{
				{Key: "_id", Value: nil},
				// ใช้ $first เพื่อรักษาค่า maxParticipants
				{Key: "maxParticipants", Value: bson.D{{Key: "$first", Value: "$maxParticipants"}}},
				// นับ totalRegistered จาก sum ของ count
				{Key: "totalRegistered", Value: bson.D{{Key: "$sum", Value: "$count"}}},
				// เก็บ registeredByMajor เป็น array
				{Key: "registeredByMajor", Value: bson.D{{
					Key: "$push", Value: bson.D{
						// จะเก็บ majorId ไว้ด้วยก็ได้ หากต้องการ
						// {Key: "majorId", Value: "$_id"},

						// หรือจะแทน _id ด้วย majorName เลย
						{Key: "majorName", Value: "$majorData.majorName"},
						{Key: "count", Value: "$count"},
					},
				}}},
			},
		}},

		// 11) Add field remainingSlots
		{{
			Key: "$addFields", Value: bson.D{
				{Key: "remainingSlots", Value: bson.D{
					{Key: "$subtract", Value: bson.A{"$maxParticipants", "$totalRegistered"}},
				}},
			},
		}},
		// 12) Project ค่า Final
		{{
			Key: "$project", Value: bson.D{
				{Key: "_id", Value: 0},
				{Key: "maxParticipants", Value: 1},
				{Key: "totalRegistered", Value: 1},
				{Key: "remainingSlots", Value: 1},
				{Key: "registeredByMajor", Value: 1},
			},
		}},
	}
}

func getRegisterPipeline(activityId primitive.ObjectID) mongo.Pipeline {
	return mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "activityId", Value: activityId}}}},

		//  Lookup Enrollments จาก collection enrollments
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "enrollments"},
			{Key: "localField", Value: "_id"},
			{Key: "foreignField", Value: "activityItemId"},
			{Key: "as", Value: "enrollments"},
		}}},

		//  Unwind Enrollments (เก็บค่า null)
		{{Key: "$unwind", Value: bson.D{{Key: "path", Value: "$enrollments"}, {Key: "preserveNullAndEmptyArrays", Value: true}}}},

		//  Lookup Students
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "students"},
			{Key: "localField", Value: "enrollments.studentId"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "students"},
		}}},

		//  Unwind Students (เก็บค่า null)
		{{Key: "$unwind", Value: bson.D{{Key: "path", Value: "$students"}, {Key: "preserveNullAndEmptyArrays", Value: true}}}},

		//  Group by ActivityItemID เพื่อเก็บ maxParticipants และ totalRegistered
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$_id"},
			{Key: "students", Value: bson.D{{Key: "$push", Value: "$students"}}},
		}}},

		//  Project Final Output
		{{Key: "$project", Value: bson.D{
			{Key: "_id", Value: 0},
			{Key: "students", Value: "$students"},
		}}},
	}
}
