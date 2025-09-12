package programs

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

func invalidateAllProgramsListCache() {
	if DB.RedisClient == nil {
		return
	}
	iter := DB.RedisClient.Scan(DB.RedisCtx, 0, "programs:list:*", 0).Iterator()
	for iter.Next(DB.RedisCtx) {
		DB.RedisClient.Del(DB.RedisCtx, iter.Val())
	}
}

var ctx = context.Background()

// CreateProgram - สร้าง Program และ ProgramItems
func CreateProgram(program *models.ProgramDto) (*models.ProgramDto, error) {
	// หลังจาก insert DB สำเร็จ ให้ invalidate cache list
	defer invalidateAllProgramsListCache()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// ✅ สร้าง ID สำหรับ Program
	program.ID = primitive.NewObjectID()

	// ✅ สร้าง Program ที่ต้องบันทึกลง MongoDB
	programToInsert := models.Program{
		ID:            program.ID,
		FormID:        program.FormID,
		Name:          program.Name,
		Type:          program.Type,
		ProgramState:  program.ProgramState,
		Skill:         program.Skill,
		File:          program.File,
		FoodVotes:     program.FoodVotes,
		EndDateEnroll: program.EndDateEnroll,
	}

	// ✅ บันทึก Program และรับค่า InsertedID กลับมา
	_, err := DB.ProgramCollection.InsertOne(ctx, programToInsert)
	if err != nil {
		return nil, err
	}

	// ✅ บันทึก ProgramItems
	var itemsToInsert []any

	// ✅ วนหาเวลาสิ้นสุดที่มากที่สุด
	var latestTime time.Time

	for _, item := range program.ProgramItems {
		itemToInsert := models.ProgramItem{
			ID:              primitive.NewObjectID(),
			ProgramID:       program.ID,
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
	_, err = DB.ProgramItemCollection.InsertMany(ctx, itemsToInsert)
	if err != nil {
		return nil, err
	}

	log.Println("Program and ProgramItems created successfully")

	// Schedule state transitions if program is created with "open" state
	if DB.AsynqClient != nil && program.ProgramState == "open" {
		log.Println("✅ Scheduling state transitions for new program:", program.ID.Hex())
		err = ScheduleChangeProgramStateJob(DB.AsynqClient, DB.RedisURI, latestTime, program.EndDateEnroll, program.ID.Hex())
		if err != nil {
			log.Println("❌ Failed to schedule state transitions for new program:", err)
			// Don't return error here, just log it - we don't want to fail program creation
			// if scheduling fails
		}
	}

	// ✅ ดึงข้อมูล Program ที่เพิ่งสร้างเสร็จกลับมาให้ Response ✅
	return GetProgramByID(program.ID.Hex())
}

func UploadProgramImage(programID string, fileName string) error {
	// string to primitive.ObjectID
	objectID, err := primitive.ObjectIDFromHex(programID)
	if err != nil {
		return err
	}

	// update image
	filter := bson.M{"_id": objectID}
	update := bson.M{"$set": bson.M{"file": fileName}}
	_, err = DB.ProgramCollection.UpdateOne(context.Background(), filter, update)
	return err
}

// GetAllPrograms - ดึง Program พร้อม ProgramItems + Pagination, Search, Sorting
func GetAllPrograms(params models.PaginationParams, skills, states, majors []string, studentYears []int) ([]models.ProgramDto, int64, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := "programs:list:" + hashParams(struct {
		Params       models.PaginationParams
		Skills       []string
		States       []string
		Majors       []string
		StudentYears []int
	}{params, skills, states, majors, studentYears})

	var cached struct {
		Data       []models.ProgramDto
		Total      int64
		TotalPages int
	}
	if getCache(key, &cached) {
		return cached.Data, cached.Total, cached.TotalPages, nil
	}

	filter, isSortNearest := buildProgramsFilter(params, skills, states)
	skip := int64((params.Page - 1) * params.Limit)
	sortField, sortOrder := getSortFieldAndOrder(params.SortBy, params.Order)

	pipeline := getLightweightProgramsPipeline(filter, sortField, sortOrder, isSortNearest, skip, int64(params.Limit), majors, studentYears)

	results, err := aggregatePrograms(ctx, pipeline)
	if err != nil {
		return nil, 0, 0, err
	}

	total, err := countPrograms(ctx, filter, majors, studentYears, isSortNearest)
	if err != nil {
		return nil, 0, 0, err
	}

	populateEnrollmentCounts(ctx, results)
	totalPages := int(math.Ceil(float64(total) / float64(params.Limit)))

	setCache(key, struct {
		Data       []models.ProgramDto
		Total      int64
		TotalPages int
	}{results, total, totalPages}, 5*time.Minute)

	if DB.RedisURI != "" {
		cacheProgramsResult(key, results, total, totalPages)
	}

	return results, total, totalPages, nil
}

// GetAllProgramCalendar - ดึง Program และ ProgramItems ตามเดือนและปีที่ระบุ
func GetAllProgramCalendar(month int, year int) ([]models.ProgramDto, error) {
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
	pipeline := GetAllProgramCalendarPipeline(startDateStr, endDateStr)

	// Execute the pipeline on the 'programItems' collection
	cursor, err := DB.ProgramItemCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to execute aggregation pipeline: %w", err)
	}
	defer cursor.Close(ctx)

	// Decode the results into a slice of ProgramDto
	var results []models.ProgramDto
	if err = cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode aggregation results: %w", err)
	}

	return results, nil
}

