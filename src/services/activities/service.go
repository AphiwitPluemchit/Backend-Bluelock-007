package activities

import (
	DB "Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// --- Redis Cache Helper ---
func hashParams(params interface{}) string {
	b, _ := json.Marshal(params)
	h := sha1.New()
	h.Write(b)
	return hex.EncodeToString(h.Sum(nil))
}

func setCache(key string, value interface{}, ttl time.Duration) {
	if DB.RedisClient == nil {
		return
	}
	b, _ := json.Marshal(value)
	DB.RedisClient.Set(DB.RedisCtx, key, b, ttl)
}

func getCache(key string, dest interface{}) bool {
	if DB.RedisClient == nil {
		return false
	}
	val, err := DB.RedisClient.Get(DB.RedisCtx, key).Result()
	if err != nil {
		return false
	}
	return json.Unmarshal([]byte(val), dest) == nil
}

func delCache(keys ...string) {
	if DB.RedisClient == nil {
		return
	}
	DB.RedisClient.Del(DB.RedisCtx, keys...)
}

func invalidateAllActivitiesListCache() {
	if DB.RedisClient == nil {
		return
	}
	iter := DB.RedisClient.Scan(DB.RedisCtx, 0, "activities:list:*", 0).Iterator()
	for iter.Next(DB.RedisCtx) {
		DB.RedisClient.Del(DB.RedisCtx, iter.Val())
	}
}

var ctx = context.Background()

// CreateActivity - สร้าง Activity และ ActivityItems
func CreateActivity(activity *models.ActivityDto) (*models.ActivityDto, error) {
	// หลังจาก insert DB สำเร็จ ให้ invalidate cache list
	defer invalidateAllActivitiesListCache()

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
	_, err := DB.ActivityCollection.InsertOne(ctx, activityToInsert)
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

		// ✅ คำนวณ latestTime
		latestTime = MaxEndTimeFromItem(item, latestTime)
	}

	// ✅ Insert ทั้งหมดในครั้งเดียว เร็วขึ้นมากในการ insert หลายรายการ ลดจำนวนการ round-trip ไปยัง MongoDB
	_, err = DB.ActivityItemCollection.InsertMany(ctx, itemsToInsert)
	if err != nil {
		return nil, err
	}

	log.Println("Activity and ActivityItems created successfully")

	// Schedule state transitions if activity is created with "open" state
	if DB.AsynqClient != nil && activity.ActivityState == "open" {
		log.Println("✅ Scheduling state transitions for new activity:", activity.ID.Hex())
		err = ScheduleChangeActivityStateJob(DB.AsynqClient, DB.RedisURI, latestTime, activity.EndDateEnroll, activity.ID.Hex())
		if err != nil {
			log.Println("❌ Failed to schedule state transitions for new activity:", err)
			// Don't return error here, just log it - we don't want to fail activity creation
			// if scheduling fails
		}
	}

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
	_, err = DB.ActivityCollection.UpdateOne(context.Background(), filter, update)
	return err
}

// GetAllActivities - ดึง Activity พร้อม ActivityItems + Pagination, Search, Sorting
func GetAllActivities(params models.PaginationParams, skills, states, majors []string, studentYears []int) ([]models.ActivityDto, int64, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := "activities:list:" + hashParams(struct {
		Params       models.PaginationParams
		Skills       []string
		States       []string
		Majors       []string
		StudentYears []int
	}{params, skills, states, majors, studentYears})

	var cached struct {
		Data       []models.ActivityDto
		Total      int64
		TotalPages int
	}
	if getCache(key, &cached) {
		return cached.Data, cached.Total, cached.TotalPages, nil
	}

	filter, isSortNearest := buildActivitiesFilter(params, skills, states)
	skip := int64((params.Page - 1) * params.Limit)
	sortField, sortOrder := getSortFieldAndOrder(params.SortBy, params.Order)

	pipeline := getLightweightActivitiesPipeline(filter, sortField, sortOrder, isSortNearest, skip, int64(params.Limit), majors, studentYears)
	results, err := aggregateActivities(ctx, pipeline)
	if err != nil {
		return nil, 0, 0, err
	}

	total, err := countActivities(ctx, filter, majors, studentYears, isSortNearest)
	if err != nil {
		return nil, 0, 0, err
	}

	populateEnrollmentCounts(ctx, results)
	totalPages := int(math.Ceil(float64(total) / float64(params.Limit)))

	setCache(key, struct {
		Data       []models.ActivityDto
		Total      int64
		TotalPages int
	}{results, total, totalPages}, 5*time.Minute)

	if DB.RedisURI != "" {
		cacheActivitiesResult(key, results, total, totalPages)
	}

	return results, total, totalPages, nil
}

