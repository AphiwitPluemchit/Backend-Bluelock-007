package programs

import (
	DB "Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/jobs"
	"Backend-Bluelock-007/src/models"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/hibiken/asynq"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Set process default location to Asia/Bangkok once on package init.
// This makes time.Local point to Bangkok so time.Parse/Now/ comparisons
// that rely on the local location will use Asia/Bangkok consistently.
func init() {
	loc, err := time.LoadLocation("Asia/Bangkok")
	if err != nil {
		log.Println("⚠️ Failed to load Asia/Bangkok location, leaving time.Local as-is:", err)
		return
	}
	// Override the package-local default timezone for the running process.
	time.Local = loc
	log.Println("✅ Set process timezone to Asia/Bangkok (time.Local)")
}

// Helper functions for programs service

// ===== Pipeline Helper Functions =====

// 🔢 คำนวณปีการศึกษาปัจจุบัน (พ.ศ.)
func GetCurrentAcademicYear() int {
	now := time.Now()        // เวลาปัจจุบัน
	year := now.Year() + 543 // แปลง ค.ศ. เป็น พ.ศ.

	// ถ้ายังไม่ถึงเดือนกรกฎาคม ถือว่ายังเป็นปีการศึกษาที่แล้ว
	if now.Month() < 7 {
		year -= 1
	}
	return year % 100 // ✅ เอาเฉพาะ 2 หลักท้าย (2568 → 68)
}

// 🎯 ฟังก์ชันสำหรับสร้างเงื่อนไขการคัดกรองรหัสนิสิต
func GenerateStudentCodeFilter(studentYears []int) []string {
	currentYear := GetCurrentAcademicYear()
	var codes []string

	for _, year := range studentYears {
		if year >= 1 && year <= 4 {
			studentYearPrefix := strconv.Itoa(currentYear - (year - 1))
			codes = append(codes, studentYearPrefix) // เพิ่ม Prefix 67, 66, 65, 64 ตามปี
		}
	}
	return codes
}

// MaxEndTimeFromItem คำนวณเวลาสิ้นสุดที่มากที่สุดจาก ProgramItemDto
func MaxEndTimeFromItem(item models.ProgramItemDto, latestTime time.Time) time.Time {
	// Use process local timezone (set in init) for parsing/comparisons.
	loc := time.Local
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

// ===== Asynq/Task Helper Functions =====

// DeleteTask ลบ task เดิมก่อน (ถ้ามี)
func DeleteTask(taskID string, programID string, redisURI string) {
	fmt.Println("🗑️ Deleting old task:", taskID)
	inspector := asynq.NewInspector(asynq.RedisClientOpt{Addr: redisURI})
	err := inspector.DeleteTask("default", taskID)
	if err != nil && err != asynq.ErrTaskNotFound {
		log.Println("⚠️ Failed to delete old task "+taskID+", then skipping:", err)
	} else if err == nil {
		log.Println("🗑️ Deleted previous task:", taskID)
	}
}

// enqueueTask สร้างและ enqueue งานใหม่
func enqueueTask(
	AsynqClient *asynq.Client,
	taskID string,
	createFunc func(string) (*asynq.Task, error),
	runAt time.Time,
	programID string,
	redisURI string,
) error {
	task, err := createFunc(programID)
	if err != nil {
		log.Printf("❌ Failed to create task %s: %v", taskID, err)
		return err
	}
	DeleteTask(taskID, programID, redisURI)
	_, err = AsynqClient.Enqueue(task, asynq.ProcessAt(runAt), asynq.TaskID(taskID))
	if err != nil {
		log.Printf("❌ Failed to enqueue task %s: %v", taskID, err)
		return err
	}
	log.Printf("✅ Task scheduled: %s | RunAt=%s", taskID, runAt.Format(time.RFC3339))
	return nil
}

// ScheduleChangeProgramStateJob สร้าง schedule งานเปลี่ยนสถานะ program
func ScheduleChangeProgramStateJob(AsynqClient *asynq.Client, redisURI string, latestTime time.Time, endDateEnroll string, programID string) error {
	if AsynqClient == nil {
		return errors.New("asynq client is not initialized")
	}

	// First, delete any existing scheduled jobs for this program
	// This ensures we don't have duplicate jobs when program is updated
	DeleteTask("complete-program-"+programID, programID, redisURI)
	DeleteTask("close-enroll-"+programID, programID, redisURI)

	// Schedule the "complete" state transition at the latest program end time
	nowLocal := time.Now().In(time.Local)
	if !latestTime.IsZero() && latestTime.After(nowLocal) {
		if err := enqueueTask(
			AsynqClient,
			"complete-program-"+programID,
			jobs.NewCompleteProgramTask,
			latestTime.In(time.Local).Add(time.Hour*1), // เพิ่ม 1 ชั่วโมง
			programID,
			redisURI,
		); err != nil {
			return err
		}
		log.Println("✅ Complete-program task scheduled: complete-program-" + programID)
	} else {
		log.Println("⏩ Skipped complete-program task (invalid or past time)")
	}

	// Schedule the "close" state transition at the endDateEnroll
	// Parse endDateEnroll in process local timezone (Bangkok)
	deadline, err := time.ParseInLocation("2006-01-02", endDateEnroll, time.Local)
	if err != nil {
		log.Println("⚠️ Invalid endDateEnroll format:", endDateEnroll, err)
		return err
	}

	// Add end of day time to ensure it runs at the end of the enrollment day
	deadline = time.Date(deadline.Year(), deadline.Month(), deadline.Day(), 23, 59, 59, 0, deadline.Location())
	log.Println("Enrollment deadline:", deadline.Format(time.RFC3339))

	// Only schedule if deadline is in the future
	if !deadline.IsZero() && deadline.After(time.Now().In(time.Local)) {
		if err := enqueueTask(
			AsynqClient,
			"close-enroll-"+programID,
			jobs.NewCloseEnrollTask,
			deadline,
			programID,
			redisURI,
		); err != nil {
			return err
		}

		log.Println("✅ Close-enroll task scheduled: close-enroll-" + programID)
	} else {
		log.Println("⏩ Skipped close-enroll task (invalid or past time)")
	}
	return nil
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

// --- Helper functions for GetAllPrograms ---

func buildProgramsCacheKey(params models.PaginationParams, skills, states, majors []string, studentYears []int) string {
	return fmt.Sprintf(
		"programs:page=%d&limit=%d&search=%s&sortBy=%s&order=%s&skills=%v&states=%v&majors=%v&years=%v",
		params.Page, params.Limit, params.Search, params.SortBy, params.Order,
		skills, states, majors, studentYears,
	)
}

type programsCache struct {
	Data       []models.ProgramDto `json:"data"`
	Total      int64               `json:"total"`
	TotalPages int                 `json:"totalPages"`
}

func getProgramsFromCache(key string) (*programsCache, error) {
	if DB.RedisClient == nil {
		return nil, errors.New("redis not available")
	}
	cached, err := DB.RedisClient.Get(DB.RedisCtx, key).Result()
	if err != nil {
		return nil, err
	}
	var cachedResult programsCache
	if err := json.Unmarshal([]byte(cached), &cachedResult); err != nil {
		return nil, err
	}
	return &cachedResult, nil
}

func buildProgramsFilter(params models.PaginationParams, skills, states []string) (bson.M, bool) {
	filter := bson.M{}
	isSortNearest := false
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
		filter["programState"] = bson.M{"$in": states}
		// if state only contains "open" and "close", we want to sort by nearest date
		if len(states) == 2 && containsString(states, "open") && containsString(states, "close") {
			isSortNearest = true
		}
	}
	return filter, isSortNearest
}

func getSortFieldAndOrder(sortBy, order string) (string, int) {
	field := sortBy
	if field == "" {
		field = "dates"
	}
	ord := 1
	if strings.ToLower(order) == "desc" {
		ord = -1
	}
	return field, ord
}

func aggregatePrograms(ctx context.Context, pipeline mongo.Pipeline) ([]models.ProgramDto, error) {
	var results []models.ProgramDto
	cursor, err := DB.ProgramCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}
	return results, nil
}

func countPrograms(ctx context.Context, filter bson.M, majors []string, studentYears []int, isSortNearest bool) (int64, error) {
	countPipeline := getLightweightProgramsPipeline(filter, "", 0, isSortNearest, 0, 0, majors, studentYears)
	countPipeline = append(countPipeline, bson.D{{Key: "$count", Value: "total"}})
	cursor, err := DB.ProgramCollection.Aggregate(ctx, countPipeline)
	if err != nil {
		return 0, err
	}
	defer cursor.Close(ctx)
	var countResult []bson.M
	if err := cursor.All(ctx, &countResult); err != nil {
		return 0, err
	}
	if len(countResult) > 0 {
		switch v := countResult[0]["total"].(type) {
		case int32:
			return int64(v), nil
		case int64:
			return v, nil
		}
	}
	return 0, nil
}

func populateEnrollmentCounts(ctx context.Context, programs []models.ProgramDto) {
	for i, program := range programs {
		for j, item := range program.ProgramItems {
			count, err := DB.EnrollmentCollection.CountDocuments(ctx, bson.M{
				"programItemId": item.ID,
			})
			if err == nil {
				programs[i].ProgramItems[j].EnrollmentCount = int(count)
			}
		}
	}
}

func cacheProgramsResult(key string, results []models.ProgramDto, total int64, totalPages int) {
	if DB.RedisClient == nil {
		return
	}
	cacheValue, _ := json.Marshal(programsCache{
		Data:       results,
		Total:      total,
		TotalPages: totalPages,
	})
	_ = DB.RedisClient.Set(DB.RedisCtx, key, cacheValue, 2*time.Minute).Err()
}

// containsString checks if a slice contains a specific string.
func containsString(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}
