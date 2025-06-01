package services

import (
	"Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/jobs"
	"Backend-Bluelock-007/src/models"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hibiken/asynq"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var ctx = context.Background()

var activityCollection *mongo.Collection
var activityItemCollection *mongo.Collection
var AsynqClient *asynq.Client
var redisURI string

func InitAsynq() {
	redisURI = os.Getenv("REDIS_URI")
	if redisURI == "" {
		// redisURI = "localhost:6379"
	} else {
		AsynqClient = asynq.NewClient(asynq.RedisClientOpt{Addr: redisURI})
		fmt.Println("Redis URI:", redisURI)
	}

	// AsynqClient = asynq.NewClient(asynq.RedisClientOpt{Addr: redisURI})
}

func init() {
	if err := database.ConnectMongoDB(); err != nil {
		log.Fatal("MongoDB connection error:", err)
	}

	activityCollection = database.GetCollection("BluelockDB", "activitys")
	activityItemCollection = database.GetCollection("BluelockDB", "activityItems")

	if activityCollection == nil || activityItemCollection == nil {
		log.Fatal("Failed to get the required collections")
	}

	if redisURI != "" {
		InitAsynq()
	}

}

// CreateActivity - สร้าง Activity และ ActivityItems
func CreateActivity(activity *models.ActivityDto) (*models.ActivityDto, error) {
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
		EndDateEnroll: activity.EndDateEnroll,
	}

	// ✅ บันทึก Activity และรับค่า InsertedID กลับมา
	_, err := activityCollection.InsertOne(ctx, activityToInsert)
	if err != nil {
		return nil, err
	}

	// ✅ บันทึก ActivityItems
	var itemsToInsert []any

	// ✅ วนหาเวลาสิ้นสุดที่มากที่สุด
	var latestTime time.Time

	for _, item := range activity.ActivityItems {
		itemToInsert := models.ActivityItem{
			ID:              primitive.NewObjectID(),
			ActivityID:      activity.ID,
			Name:            item.Name,
			Description:     item.Description,
			StudentYears:    item.StudentYears,
			MaxParticipants: item.MaxParticipants,
			Majors:          item.Majors,
			Rooms:           item.Rooms,
			Operator:        item.Operator,
			Dates:           item.Dates,
			Hour:            item.Hour,
		}
		itemsToInsert = append(itemsToInsert, itemToInsert)

		if redisURI != "" {

			// ✅ คำนวณ latestTime
			latestTime = MaxEndTimeFromItem(item, latestTime)
		}

	}

	// ✅ Insert ทั้งหมดในครั้งเดียว เร็วขึ้นมากในการ insert หลายรายการ ลดจำนวนการ round-trip ไปยัง MongoDB
	_, err = activityItemCollection.InsertMany(ctx, itemsToInsert)
	if err != nil {
		return nil, err
	}

	if redisURI != "" {
		err = ScheduleChangeActivityStateJob(latestTime, activity.EndDateEnroll, activity.ID.Hex())
		if err != nil {
			return nil, err
		}
	}

	log.Println("Activity and ActivityItems created successfully")

	// ✅ ดึงข้อมูล Activity ที่เพิ่งสร้างเสร็จกลับมาให้ Response ✅
	return GetActivityByID(activity.ID.Hex())
}

func UploadActivityImage(activityID string, fileName string) error {
	// string to primitive.ObjectID
	objectID, err := primitive.ObjectIDFromHex(activityID)
	if err != nil {
		return err
	}

	// update image
	filter := bson.M{"_id": objectID}
	update := bson.M{"$set": bson.M{"file": fileName}}
	_, err = activityCollection.UpdateOne(context.Background(), filter, update)
	return err
}