// GetAllActivityCalendar - ดึง Activity และ ActivityItems ตามเดือนและปีที่ระบุ
func GetAllActivityCalendar(month int, year int) ([]models.ActivityDto, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Validate month input
	if month < 1 || month > 12 {
		return nil, fmt.Errorf("invalid month provided: %d", month)
	}
	fmt.Println("month: ", month)
	fmt.Println("year: ", year)

	// Calculate the first and last day of the given month and year
	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, 0).Add(-time.Nanosecond)

	// Format dates to "YYYY-MM-DD" string for matching in MongoDB
	startDateStr := startDate.Format("2006-01-02")
	endDateStr := endDate.Format("2006-01-02")

	// Define the aggregation pipeline
	pipeline := GetAllActivityCalendarPipeline(startDateStr, endDateStr)

	// Execute the pipeline on the 'activityItems' collection
	cursor, err := DB.ActivityItemCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to execute aggregation pipeline: %w", err)
	}
	defer cursor.Close(ctx)

	// Decode the results into a slice of ActivityDto
	var results []models.ActivityDto
	if err = cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode aggregation results: %w", err)
	}

	return results, nil
}

func GetActivityByID(activityID string) (*models.ActivityDto, error) {
	cacheKey := "activity:" + activityID
	var cached models.ActivityDto
	if getCache(cacheKey, &cached) {
		return &cached, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(activityID)
	if err != nil {
		return nil, fmt.Errorf("invalid activity ID format")
	}

	var result models.ActivityDto

	pipeline := GetOneActivityPipeline(objectID)

	cursor, err := DB.ActivityCollection.Aggregate(ctx, pipeline)
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

		setCache(cacheKey, result, 5*time.Minute)
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

	cursor, err := DB.ActivityItemCollection.Aggregate(ctx, pipeline)
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
	cursor, err := DB.ActivityItemCollection.Find(ctx, bson.M{"activityId": activityID})
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
	defer func() {
		invalidateAllActivitiesListCache()
		delCache("activity:" + id.Hex())
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get the old activity to compare states and dates
	var oldActivity models.ActivityDto
	err := DB.ActivityCollection.FindOne(ctx, bson.M{"_id": id}).Decode(&oldActivity)
	if err != nil {
		return nil, err
	}

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

	_, err = DB.ActivityCollection.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		return nil, err
	}

	// ✅ ดึงรายการ `ActivityItems` ที่มีอยู่
	var existingItems []models.ActivityItem
	cursor, err := DB.ActivityItemCollection.Find(ctx, bson.M{"activityId": id})
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

	for _, newItem := range activity.ActivityItems {
		if newItem.ID.IsZero() {
			// ✅ ถ้าไม่มี `_id` ให้สร้างใหม่
			newItem.ID = primitive.NewObjectID()
			newItem.ActivityID = id
			_, err := DB.ActivityItemCollection.InsertOne(ctx, newItem)
			if err != nil {
				return nil, err
			}

			// ✅ คำนวณ latestTime
			latestTime = MaxEndTimeFromItem(newItem, latestTime)
		} else {
			// ✅ ถ้ามี `_id` → อัปเดต
			newItemIDs[newItem.ID.Hex()] = true

			_, err := DB.ActivityItemCollection.UpdateOne(ctx,
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
	}

	// ✅ ลบ `ActivityItems` ที่ไม่มีในรายการใหม่
	for existingID := range existingItemMap {
		if !newItemIDs[existingID] {
			objID, err := primitive.ObjectIDFromHex(existingID) // 🔥 แปลง `string` เป็น `ObjectID`
			if err != nil {
				continue
			}
			_, err = DB.ActivityItemCollection.DeleteOne(ctx, bson.M{"_id": objID})
			if err != nil {
				return nil, err
			}
		}
	}

	// Handle scheduling of state transitions based on activity state changes
	if DB.AsynqClient != nil {
		stateChanged := oldActivity.ActivityState != activity.ActivityState
		datesChanged := oldActivity.EndDateEnroll != activity.EndDateEnroll
		itemsChanged := len(activity.ActivityItems) != len(oldActivity.ActivityItems)

		// Case 1: Activity is set to "open" (either newly or was something else before)
		if activity.ActivityState == "open" {
			// Schedule state transitions when:
			// - State changed to "open" from something else
			// - State was already "open" but dates or items changed
			if stateChanged || datesChanged || itemsChanged {
				log.Println("✅ Scheduling state transitions for activity:", id.Hex())
				err = ScheduleChangeActivityStateJob(DB.AsynqClient, DB.RedisURI, latestTime, activity.EndDateEnroll, activity.ID.Hex())
				if err != nil {
					log.Println("❌ Failed to schedule state transitions:", err)
					return nil, err
				}
			}
		} else if stateChanged && (oldActivity.ActivityState == "open" || oldActivity.ActivityState == "close") {
			// Case 2: Activity was "open" or "close" but manually changed to something else
			// Delete any scheduled jobs since manual intervention takes precedence
			activityIDHex := id.Hex()
			DeleteTask("complete-activity-"+activityIDHex, activityIDHex, DB.RedisURI)
			DeleteTask("close-enroll-"+activityIDHex, activityIDHex, DB.RedisURI)
			log.Println("✅ Removed scheduled jobs due to manual state change for activity:", activityIDHex)
		}
	}

	// ✅ ดึงข้อมูล Activity ที่เพิ่งสร้างเสร็จกลับมาให้ Response ✅
	return GetActivityByID(id.Hex())
}

// DeleteActivity - ลบกิจกรรมและ ActivityItems ที่เกี่ยวข้อง
func DeleteActivity(id primitive.ObjectID) error {
	defer func() {
		invalidateAllActivitiesListCache()
		delCache("activity:" + id.Hex())
	}()

	// ลบ ActivityItems ที่เชื่อมโยงกับ Activity
	_, err := DB.ActivityItemCollection.DeleteMany(ctx, bson.M{"activityId": id})
	if err != nil {
		return err
	}

	// ลบ Activity
	_, err = DB.ActivityCollection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}

	// ลบ scheduled jobs ที่เกี่ยวข้องกับ activity นี้
	if DB.RedisURI != "" {
		activityIDHex := id.Hex()
		// ลบ task ที่เกี่ยวข้องโดยใช้ task ID ที่ถูกต้อง
		DeleteTask("complete-activity-"+activityIDHex, activityIDHex, DB.RedisURI)
		DeleteTask("close-enroll-"+activityIDHex, activityIDHex, DB.RedisURI)
		log.Println("✅ Deleted scheduled jobs for activity:", activityIDHex)
	}

	return nil
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
			"from": "checkInOuts",
			"let":  bson.M{"studentId": "$student._id", "activityItemId": "$activityItemId"},
			"pipeline": mongo.Pipeline{
				{{Key: "$match", Value: bson.M{
					"$expr": bson.M{
						"$and": bson.A{
							bson.M{"$eq": bson.A{"$userId", "$$studentId"}},
							bson.M{"$eq": bson.A{"$activityItemId", "$$activityItemId"}},
						},
					},
				}}},
			},
			"as": "checkInOuts",
		}}},
		{{Key: "$lookup", Value: bson.M{
			"from": "enrollments",
			"let":  bson.M{"studentId": "$student._id"},
			"pipeline": mongo.Pipeline{
				{{Key: "$match", Value: bson.M{
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
		for _, year := range GenerateStudentCodeFilter(studentYears) {
			regexFilters = append(regexFilters, bson.M{"student.code": bson.M{"$regex": "^" + year, "$options": "i"}})
		}
		filter = append(filter, bson.E{Key: "$or", Value: regexFilters})
	}
	if pagination.Search != "" {
		regex := bson.M{"$regex": pagination.Search, "$options": "i"}
		filter = append(filter, bson.E{Key: "$or", Value: bson.A{
			bson.M{"student.code": regex},
		}})
	}
	if len(filter) > 0 {
		pipeline = append(pipeline, bson.D{{Key: "$match", Value: filter}})
	}

	// Project student fields + checkInOuts
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
		"enrollmentId":     "$enrollment._id",
		"food":             "$enrollment.food",
		"registrationDate": "$enrollment.registrationDate",
		"checkInOut":       "$checkInOuts",
	}}})

	// Count total before skip/limit
	countPipeline := append(pipeline, bson.D{{Key: "$count", Value: "total"}})
	countCursor, err := DB.EnrollmentCollection.Aggregate(ctx, countPipeline)
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

	cursor, err := DB.EnrollmentCollection.Aggregate(ctx, pipeline)
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
func GetEnrollmentsByActivityID(
	activityID primitive.ObjectID,
	pagination models.PaginationParams,
	majors []string,
	status []int,
	studentYears []int,
) ([]bson.M, int64, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 1) หา activityItemIds ทั้งหมดของ activity นี้
	itemCur, err := DB.ActivityItemCollection.Find(ctx, bson.M{"activityId": activityID}, options.Find().SetProjection(bson.M{"_id": 1}))
	if err != nil {
		return nil, 0, err
	}
	defer itemCur.Close(ctx)

	var itemIDs []primitive.ObjectID
	for itemCur.Next(ctx) {
		var v struct {
			ID primitive.ObjectID `bson:"_id"`
		}
		if err := itemCur.Decode(&v); err == nil {
			itemIDs = append(itemIDs, v.ID)
		}
	}
	if len(itemIDs) == 0 {
		// ไม่มี item ใน activity นี้
		return []bson.M{}, 0, nil
	}

	// 2) สร้าง pipeline
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"activityItemId": bson.M{"$in": itemIDs}}}},
		// join student
		{{Key: "$lookup", Value: bson.M{
			"from":         "students",
			"localField":   "studentId",
			"foreignField": "_id",
			"as":           "student",
		}}},
		{{Key: "$unwind", Value: "$student"}},
		// join checkInOuts ของแต่ละ enrollment (แยกตาม item)
		{{Key: "$lookup", Value: bson.M{
			"from": "checkInOuts",
			"let":  bson.M{"studentId": "$student._id", "activityItemId": "$activityItemId"},
			"pipeline": mongo.Pipeline{
				{{Key: "$match", Value: bson.M{
					"$expr": bson.M{
						"$and": bson.A{
							bson.M{"$eq": bson.A{"$userId", "$$studentId"}},
							bson.M{"$eq": bson.A{"$activityItemId", "$$activityItemId"}},
						},
					},
				}}},
			},
			"as": "checkInOuts",
		}}},
		// เลือกฟิลด์ที่ต้องใช้ (จาก enrollment ปัจจุบัน)
		{{Key: "$project", Value: bson.M{
			"_id":              0,
			"studentId":        "$student._id",
			"code":             "$student.code",
			"name":             "$student.name",
			"engName":          "$student.engName",
			"status":           "$student.status",
			"softSkill":        "$student.softSkill",
			"hardSkill":        "$student.hardSkill",
			"major":            "$student.major",
			"enrollmentId":     "$_id",
			"food":             "$food",
			"registrationDate": "$registrationDate",
			"checkInOut":       "$checkInOuts",
		}}},
	}

	// 3) ฟิลเตอร์ (ทำหลัง $project เพื่ออ้าง student.xxx ได้ง่าย)
	filter := bson.D{}
	if len(majors) > 0 {
		filter = append(filter, bson.E{Key: "major", Value: bson.M{"$in": majors}})
	}
	if len(status) > 0 {
		filter = append(filter, bson.E{Key: "status", Value: bson.M{"$in": status}})
	}
	if len(studentYears) > 0 {
		var regexFilters []bson.M
		for _, year := range GenerateStudentCodeFilter(studentYears) {
			regexFilters = append(regexFilters, bson.M{"code": bson.M{"$regex": "^" + year, "$options": "i"}})
		}
		filter = append(filter, bson.E{Key: "$or", Value: regexFilters})
	}
	if s := strings.TrimSpace(pagination.Search); s != "" {
		regex := bson.M{"$regex": s, "$options": "i"}
		filter = append(filter, bson.E{Key: "$or", Value: bson.A{
			bson.M{"code": regex},
			bson.M{"name": regex},
		}})
	}
	if len(filter) > 0 {
		pipeline = append(pipeline, bson.D{{Key: "$match", Value: filter}})
	}

	// 4) รวมเป็น "คนละ 1 แถว" (เด็กคนเดียวอาจลงหลาย item)
	pipeline = append(pipeline,
		bson.D{{Key: "$group", Value: bson.M{
			// ใช้ studentId เป็น _id เพื่อหลีกเลี่ยง non-accumulator field error
			"_id": "$studentId",

			// เก็บค่าจากเอกสารแรกในกลุ่ม
			"studentId": bson.M{"$first": "$studentId"},
			"code":      bson.M{"$first": "$code"},
			"name":      bson.M{"$first": "$name"},
			"engName":   bson.M{"$first": "$engName"},
			"status":    bson.M{"$first": "$status"},
			"softSkill": bson.M{"$first": "$softSkill"},
			"hardSkill": bson.M{"$first": "$hardSkill"},
			"major":     bson.M{"$first": "$major"},

			// enrollment อาจต่างกันระหว่าง item
			"food":             bson.M{"$first": "$food"},
			"registrationDate": bson.M{"$min": "$registrationDate"},

			// รวม enrollmentId ทั้งหมด + เก็บตัวแรกไว้เผื่อใช้งาน
			"enrollmentId": bson.M{"$first": "$enrollmentId"},

			// รวมเช็คชื่อทุก item -> จะเป็น array ของ array
			"checkInOutNested": bson.M{"$push": "$checkInOut"},
		}}},

		// flatten checkInOutNested ให้เป็น array เดียว
		bson.D{{Key: "$addFields", Value: bson.M{
			"checkInOut": bson.M{
				"$reduce": bson.M{
					"input":        "$checkInOutNested",
					"initialValue": bson.A{},
					"in": bson.M{
						"$concatArrays": bson.A{"$$value", "$$this"},
					},
				},
			},
		}}},
		// map _id -> id เพื่อให้ออกเหมือน GetEnrollmentByActivityItemID
		bson.D{{Key: "$addFields", Value: bson.M{
			"id": "$_id",
		}}},
		bson.D{{Key: "$project", Value: bson.M{
			"checkInOutNested": 0,
			"_id":              0, // ไม่ต้องการ _id ในผลลัพธ์
		}}},
	)

	// 5) จัดเรียง (sort ได้ตาม field ที่เพิ่ง group มา)
	sortDoc := bson.D{}
	order := 1
	if strings.ToLower(pagination.Order) == "desc" {
		order = -1
	}
	switch pagination.SortBy {
	case "code":
		sortDoc = append(sortDoc, bson.E{Key: "code", Value: order})
	case "name":
		sortDoc = append(sortDoc, bson.E{Key: "name", Value: order})
	case "major":
		sortDoc = append(sortDoc, bson.E{Key: "major", Value: order})
	case "status":
		sortDoc = append(sortDoc, bson.E{Key: "status", Value: order})
	case "registrationDate":
		sortDoc = append(sortDoc, bson.E{Key: "registrationDate", Value: order})
	default:
		sortDoc = append(sortDoc, bson.E{Key: "code", Value: order})
	}
	if len(sortDoc) > 0 {
		pipeline = append(pipeline, bson.D{{Key: "$sort", Value: sortDoc}})
	}

	// 6) นับก่อน skip/limit
	countPipeline := append(append(mongo.Pipeline{}, pipeline...), bson.D{{Key: "$count", Value: "total"}})
	countCursor, err := DB.EnrollmentCollection.Aggregate(ctx, countPipeline)
	if err != nil {
		return nil, 0, err
	}
	defer countCursor.Close(ctx)

	var total int64
	if countCursor.Next(ctx) {
		var c struct {
			Total int64 `bson:"total"`
		}
		if err := countCursor.Decode(&c); err == nil {
			total = c.Total
		}
	}

	// 7) ใส่ pagination
	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	if pagination.Limit <= 0 {
		pagination.Limit = 10
	}
	pipeline = append(pipeline,
		bson.D{{Key: "$skip", Value: (pagination.Page - 1) * pagination.Limit}},
		bson.D{{Key: "$limit", Value: pagination.Limit}},
	)

	// 8) รัน aggregate
	cursor, err := DB.EnrollmentCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		return nil, 0, err
	}

	// map ให้ออกมาเป็นเหมือนโครงสร้างเดิม
	for i := range results {
		results[i]["id"] = results[i]["studentId"]
		delete(results[i], "studentId")
	}

	return results, total, nil
}
