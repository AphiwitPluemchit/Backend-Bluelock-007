package activities

import (
	"Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"time"

	"github.com/hibiken/asynq"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var ctx = context.Background()

var activityCollection *mongo.Collection
var activityItemCollection *mongo.Collection
var enrollmentCollection *mongo.Collection
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
	enrollmentCollection = database.GetCollection("BluelockDB", "enrollments")

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
		// Schedule job (helper.go)
		err = ScheduleChangeActivityStateJob(AsynqClient, redisURI, latestTime, activity.EndDateEnroll, activity.ID.Hex())
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
func GetAllActivities(params models.PaginationParams, skills, states, majors []string, studentYears []int) ([]models.ActivityDto, int64, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := buildActivitiesCacheKey(params, skills, states, majors, studentYears)

	if redisURI != "" {
		if cached, err := getActivitiesFromCache(key); err == nil && cached != nil {
			return cached.Data, cached.Total, cached.TotalPages, nil
		}
	}

	filter := buildActivitiesFilter(params, skills, states)
	skip := int64((params.Page - 1) * params.Limit)
	sortField, sortOrder := getSortFieldAndOrder(params.SortBy, params.Order)

	pipeline := getLightweightActivitiesPipeline(filter, sortField, sortOrder, skip, int64(params.Limit), majors, studentYears)
	results, err := aggregateActivities(ctx, pipeline)
	if err != nil {
		return nil, 0, 0, err
	}

	total, err := countActivities(ctx, filter, majors, studentYears, params.Limit)
	if err != nil {
		return nil, 0, 0, err
	}

	populateEnrollmentCounts(ctx, results)
	totalPages := int(math.Ceil(float64(total) / float64(params.Limit)))

	if redisURI != "" {
		cacheActivitiesResult(key, results, total, totalPages)
	}

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
		// Schedule job (helper.go)
		err = ScheduleChangeActivityStateJob(AsynqClient, redisURI, latestTime, activity.EndDateEnroll, id.Hex())
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
		DeleteTask("complete", id.Hex(), redisURI) // ลบ task ที่เกี่ยวข้อง
		DeleteTask("close", id.Hex(), redisURI)    // ลบ task ที่เกี่ยวข้อง

	}

	return err
}