// GetAllActivities - ดึง Activity พร้อม ActivityItems + Pagination, Search, Sorting
func GetAllActivities(params models.PaginationParams, skills []string, states []string, majors []string, studentYears []int) ([]models.ActivityDto, int64, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 🔑 สร้าง Redis Key จาก params
	key := fmt.Sprintf(
		"activities:page=%d&limit=%d&search=%s&sortBy=%s&order=%s&skills=%v&states=%v&majors=%v&years=%v",
		params.Page, params.Limit, params.Search, params.SortBy, params.Order,
		skills, states, majors, studentYears,
	)

	if redisURI != "" { // ถ้ามีการเชื่อมต่อกับ Redis ให้ใช้ Redis ในการดึงข้อมูล
		// ✅ ลองอ่านจาก Redis ก่อน
		cached, err := database.RedisClient.Get(database.RedisCtx, key).Result()
		if err == nil {
			var cachedResult struct {
				Data       []models.ActivityDto `json:"data"`
				Total      int64                `json:"total"`
				TotalPages int                  `json:"totalPages"`
			}
			if json.Unmarshal([]byte(cached), &cachedResult) == nil {
				return cachedResult.Data, cachedResult.Total, cachedResult.TotalPages, nil
			}
		}
	}

	var results []models.ActivityDto

	// Calculate skip
	skip := int64((params.Page - 1) * params.Limit)

	// Set default sort field
	sortField := params.SortBy
	fmt.Println("sortField:", sortField)
	if sortField == "" {

		fmt.Println("No sort field provided, defaulting to 'dates'" + sortField)
		sortField = "dates"
	}

	sortOrder := 1
	if strings.ToLower(params.Order) == "desc" {
		sortOrder = -1
	}

	// Build filter
	filter := bson.M{}

	if params.Search != "" {
		searchRegex := bson.M{"$regex": params.Search, "$options": "i"}
		filter["$or"] = bson.A{
			bson.M{"name": searchRegex},
			bson.M{"skill": searchRegex},
		}
	}
	if len(skills) > 0 && skills[0] != "" {
		filter["skill"] = bson.M{"$in": skills}
	}
	if len(states) > 0 && states[0] != "" {
		filter["activityState"] = bson.M{"$in": states}
	}

	pipeline := getLightweightActivitiesPipeline(filter, sortField, sortOrder, skip, int64(params.Limit), majors, studentYears)
	cursor, err := activityCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, 0, 0, err
	}
	defer cursor.Close(ctx)

	if err := cursor.All(ctx, &results); err != nil {
		return nil, 0, 0, err
	}

	// 👇 ใช้ pipeline ใหม่สำหรับ count โดยไม่ใส่ skip/limit
	countPipeline := getLightweightActivitiesPipeline(filter, "", 0, 0, 0, majors, studentYears)
	countPipeline = append(countPipeline, bson.D{{Key: "$count", Value: "total"}})

	cursor, err = activityCollection.Aggregate(ctx, countPipeline)
	if err != nil {
		return nil, 0, 0, err
	}

	var countResult []bson.M
	if err = cursor.All(ctx, &countResult); err != nil {
		return nil, 0, 0, err
	}

	var total int64 = 0
	if len(countResult) > 0 {
		switch v := countResult[0]["total"].(type) {
		case int32:
			total = int64(v)
		case int64:
			total = v
		default:
			log.Printf("unexpected type for count: %T", v)
		}
	}
	// Populate enrollment count in Go
	for i, activity := range results {
		for j, item := range activity.ActivityItems {
			count, err := enrollmentCollection.CountDocuments(ctx, bson.M{"activityItemId": item.ID})
			if err == nil {
				results[i].ActivityItems[j].EnrollmentCount = int(count)
			}
		}
	}

	totalPages := int(math.Ceil(float64(total) / float64(params.Limit)))

	if redisURI != "" {

		// 🔚 เก็บผลลัพธ์ใน Redis
		cacheValue, _ := json.Marshal(struct {
			Data       []models.ActivityDto `json:"data"`
			Total      int64                `json:"total"`
			TotalPages int                  `json:"totalPages"`
		}{
			Data:       results,
			Total:      total,
			TotalPages: totalPages,
		})

		log.Println("🗃️ Cache miss: querying MongoDB and storing to Redis:", key)

		_ = database.RedisClient.Set(database.RedisCtx, key, cacheValue, 2*time.Minute).Err()
	}

	return results, total, totalPages, nil
}

