package programs

import (
	DB "Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	hourhistory "Backend-Bluelock-007/src/services/hour-history"

	// "Backend-Bluelock-007/src/services/programs/email"
	"Backend-Bluelock-007/src/services/summary_reports"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"time"

	// email "Backend-Bluelock-007/src/services/programs/email"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

func CreateProgram(program *models.ProgramDto) (*models.ProgramDto, error) {
	defer invalidateAllProgramsListCache()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	program.ID = primitive.NewObjectID()

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

	if _, err := DB.ProgramCollection.InsertOne(ctx, programToInsert); err != nil {
		return nil, err
	}

	var itemsToInsert []any
	var latestTime time.Time

	for _, item := range program.ProgramItems {
		itemsToInsert = append(itemsToInsert, models.ProgramItem{
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
		})
		latestTime = MaxEndTimeFromItem(item, latestTime)
	}

	if len(itemsToInsert) > 0 {
		if _, err := DB.ProgramItemCollection.InsertMany(ctx, itemsToInsert); err != nil {
			return nil, err
		}
	}

	// ⏱️ 2) ตั้ง schedule เปลี่ยนสถานะ (close-enroll / success)
	if DB.AsynqClient != nil && program.ProgramState == "open" {
		progName := ""
		if program.Name != nil {
			progName = *program.Name
		}
		if err := ScheduleChangeProgramStateJob(
			DB.AsynqClient,
			DB.RedisURI,
			latestTime,
			program.EndDateEnroll,
			program.ID.Hex(),
			progName,
		); err != nil {
			log.Println("❌ Failed to schedule state transitions:", err)
			// ไม่ return error เพื่อไม่ให้การสร้างโปรแกรม fail
		}
	}

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

	// ✅ ดึง ProgramItems ของ oldProgram เพื่อเปรียบเทียบ
	var oldProgramItems []models.ProgramItem
	oldCursor, err := DB.ProgramItemCollection.Find(ctx, bson.M{"programId": id})
	if err != nil {
		return nil, err
	}
	if err := oldCursor.All(ctx, &oldProgramItems); err != nil {
		return nil, err
	}
	oldCursor.Close(ctx)

	// ✅ แปลง oldProgramItems เป็น ProgramItemDto เพื่อเปรียบเทียบ
	var oldProgramItemDtos []models.ProgramItemDto
	for _, item := range oldProgramItems {
		oldProgramItemDtos = append(oldProgramItemDtos, models.ProgramItemDto(item))
	}
	oldProgram.ProgramItems = oldProgramItemDtos

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

	// ✅ รวบรวม ProgramItem IDs ที่จะถูกลบ
	var itemsToDelete []primitive.ObjectID
	for existingID := range existingItemMap {
		if !newItemIDs[existingID] {
			objID, err := primitive.ObjectIDFromHex(existingID)
			if err != nil {
				continue
			}
			itemsToDelete = append(itemsToDelete, objID)
		}
	}

	// ✅ ลบข้อมูลที่เกี่ยวข้องกับ ProgramItems ที่จะถูกลบ
	if len(itemsToDelete) > 0 {
		// 1) ดึงข้อมูล Enrollments ที่จะถูกลบเพื่อคำนวณ Summary Reports
		var enrollmentsToDelete []models.Enrollment
		cursor, err := DB.EnrollmentCollection.Find(ctx, bson.M{"programItemId": bson.M{"$in": itemsToDelete}})
		if err == nil {
			if err := cursor.All(ctx, &enrollmentsToDelete); err != nil {
				log.Printf("⚠️ Warning: Failed to fetch enrollments for calculation: %v", err)
			}
		}
		cursor.Close(ctx)

		// 3) ลบ Enrollments ที่เกี่ยวข้องกับ ProgramItems เหล่านี้
		if _, err := DB.EnrollmentCollection.DeleteMany(ctx, bson.M{"programItemId": bson.M{"$in": itemsToDelete}}); err != nil {
			log.Printf("⚠️ Warning: Failed to delete enrollments for programItems: %v", err)
		}

		// 4) ลบ Hour Change Histories ที่เกี่ยวข้องกับ ProgramItems เหล่านี้
		if _, err := DB.HourChangeHistoryCollection.DeleteMany(ctx, bson.M{"enrollmentId": bson.M{"$in": itemsToDelete}}); err != nil {
			log.Printf("⚠️ Warning: Failed to delete hour change histories for programItems: %v", err)
		}

		// 5) หา Dates ที่จะถูกลบ (จาก ProgramItems ที่จะถูกลบ)
		var datesToCheck []string
		dateCursor, err := DB.ProgramItemCollection.Find(ctx, bson.M{"_id": bson.M{"$in": itemsToDelete}}, options.Find().SetProjection(bson.M{"dates.date": 1}))
		if err == nil {
			var items []bson.M
			if err := dateCursor.All(ctx, &items); err == nil {
				for _, item := range items {
					if dates, ok := item["dates"].([]interface{}); ok {
						for _, date := range dates {
							if dateMap, ok := date.(bson.M); ok {
								if dateStr, ok := dateMap["date"].(string); ok {
									datesToCheck = append(datesToCheck, dateStr)
								}
							}
						}
					}
				}
			}
			dateCursor.Close(ctx)
		}

		// 6) ลบ Summary Reports เฉพาะ Dates ที่ไม่มี ProgramItem อื่นใช้
		if len(datesToCheck) > 0 {
			// หา Dates ที่ยังมี ProgramItem อื่นใช้อยู่
			var datesStillInUse []string
			for _, date := range datesToCheck {
				count, err := DB.ProgramItemCollection.CountDocuments(ctx, bson.M{
					"programId":  id,
					"dates.date": date,
					"_id":        bson.M{"$nin": itemsToDelete}, // ไม่นับ ProgramItems ที่กำลังจะลบ
				})
				if err != nil {
					log.Printf("⚠️ Warning: Failed to check date %s: %v", date, err)
					continue
				}
				if count > 0 {
					datesStillInUse = append(datesStillInUse, date)
				}
			}

			// ลบ Summary Reports เฉพาะ Dates ที่ไม่มีใครใช้
			datesToDelete := make([]string, 0)
			for _, date := range datesToCheck {
				found := false
				for _, usedDate := range datesStillInUse {
					if date == usedDate {
						found = true
						break
					}
				}
				if !found {
					datesToDelete = append(datesToDelete, date)
				}
			}

			if len(datesToDelete) > 0 {
				if _, err := DB.SummaryCheckInOutReportsCollection.DeleteMany(ctx, bson.M{
					"programId": id,
					"date":      bson.M{"$in": datesToDelete},
				}); err != nil {
					log.Printf("⚠️ Warning: Failed to delete summary reports for dates: %v", err)
				} else {
					log.Printf("✅ Deleted summary reports for program %s, dates: %v", id.Hex(), datesToDelete)
				}
			}

			if len(datesStillInUse) > 0 {
				log.Printf("ℹ️ Keeping summary reports for program %s, dates: %v (still in use by other program items)", id.Hex(), datesStillInUse)
			}
		}

		// 7) ลบ ProgramItems
		if _, err := DB.ProgramItemCollection.DeleteMany(ctx, bson.M{"_id": bson.M{"$in": itemsToDelete}}); err != nil {
			return nil, err
		}

		log.Printf("✅ Deleted %d program items and related data for program %s", len(itemsToDelete), id.Hex())
	}

	// Handle scheduling of state transitions based on program state changes
	if DB.AsynqClient != nil {
		stateChanged := oldProgram.ProgramState != program.ProgramState
		datesChanged := oldProgram.EndDateEnroll != program.EndDateEnroll
		itemsChanged := len(program.ProgramItems) != len(oldProgram.ProgramItems)

		fmt.Println("State changed:", stateChanged)

		// Case 1: Program is set to "open" (either newly or was something else before)
		if program.ProgramState == "open" {
			// Schedule state transitions when:
			// - State changed to "open" from something else
			// - State was already "open" but dates or items changed
			if stateChanged || datesChanged || itemsChanged {
				log.Println("✅ Scheduling state transitions for program:", id.Hex())
				programName := ""
				if program.Name != nil {
					programName = *program.Name
				}
				err = ScheduleChangeProgramStateJob(DB.AsynqClient, DB.RedisURI, latestTime, program.EndDateEnroll, id.Hex(), programName)

				if err != nil {
					log.Println("❌ Failed to schedule state transitions:", err)
					return nil, err
				}
			}
		} else if stateChanged && (oldProgram.ProgramState == "open" && program.ProgramState == "planning") {
			// Case 2: Program was "open" to "planning" but manually changed to something else
			// Delete any scheduled jobs since manual intervention takes precedence
			programIDHex := id.Hex()
			DeleteTask("close-enroll-"+programIDHex, programIDHex, DB.RedisURI)
			log.Println("✅ Removed scheduled jobs due to manual state change for program:", programIDHex)
		} else if stateChanged && oldProgram.ProgramState == "close" && program.ProgramState == "success" {
			// Case 3: Program was "close" and is now "success" (completed)
			programIDHex := id.Hex()
			DeleteTask("complete-program-"+programIDHex, programIDHex, DB.RedisURI)
			log.Println("✅ Ensured no scheduled jobs for success program:", programIDHex)
			// update student enrollment hours history
			if err := hourhistory.ProcessEnrollmentsForCompletedProgram(ctx, id); err != nil {
				log.Printf("⚠️ Warning: failed to process enrollments for program %s: %v", id.Hex(), err)
				// don't return error - admin manual completion should succeed even if hour processing fails
			}
		}
	}

	// newState := strings.ToLower(program.ProgramState)
	// oldState := strings.ToLower(oldProgram.ProgramState)

	// if oldState != "open" && newState == "open" {
	// 	progName := ""
	// 	if program.Name != nil {
	// 		progName = *program.Name
	// 	}

	// 	email.NotifyStudentsOnOpen(
	// 		id.Hex(),
	// 		progName,
	// 		GetProgramByID,
	// 		GenerateStudentCodeFilter,
	// 	)
	// }

	// updated, err := GetProgramByID(id.Hex())
	// if err != nil {
	// 	return nil, err
	// }
	// email.ScheduleReminderJobs(updated)

	// err = updateSummaryReportsForProgramChanges(id, &oldProgram, &program)
	// if err != nil {
	// 	log.Printf("⚠️ Warning: Failed to update summary reports for program changes: %v", err)
	// }

	// ✅ ดึงข้อมูล Program ที่เพิ่งสร้างเสร็จกลับมาให้ Response ✅
	return GetProgramByID(id.Hex())
}

// DeleteProgram - ลบกิจกรรมและ ProgramItems ที่เกี่ยวข้อง
func DeleteProgram(id primitive.ObjectID) error {
	defer func() {
		invalidateAllProgramsListCache()
		delCache("program:" + id.Hex())
	}()

	// 1) หา ProgramItem IDs ทั้งหมดของโปรแกรมนี้
	itemCursor, err := DB.ProgramItemCollection.Find(ctx, bson.M{"programId": id}, options.Find().SetProjection(bson.M{"_id": 1}))
	if err != nil {
		return err
	}
	var itemIDs []primitive.ObjectID
	for itemCursor.Next(ctx) {
		var v struct {
			ID primitive.ObjectID `bson:"_id"`
		}
		if derr := itemCursor.Decode(&v); derr == nil {
			itemIDs = append(itemIDs, v.ID)
		}
	}
	itemCursor.Close(ctx)

	// 2) ลบ Enrollments ที่อยู่ใน ProgramItems ของโปรแกรมนี้
	if len(itemIDs) > 0 {
		if _, err := DB.EnrollmentCollection.DeleteMany(ctx, bson.M{"programItemId": bson.M{"$in": itemIDs}}); err != nil {
			return err
		}
	}

	// 3) ลบสรุปรายงานเช็คอินเช็คเอาท์ของโปรแกรมนี้
	if err := summary_reports.DeleteAllSummaryReportsForProgram(id); err != nil {
		// log แล้วไปต่อ เพื่อไม่ให้การลบหลักพัง
		log.Printf("⚠️ Warning: Failed to delete summary reports for program %s: %v", id.Hex(), err)
	}

	// 4) ลบประวัติการเปลี่ยนแปลงชั่วโมงที่มาจากโปรแกรมนี้
	if _, err := DB.HourChangeHistoryCollection.DeleteMany(ctx, bson.M{"sourceType": "program", "sourceId": id}); err != nil {
		return err
	}

	// 5) ลบ ProgramItems ที่เชื่อมโยงกับ Program
	_, err = DB.ProgramItemCollection.DeleteMany(ctx, bson.M{"programId": id})
	if err != nil {
		return err
	}

	// 6) ลบ Program
	_, err = DB.ProgramCollection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}

	// 7) ลบ scheduled jobs ที่เกี่ยวข้องกับ program นี้
	if DB.RedisURI != "" {
		programIDHex := id.Hex()
		// ลบ task ที่เกี่ยวข้องโดยใช้ task ID ที่ถูกต้อง
		DeleteTask("complete-program-"+programIDHex, programIDHex, DB.RedisURI)
		DeleteTask("close-enroll-"+programIDHex, programIDHex, DB.RedisURI)
		log.Println("✅ Deleted scheduled jobs for program:", programIDHex)
	}

	return nil
}