func GetProgramByID(programID string) (*models.ProgramDto, error) {
	// cacheKey := "program:" + programID
	// var cached models.ProgramDto
	// if getCache(cacheKey, &cached) {
	// 	return &cached, nil
	// }
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(programID)
	if err != nil {
		return nil, fmt.Errorf("invalid program ID format")
	}

	var result models.ProgramDto

	pipeline := GetOneProgramPipeline(objectID)

	cursor, err := DB.ProgramCollection.Aggregate(ctx, pipeline)
	if err != nil {
		log.Println("Error fetching program by ID:", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			log.Println("Error decoding program:", err)
			return nil, err
		}

		// setCache(cacheKey, result, 5*time.Minute)
		return &result, nil
	}

	return nil, fmt.Errorf("program not found")
}

func GetProgramEnrollSummary(programID string) (models.EnrollmentSummary, error) {

	fmt.Println("programID:", programID)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(programID)
	if err != nil {
		return models.EnrollmentSummary{}, err
	}

	var result models.EnrollmentSummary

	pipeline := GetProgramStatisticsPipeline(objectID)

	cursor, err := DB.ProgramItemCollection.Aggregate(ctx, pipeline)
	if err != nil {
		log.Println("Error fetching program by ID:", err)
		return result, err
	}
	defer cursor.Close(ctx)

	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			log.Println("Error decoding program:", err)
			return result, err
		}
		fmt.Println(result)

		// Loop ตรวจสอบ programItemSums
		cleanedProgramItems := []models.ProgramItemSum{}
		adjustedTotalRegistered := result.TotalRegistered
		for _, item := range result.ProgramItemSums {
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
			cleanedProgramItems = append(cleanedProgramItems, item)
		}

		// อัปเดต result ใหม่
		result.ProgramItemSums = cleanedProgramItems
		result.TotalRegistered = adjustedTotalRegistered

		return result, nil
	}

	return result, err
}