func getLightweightActivitiesPipeline(filter bson.M, sortField string, sortOrder int, skip int64, limit int64, majors []string, studentYears []int) mongo.Pipeline {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: filter}},

		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "activityItems"},
			{Key: "localField", Value: "_id"},
			{Key: "foreignField", Value: "activityId"},
			{Key: "as", Value: "activityItems"},
		}}},
	}

	// ✅ กรองเฉพาะ Major ที่ต้องการ **ถ้ามีค่า major**
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
	if len(studentYears) > 0 && studentYears[0] != 0 {
		pipeline = append(pipeline, bson.D{
			{Key: "$match", Value: bson.D{
				{Key: "activityItems.studentYears", Value: bson.D{{Key: "$in", Value: studentYears}}},
			}},
		})
	}

	if sortField != "" && (sortOrder == 1 || sortOrder == -1) {
		if sortField == "dates" {
			pipeline = append(pipeline, bson.D{
				{Key: "$addFields", Value: bson.D{
					{Key: "activityItems.firstDate", Value: bson.D{
						{Key: "$arrayElemAt", Value: bson.A{"$activityItems.dates.date", 0}},
					}},
				}},
			})

			pipeline = append(pipeline, bson.D{
				{Key: "$sort", Value: bson.D{{Key: "activityItems.firstDate", Value: sortOrder}}},
			})

		} else {
			pipeline = append(pipeline, bson.D{{Key: "$sort", Value: bson.D{{Key: sortField, Value: sortOrder}}}})
		}
	}

	if skip > 0 {
		pipeline = append(pipeline, bson.D{{Key: "$skip", Value: skip}})
	}
	if limit > 0 {
		pipeline = append(pipeline, bson.D{{Key: "$limit", Value: limit}})
	}

	return pipeline
}

func GetActivityByID(activityID string) (*models.ActivityDto, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(activityID)
	if err != nil {
		return nil, fmt.Errorf("invalid activity ID format")
	}

	var result models.ActivityDto

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

	fmt.Println("activityID:", activityID)
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
		fmt.Println(result)

		// Loop ตรวจสอบ activityItemSums
		cleanedActivityItems := []models.ActivityItemSum{}
		adjustedTotalRegistered := result.TotalRegistered
		for _, item := range result.ActivityItemSums {
			cleanedMajors := []models.MajorEnrollment{}

			for _, major := range item.RegisteredByMajor {
				if major.MajorName != "" {
					cleanedMajors = append(cleanedMajors, major)
				} else {
					// ถ้า MajorName ว่าง → ปรับ totalRegistered และ remainingSlots
					adjustedTotalRegistered -= major.Count
					result.RemainingSlots += major.Count
				}
			}

			// ถ้ามี RegisteredByMajor เหลือ → เก็บไว้
			item.RegisteredByMajor = cleanedMajors
			cleanedActivityItems = append(cleanedActivityItems, item)
		}

		// อัปเดต result ใหม่
		result.ActivityItemSums = cleanedActivityItems
		result.TotalRegistered = adjustedTotalRegistered

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

func UpdateActivity(id primitive.ObjectID, activity models.ActivityDto) (*models.ActivityDto, error) {
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
			"endDateEnroll": activity.EndDateEnroll,
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

	// ✅ วนหาเวลาสิ้นสุดที่มากที่สุด
	var latestTime time.Time

	// isOpen := 0

	for _, newItem := range activity.ActivityItems {
		if newItem.ID.IsZero() {
			// ✅ ถ้าไม่มี `_id` ให้สร้างใหม่
			newItem.ID = primitive.NewObjectID()
			newItem.ActivityID = id
			_, err := activityItemCollection.InsertOne(ctx, newItem)
			if err != nil {
				return nil, err
			}

			// ✅ คำนวณ latestTime
			latestTime = MaxEndTimeFromItem(newItem, latestTime)
		} else {
			// ✅ ถ้ามี `_id` → อัปเดต
			newItemIDs[newItem.ID.Hex()] = true

			_, err := activityItemCollection.UpdateOne(ctx,
				bson.M{"_id": newItem.ID},
				bson.M{"$set": bson.M{
					"activityId":      newItem.ActivityID,
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
			latestTime = MaxEndTimeFromItem(newItem, latestTime)
		}

		// if activity.ActivityState == "open" {
		// 	isOpen += 1
		// }

		// // ✅ ถ้า activityState เปลี่ยนเป็น "open" เพียงแค่ 1 ตัว → ส่งอีเมลหานิสิต
		// if isOpen == 1 {
		// 	// ดึง users ที่ role == student
		// 	userCollection := database.GetCollection("BluelockDB", "users")
		// 	cursor, err := userCollection.Find(ctx, bson.M{"role": "Student"})
		// 	if err != nil {
		// 		return nil, err
		// 	}

		// 	var students []models.User
		// 	if err := cursor.All(ctx, &students); err != nil {
		// 		return nil, err
		// 	}

		// 	// ส่งอีเมลหาแต่ละคน
		// 	for _, student := range students {
		// 		fmt.Println("student", student.Email)
		// 		name := ""
		// 		if activity.Name != nil {
		// 			name = *activity.Name
		// 		}
		// 		subject := fmt.Sprintf("📢 เปิดลงทะเบียนกิจกรรม: %s", name)
		// 		body := fmt.Sprintf(`
		// 		<table style="max-width: 600px; margin: auto; font-family: Arial, sans-serif; border: 1px solid #e0e0e0; border-radius: 8px; box-shadow: 0 2px 5px rgba(0,0,0,0.05); overflow: hidden;">
		// 		  <tr>
		// 			<td style="background-color: #2E86C1; color: white; padding: 20px; text-align: center;">
		// 			  <h2 style="margin: 0;">📢 แจ้งเตือนกิจกรรม</h2>
		// 			</td>
		// 		  </tr>
		// 		  <tr>
		// 			<td style="padding: 24px;">
		// 			  <h3 style="color: #333;">เรียน นิสิต,</h3>
		// 			  <p style="font-size: 16px; color: #555;">
		// 				กิจกรรม <strong style="color: #2E86C1;">%s</strong> ได้เปิดให้ลงทะเบียนแล้ว 🎉
		// 			  </p>
		// 			  <p style="font-size: 16px; color: #555;">
		// 				สามารถเข้าสู่ระบบเพื่อลงทะเบียนได้ทันที โดยคลิกที่ปุ่มด้านล่าง
		// 			  </p>
		// 			  <div style="text-align: center; margin: 30px 0;">
		// 				<a href="%s"
		// 				   style="background-color: #2E86C1; color: white; padding: 12px 24px; border-radius: 6px; text-decoration: none; font-weight: bold; display: inline-block;">
		// 				   📝 ลงทะเบียนกิจกรรม
		// 				</a>
		// 			  </div>
		// 			  <p style="font-size: 14px; color: #888;">หากคุณไม่ได้เป็นผู้รับผิดชอบกิจกรรมนี้ กรุณาเมินเฉยอีเมลนี้</p>
		// 			</td>
		// 		  </tr>
		// 		  <tr>
		// 			<td style="background-color: #f4f4f4; text-align: center; padding: 12px; font-size: 12px; color: #999;">
		// 			  © 2025 Activity Tracking System, Your University
		// 			</td>
		// 		  </tr>
		// 		</table>
		// 	  `, name, fmt.Sprintf("http://localhost:9000/#/Student/Activity/ActivityDetail/%s", id.Hex()))

		// 		fmt.Println("subject", subject)
		// 		fmt.Println("body", body)
		// 		// ✅ ส่งอีเมล (อาจใส่ go routine เพื่อไม่ block)
		// 		// go func(email string) {
		// 		// 	if err := SendEmail(email, subject, body); err != nil {
		// 		// 		fmt.Println("ส่งอีเมลล้มเหลว:", email, err)
		// 		// 	}
		// 		// }(student.Email)
		// 	}
		// }
	}
	if redisURI != "" {
		err = ScheduleChangeActivityStateJob(latestTime, activity.EndDateEnroll, id.Hex())
		if err != nil {
			return nil, err
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
	if err != nil {
		return err
	}

	if redisURI != "" {
		DeleteTask("complete", id.Hex()) // ลบ task ที่เกี่ยวข้อง
		DeleteTask("close", id.Hex())    // ลบ task ที่เกี่ยวข้อง

	}

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

		// 3️⃣ Lookup EnrollmentCount แทนที่จะดึงทั้ง array
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "enrollments"},
			{Key: "let", Value: bson.D{{Key: "itemId", Value: "$activityItems._id"}}},
			{Key: "pipeline", Value: bson.A{
				bson.D{{Key: "$match", Value: bson.D{
					{Key: "$expr", Value: bson.D{
						{Key: "$eq", Value: bson.A{"$activityItemId", "$$itemId"}},
					}},
				}}},
				bson.D{{Key: "$count", Value: "count"}},
			}},
			{Key: "as", Value: "activityItems.enrollmentCountData"},
		}}},

		// 4️⃣ Add enrollmentCount field จาก enrollmentCountData
		{{Key: "$addFields", Value: bson.D{
			{Key: "activityItems.enrollmentCount", Value: bson.D{
				{Key: "$ifNull", Value: bson.A{bson.D{
					{Key: "$arrayElemAt", Value: bson.A{"$activityItems.enrollmentCountData.count", 0}},
				}, 0}},
			}},
		}}},
	}

	// ✅ กรองเฉพาะ Major ที่ต้องการ **ถ้ามีค่า major**
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
func GetEnrollmentByActivityItemID(
	activityItemID primitive.ObjectID,
	pagination models.PaginationParams,
	majors []string,
	status []int,
	studentYears []int,
) ([]bson.M, int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Base aggregation pipeline
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"activityItemId": activityItemID}}},
		{{Key: "$lookup", Value: bson.M{
			"from":         "students",
			"localField":   "studentId",
			"foreignField": "_id",
			"as":           "student",
		}}},
		{{Key: "$unwind", Value: "$student"}},
		{{Key: "$lookup", Value: bson.M{
			"from": "enrollments",
			"let":  bson.M{"studentId": "$student._id"},
			"pipeline": mongo.Pipeline{
				{{"$match", bson.M{
					"$expr": bson.M{
						"$and": bson.A{
							bson.M{"$eq": bson.A{"$studentId", "$$studentId"}},
							bson.M{"$eq": bson.A{"$activityItemId", activityItemID}},
						},
					},
				}}},
			},
			"as": "enrollment",
		}}},

		{{Key: "$unwind", Value: bson.M{
			"path":                       "$enrollment",
			"preserveNullAndEmptyArrays": true,
		}}},
	}

	// Filters
	filter := bson.D{}
	if len(majors) > 0 {
		filter = append(filter, bson.E{Key: "student.major", Value: bson.M{"$in": majors}})
	}
	if len(status) > 0 {
		filter = append(filter, bson.E{Key: "student.status", Value: bson.M{"$in": status}})
	}
	if len(studentYears) > 0 {
		var regexFilters []bson.M
		for _, year := range generateStudentCodeFilter(studentYears) {
			regexFilters = append(regexFilters, bson.M{"student.code": bson.M{"$regex": "^" + year, "$options": "i"}})
		}
		filter = append(filter, bson.E{Key: "$or", Value: regexFilters})
	}
	if pagination.Search != "" {
		regex := bson.M{"$regex": pagination.Search, "$options": "i"}
		filter = append(filter, bson.E{Key: "$or", Value: bson.A{
			bson.M{"student.name": regex},
			bson.M{"student.code": regex},
		}})
	}
	if len(filter) > 0 {
		pipeline = append(pipeline, bson.D{{Key: "$match", Value: filter}})
	}

	// Project student fields
	pipeline = append(pipeline, bson.D{{Key: "$project", Value: bson.M{
		"_id":              0,
		"id":               "$student._id",
		"code":             "$student.code",
		"name":             "$student.name",
		"engName":          "$student.engName",
		"status":           "$student.status",
		"softSkill":        "$student.softSkill",
		"hardSkill":        "$student.hardSkill",
		"major":            "$student.major",
		"food":             "$enrollment.food",
		"registrationDate": "$enrollment.registrationDate",
	}}})

	// Count total before skip/limit
	countPipeline := append(pipeline, bson.D{{Key: "$count", Value: "total"}})
	countCursor, err := enrollmentCollection.Aggregate(ctx, countPipeline)
	if err != nil {
		return nil, 0, err
	}
	defer countCursor.Close(ctx)

	var total int64
	if countCursor.Next(ctx) {
		var countResult struct {
			Total int64 `bson:"total"`
		}
		if err := countCursor.Decode(&countResult); err == nil {
			total = countResult.Total
		}
	}

	// Add pagination
	pipeline = append(pipeline,
		bson.D{{Key: "$skip", Value: (pagination.Page - 1) * pagination.Limit}},
		bson.D{{Key: "$limit", Value: pagination.Limit}},
	)

	cursor, err := enrollmentCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		return nil, 0, err
	}

	return results, total, nil
}