// GetProgramItemsByProgramID - ดึง ProgramItems ตาม ProgramID
func GetProgramItemsByProgramID(programID primitive.ObjectID) ([]models.ProgramItem, error) {
	var programItems []models.ProgramItem
	cursor, err := DB.ProgramItemCollection.Find(ctx, bson.M{"programId": programID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var programItem models.ProgramItem
		if err := cursor.Decode(&programItem); err != nil {
			return nil, err
		}
		programItems = append(programItems, programItem)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return programItems, nil
}

func UpdateProgram(id primitive.ObjectID, program models.ProgramDto) (*models.ProgramDto, error) {
	defer func() {
		invalidateAllProgramsListCache()
		delCache("program:" + id.Hex())
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get the old program to compare states and dates
	var oldProgram models.ProgramDto
	err := DB.ProgramCollection.FindOne(ctx, bson.M{"_id": id}).Decode(&oldProgram)
	if err != nil {
		return nil, err
	}

	// ✅ อัปเดต Program หลัก
	update := bson.M{
		"$set": bson.M{
			"name":          program.Name,
			"formId":        program.FormID,
			"type":          program.Type,
			"programState":  program.ProgramState,
			"skill":         program.Skill,
			"file":          program.File,
			"foodVotes":     program.FoodVotes,
			"endDateEnroll": program.EndDateEnroll,
		},
	}

	_, err = DB.ProgramCollection.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		return nil, err
	}

	// ✅ ดึงรายการ `ProgramItems` ที่มีอยู่
	var existingItems []models.ProgramItem
	cursor, err := DB.ProgramItemCollection.Find(ctx, bson.M{"programId": id})
	if err != nil {
		return nil, err
	}
	if err := cursor.All(ctx, &existingItems); err != nil {
		return nil, err
	}

	// ✅ สร้าง Map ของ `existingItems` เพื่อเช็คว่าตัวไหนมีอยู่แล้ว
	existingItemMap := make(map[string]models.ProgramItem)
	for _, item := range existingItems {
		existingItemMap[item.ID.Hex()] = item
	}

	// ✅ สร้าง `Set` สำหรับเก็บ `ID` ของรายการใหม่
	newItemIDs := make(map[string]bool)

	// ✅ วนหาเวลาสิ้นสุดที่มากที่สุด
	var latestTime time.Time

	for _, newItem := range program.ProgramItems {
		if newItem.ID.IsZero() {
			// ✅ ถ้าไม่มี `_id` ให้สร้างใหม่
			newItem.ID = primitive.NewObjectID()
			newItem.ProgramID = id
			_, err := DB.ProgramItemCollection.InsertOne(ctx, newItem)
			if err != nil {
				return nil, err
			}

			// ✅ คำนวณ latestTime
			latestTime = MaxEndTimeFromItem(newItem, latestTime)
		} else {
			// ✅ ถ้ามี `_id` → อัปเดต
			newItemIDs[newItem.ID.Hex()] = true

			_, err := DB.ProgramItemCollection.UpdateOne(ctx,
				bson.M{"_id": newItem.ID},
				bson.M{"$set": bson.M{
					"programId":       newItem.ProgramID,
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

	// ✅ ลบ `ProgramItems` ที่ไม่มีในรายการใหม่
	for existingID := range existingItemMap {
		if !newItemIDs[existingID] {
			objID, err := primitive.ObjectIDFromHex(existingID) // 🔥 แปลง `string` เป็น `ObjectID`
			if err != nil {
				continue
			}
			_, err = DB.ProgramItemCollection.DeleteOne(ctx, bson.M{"_id": objID})
			if err != nil {
				return nil, err
			}
		}
	}

	// Handle scheduling of state transitions based on program state changes
	if DB.AsynqClient != nil {
		stateChanged := oldProgram.ProgramState != program.ProgramState
		datesChanged := oldProgram.EndDateEnroll != program.EndDateEnroll
		itemsChanged := len(program.ProgramItems) != len(oldProgram.ProgramItems)

		// Case 1: Program is set to "open" (either newly or was something else before)
		if program.ProgramState == "open" {
			// Schedule state transitions when:
			// - State changed to "open" from something else
			// - State was already "open" but dates or items changed
			if stateChanged || datesChanged || itemsChanged {
				log.Println("✅ Scheduling state transitions for program:", id.Hex())
				err = ScheduleChangeProgramStateJob(DB.AsynqClient, DB.RedisURI, latestTime, program.EndDateEnroll, program.ID.Hex())
				if err != nil {
					log.Println("❌ Failed to schedule state transitions:", err)
					return nil, err
				}
			}
		} else if stateChanged && (oldProgram.ProgramState == "open" || oldProgram.ProgramState == "close") {
			// Case 2: Program was "open" or "close" but manually changed to something else
			// Delete any scheduled jobs since manual intervention takes precedence
			programIDHex := id.Hex()
			DeleteTask("complete-program-"+programIDHex, programIDHex, DB.RedisURI)
			DeleteTask("close-enroll-"+programIDHex, programIDHex, DB.RedisURI)
			log.Println("✅ Removed scheduled jobs due to manual state change for program:", programIDHex)
		}
	}

	// ✅ ดึงข้อมูล Program ที่เพิ่งสร้างเสร็จกลับมาให้ Response ✅
	return GetProgramByID(id.Hex())
}

// DeleteProgram - ลบกิจกรรมและ ProgramItems ที่เกี่ยวข้อง
func DeleteProgram(id primitive.ObjectID) error {
	defer func() {
		invalidateAllProgramsListCache()
		delCache("program:" + id.Hex())
	}()

	// ลบ ProgramItems ที่เชื่อมโยงกับ Program
	_, err := DB.ProgramItemCollection.DeleteMany(ctx, bson.M{"programId": id})
	if err != nil {
		return err
	}

	// ลบ Program
	_, err = DB.ProgramCollection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}

	// ลบ scheduled jobs ที่เกี่ยวข้องกับ program นี้
	if DB.RedisURI != "" {
		programIDHex := id.Hex()
		// ลบ task ที่เกี่ยวข้องโดยใช้ task ID ที่ถูกต้อง
		DeleteTask("complete-program-"+programIDHex, programIDHex, DB.RedisURI)
		DeleteTask("close-enroll-"+programIDHex, programIDHex, DB.RedisURI)
		log.Println("✅ Deleted scheduled jobs for program:", programIDHex)
	}

	return nil
}
func GetEnrollmentByProgramItemID(
	programItemID primitive.ObjectID,
	pagination models.PaginationParams,
	majors []string,
	status []int,
	studentYears []int,
) ([]bson.M, int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Base aggregation pipeline
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"programItemId": programItemID}}},
		{{Key: "$lookup", Value: bson.M{
			"from":         "Students",
			"localField":   "studentId",
			"foreignField": "_id",
			"as":           "student",
		}}},
		{{Key: "$unwind", Value: "$student"}},
		{{Key: "$lookup", Value: bson.M{
			"from": "Check_In_Check_Out",
			"let":  bson.M{"studentId": "$student._id", "programItemId": "$programItemId"},
			"pipeline": mongo.Pipeline{
				{{Key: "$match", Value: bson.M{
					"$expr": bson.M{
						"$and": bson.A{
							bson.M{"$eq": bson.A{"$studentId", "$$studentId"}},
							bson.M{"$eq": bson.A{"$programItemId", "$$programItemId"}},
						},
					},
				}}},
			},
			"as": "checkInOuts",
		}}},
		{{Key: "$lookup", Value: bson.M{
			"from": "Enrollments",
			"let":  bson.M{"studentId": "$student._id"},
			"pipeline": mongo.Pipeline{
				{{Key: "$match", Value: bson.M{
					"$expr": bson.M{
						"$and": bson.A{
							bson.M{"$eq": bson.A{"$studentId", "$$studentId"}},
							bson.M{"$eq": bson.A{"$programItemId", programItemID}},
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
func GetEnrollmentsByProgramID(
	programID primitive.ObjectID,
	pagination models.PaginationParams,
	majors []string,
	status []int,
	studentYears []int,
) ([]bson.M, int64, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 1) หา programItemIds ทั้งหมดของ program นี้
	itemCur, err := DB.ProgramItemCollection.Find(ctx, bson.M{"programId": programID}, options.Find().SetProjection(bson.M{"_id": 1}))
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
		// ไม่มี item ใน program นี้
		return []bson.M{}, 0, nil
	}

	// 2) สร้าง pipeline
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"programItemId": bson.M{"$in": itemIDs}}}},
		// join student
		{{Key: "$lookup", Value: bson.M{
			"from":         "Students",
			"localField":   "studentId",
			"foreignField": "_id",
			"as":           "student",
		}}},
		{{Key: "$unwind", Value: "$student"}},
		// join checkInOuts ของแต่ละ enrollment (แยกตาม item)
		{{Key: "$lookup", Value: bson.M{
			"from": "Check_In_Check_Out",
			"let":  bson.M{"studentId": "$student._id", "programItemId": "$programItemId"},
			"pipeline": mongo.Pipeline{
				{{Key: "$match", Value: bson.M{
					"$expr": bson.M{
						"$and": bson.A{
							bson.M{"$eq": bson.A{"$studentId", "$$studentId"}},
							bson.M{"$eq": bson.A{"$programItemId", "$$programItemId"}},
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
		// map _id -> id เพื่อให้ออกเหมือน GetEnrollmentByProgramItemID
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