// func GetEnrollmentByActivityID(activityID string, pagination models.PaginationParams, majors []string, status []int, studentYears []int) ([]models.Enrollment, int64, error) {
// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 	defer cancel()

// 	objectID, err := primitive.ObjectIDFromHex(activityID)
// 	if err != nil {
// 		return nil, 0, err
// 	}

// 	pipeline := GetEnrollmentByActivityIDPipeline(objectID, pagination, majors, status, studentYears)
// 	cursor, err := activityItemCollection.Aggregate(ctx, pipeline)
// 	if err != nil {
// 		log.Println("Error fetching enrollments:", err)
// 		return nil, 0, err
// 	}
// 	defer cursor.Close(ctx)

// 	var results []models.Enrollment
// 	if err = cursor.All(ctx, &results); err != nil {
// 		log.Println("Error decoding enrollments:", err)
// 		return nil, 0, err
// 	}

// 	// ใช้ aggregation เพื่อให้ได้นับเฉพาะ enrollments ที่ผ่าน filter จริง ๆ
// 	countPipeline := append(pipeline[:len(pipeline)-2], bson.D{{Key: "$count", Value: "total"}})
// 	countCursor, err := activityItemCollection.Aggregate(ctx, countPipeline)
// 	if err != nil {
// 		log.Println("Error counting enrollments:", err)
// 		return nil, 0, err
// 	}
// 	defer countCursor.Close(ctx)

// 	var countResult struct {
// 		Total int64 `bson:"total"`
// 	}
// 	if countCursor.Next(ctx) {
// 		if err := countCursor.Decode(&countResult); err != nil {
// 			log.Println("Error decoding count result:", err)
// 			return nil, 0, err
// 		}
// 	}

// 	return results, countResult.Total, nil
// }

// func GetEnrollmentByActivityIDPipeline(activityID primitive.ObjectID, pagination models.PaginationParams, majors []string, status []int, studentYears []int) mongo.Pipeline {
// 	pipeline := mongo.Pipeline{
// 		{{Key: "$match", Value: bson.D{{Key: "activityId", Value: activityID}}}},
// 		{{Key: "$lookup", Value: bson.D{
// 			{Key: "from", Value: "enrollments"},
// 			{Key: "localField", Value: "_id"},
// 			{Key: "foreignField", Value: "activityItemId"},
// 			{Key: "as", Value: "enrollments"},
// 		}}},
// 		{{Key: "$unwind", Value: bson.D{
// 			{Key: "path", Value: "$enrollments"},
// 			{Key: "preserveNullAndEmptyArrays", Value: true},
// 		}}},
// 		{{Key: "$lookup", Value: bson.D{
// 			{Key: "from", Value: "students"},
// 			{Key: "localField", Value: "enrollments.studentId"},
// 			{Key: "foreignField", Value: "_id"},
// 			{Key: "as", Value: "enrollments.student"},
// 		}}},
// 		{{Key: "$unwind", Value: bson.D{
// 			{Key: "path", Value: "$enrollments.student"},
// 			{Key: "preserveNullAndEmptyArrays", Value: true},
// 		}}},

// 		// เพิ่ม `$addFields` เพื่อแยก `major` ออกมาก่อนทำ `$match`
// 		{{Key: "$addFields", Value: bson.D{
// 			{Key: "studentMajor", Value: "$enrollments.student.major"},
// 		}}},
// 	}

// 	// Apply filter for student majors if provided
// 	if len(majors) > 0 {
// 		pipeline = append(pipeline, bson.D{{Key: "$match", Value: bson.D{{Key: "studentMajor", Value: bson.M{"$in": majors}}}}})
// 	}

// 	// Apply filter for student status if provided
// 	if len(status) > 0 {
// 		pipeline = append(pipeline, bson.D{{Key: "$match", Value: bson.D{{Key: "enrollments.student.status", Value: bson.M{"$in": status}}}}})
// 	}

// 	// Apply student year filter if provided
// 	if len(studentYears) > 0 {
// 		studentCodePrefixes := generateStudentCodeFilter(studentYears)

// 		var regexFilters []bson.D
// 		for _, prefix := range studentCodePrefixes {
// 			regexFilters = append(regexFilters, bson.D{
// 				{Key: "enrollments.student.code", Value: bson.M{"$regex": "^" + prefix, "$options": "i"}}, // ใช้ ^ ใน "$regex": "^" + prefix เพื่อให้แน่ใจว่า เลขที่ต้องการอยู่ต้นรหัสนิสิต
// 			})
// 		}

// 		pipeline = append(pipeline, bson.D{
// 			{Key: "$match", Value: bson.D{
// 				{Key: "$or", Value: regexFilters}, // ใช้ $or เพื่อรองรับหลายปี เช่น ["67", "66", "65", "64"]
// 			}},
// 		})
// 	}

// 	// Apply search filter if provided
// 	if pagination.Search != "" {
// 		searchRegex := bson.M{"$regex": pagination.Search, "$options": "i"} // Case-insensitive search
// 		pipeline = append(pipeline, bson.D{{Key: "$match", Value: bson.D{
// 			{Key: "$or", Value: bson.A{
// 				bson.D{{Key: "enrollments.student.name", Value: searchRegex}},
// 				bson.D{{Key: "enrollments.student.code", Value: searchRegex}},
// 			}},
// 		}}})
// 	}

// 	pipeline = append(pipeline,
// 		bson.D{{Key: "$project", Value: bson.D{
// 			{Key: "_id", Value: "$enrollments._id"},
// 			{Key: "registrationDate", Value: "$enrollments.registrationDate"},
// 			{Key: "activityItemId", Value: "$enrollments.activityItemId"},
// 			{Key: "studentId", Value: "$enrollments.studentId"},
// 			{Key: "student", Value: "$enrollments.student"},
// 		}}},
// 		bson.D{{Key: "$skip", Value: (pagination.Page - 1) * pagination.Limit}},
// 		bson.D{{Key: "$limit", Value: pagination.Limit}},
// 	)

// 	return pipeline
// }

// 🔢 คำนวณปีการศึกษาปัจจุบัน (พ.ศ.)
func getCurrentAcademicYear() int {
	now := time.Now()        // เวลาปัจจุบัน
	year := now.Year() + 543 // แปลง ค.ศ. เป็น พ.ศ.

	// ถ้ายังไม่ถึงเดือนกรกฎาคม ถือว่ายังเป็นปีการศึกษาที่แล้ว
	if now.Month() < 7 {
		year -= 1
	}
	return year % 100 // ✅ เอาเฉพาะ 2 หลักท้าย (2568 → 68)
}

// 🎯 ฟังก์ชันสำหรับสร้างเงื่อนไขการคัดกรองรหัสนิสิต
func generateStudentCodeFilter(studentYears []int) []string {
	currentYear := getCurrentAcademicYear()
	var codes []string

	for _, year := range studentYears {
		if year >= 1 && year <= 4 {
			studentYearPrefix := strconv.Itoa(currentYear - (year - 1))
			codes = append(codes, studentYearPrefix) // เพิ่ม Prefix 67, 66, 65, 64 ตามปี
		}
	}
	return codes
}

// func SendEmail(to string, subject string, html string) error {
// 	m := gomail.NewMessage()
// 	m.SetHeader("From", "65160205@go.buu.ac.th") // ✅ อีเมลที่ใช้สมัคร Brevo
// 	m.SetHeader("To", to)
// 	m.SetHeader("Subject", subject)
// 	m.SetBody("text/html", html)

// 	d := gomail.NewDialer(
// 		"smtp-relay.brevo.com",
// 		587,
// 		"88bd8f001@smtp-brevo.com",
// 		"EgkJ095wCGS36DfR",
// 	)

// 	return d.DialAndSend(m)
// }

func ScheduleChangeActivityStateJob(latestTime time.Time, endDateEnroll string, activityID string) error {

	if AsynqClient == nil {
		return errors.New("asynq client is not initialized")
	}

	deadline, err := time.ParseInLocation("2006-01-02", endDateEnroll, time.Local)
	if err != nil {
		return err
	}

	// ===== ✅ Schedule complete-activity (latestTime) =====
	if !latestTime.IsZero() && latestTime.After(time.Now()) {
		if err := enqueueTask(
			"complete-activity-"+activityID,
			jobs.NewcompleteActivityTask,
			latestTime,
			activityID,
		); err != nil {
			return err
		}
	} else {
		log.Println("⏩ Skipped complete-activity task (invalid or past time)")
	}

	// ===== ✅ Schedule close-enroll (deadline) =====
	if !deadline.IsZero() && deadline.After(time.Now()) {
		if err := enqueueTask(
			"close-enroll-"+activityID,
			jobs.NewCloseEnrollTask,
			deadline,
			activityID,
		); err != nil {
			return err
		}
	} else {
		log.Println("⏩ Skipped close-enroll task (invalid or past time)")
	}

	return nil
}

func MaxEndTimeFromItem(item models.ActivityItemDto, latestTime time.Time) time.Time {

	loc, err := time.LoadLocation("Asia/Bangkok")
	if err != nil {
		log.Println("❌ Failed to load location:", err)
		return latestTime
	}

	for _, d := range item.Dates {
		t, err := time.ParseInLocation("2006-01-02 15:04", d.Date+" "+d.Etime, loc)
		if err != nil {
			continue // ข้ามกรณีที่เวลา format ผิด
		}
		if t.After(latestTime) {
			latestTime = t
		}
	}

	return latestTime
}

func DeleteTask(taskID string, activityID string) {
	// ✅ ลบ task เดิมก่อน (ถ้ามี)
	fmt.Println("🗑️ Deleting old task:", taskID)
	inspector := asynq.NewInspector(asynq.RedisClientOpt{Addr: redisURI})
	err := inspector.DeleteTask("default", taskID)
	if err != nil && err != asynq.ErrTaskNotFound {
		log.Println("⚠️ Failed to delete old task "+taskID+", then skipping:", err)
		// ไม่ return error → ให้ไปต่อแม้ลบไม่ได้
	} else if err == nil {
		log.Println("🗑️ Deleted previous task:", taskID)
	}
}

func enqueueTask(
	taskID string,
	createFunc func(string) (*asynq.Task, error),
	runAt time.Time,
	activityID string,
) error {
	task, err := createFunc(activityID)
	if err != nil {
		log.Printf("❌ Failed to create task %s: %v", taskID, err)
		return err
	}

	DeleteTask(taskID, activityID)

	_, err = AsynqClient.Enqueue(task, asynq.ProcessAt(runAt), asynq.TaskID(taskID))
	if err != nil {
		log.Printf("❌ Failed to enqueue task %s: %v", taskID, err)
		return err
	}

	log.Printf("✅ Task scheduled: %s | RunAt=%s", taskID, runAt.Format(time.RFC3339))
	return nil
}
