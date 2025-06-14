package activities

import (
	"Backend-Bluelock-007/src/database"
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

// Helper functions for activities service

// ===== Pipeline Helper Functions =====

// MaxEndTimeFromItem คำนวณเวลาสิ้นสุดที่มากที่สุดจาก ActivityItemDto
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

// ===== Asynq/Task Helper Functions =====

// DeleteTask ลบ task เดิมก่อน (ถ้ามี)
func DeleteTask(taskID string, activityID string, redisURI string) {
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
	activityID string,
	redisURI string,
) error {
	task, err := createFunc(activityID)
	if err != nil {
		log.Printf("❌ Failed to create task %s: %v", taskID, err)
		return err
	}
	DeleteTask(taskID, activityID, redisURI)
	_, err = AsynqClient.Enqueue(task, asynq.ProcessAt(runAt), asynq.TaskID(taskID))
	if err != nil {
		log.Printf("❌ Failed to enqueue task %s: %v", taskID, err)
		return err
	}
	log.Printf("✅ Task scheduled: %s | RunAt=%s", taskID, runAt.Format(time.RFC3339))
	return nil
}

// ScheduleChangeActivityStateJob สร้าง schedule งานเปลี่ยนสถานะ activity
func ScheduleChangeActivityStateJob(AsynqClient *asynq.Client, redisURI string, latestTime time.Time, endDateEnroll string, activityID string) error {
	if AsynqClient == nil {
		return errors.New("asynq client is not initialized")
	}
	deadline, err := time.ParseInLocation("2006-01-02", endDateEnroll, time.Local)
	if err != nil {
		return err
	}
	if !latestTime.IsZero() && latestTime.After(time.Now()) {
		if err := enqueueTask(
			AsynqClient,
			"complete-activity-"+activityID,
			jobs.NewcompleteActivityTask,
			latestTime,
			activityID,
			redisURI,
		); err != nil {
			return err
		}
	} else {
		log.Println("⏩ Skipped complete-activity task (invalid or past time)")
	}
	if !deadline.IsZero() && deadline.After(time.Now()) {
		if err := enqueueTask(
			AsynqClient,
			"close-enroll-"+activityID,
			jobs.NewCloseEnrollTask,
			deadline,
			activityID,
			redisURI,
		); err != nil {
			return err
		}
	} else {
		log.Println("⏩ Skipped close-enroll task (invalid or past time)")
	}
	return nil
}

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

// --- Helper functions for GetAllActivities ---

func buildActivitiesCacheKey(params models.PaginationParams, skills, states, majors []string, studentYears []int) string {
	return fmt.Sprintf(
		"activities:page=%d&limit=%d&search=%s&sortBy=%s&order=%s&skills=%v&states=%v&majors=%v&years=%v",
		params.Page, params.Limit, params.Search, params.SortBy, params.Order,
		skills, states, majors, studentYears,
	)
}

type activitiesCache struct {
	Data       []models.ActivityDto `json:"data"`
	Total      int64                `json:"total"`
	TotalPages int                  `json:"totalPages"`
}

func getActivitiesFromCache(key string) (*activitiesCache, error) {
	cached, err := database.RedisClient.Get(database.RedisCtx, key).Result()
	if err != nil {
		return nil, err
	}
	var cachedResult activitiesCache
	if err := json.Unmarshal([]byte(cached), &cachedResult); err != nil {
		return nil, err
	}
	return &cachedResult, nil
}

func buildActivitiesFilter(params models.PaginationParams, skills, states []string) bson.M {
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
	return filter
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

func aggregateActivities(ctx context.Context, pipeline mongo.Pipeline) ([]models.ActivityDto, error) {
	var results []models.ActivityDto
	cursor, err := activityCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}
	return results, nil
}

func countActivities(ctx context.Context, filter bson.M, majors []string, studentYears []int, limit int) (int64, error) {
	countPipeline := getLightweightActivitiesPipeline(filter, "", 0, 0, 0, majors, studentYears)
	countPipeline = append(countPipeline, bson.D{{Key: "$count", Value: "total"}})
	cursor, err := activityCollection.Aggregate(ctx, countPipeline)
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

func populateEnrollmentCounts(ctx context.Context, activities []models.ActivityDto) {
	for i, activity := range activities {
		for j, item := range activity.ActivityItems {
			count, err := enrollmentCollection.CountDocuments(ctx, bson.M{"activityItemId": item.ID})
			if err == nil {
				activities[i].ActivityItems[j].EnrollmentCount = int(count)
			}
		}
	}
}

func cacheActivitiesResult(key string, results []models.ActivityDto, total int64, totalPages int) {
	cacheValue, _ := json.Marshal(activitiesCache{
		Data:       results,
		Total:      total,
		TotalPages: totalPages,
	})
	_ = database.RedisClient.Set(database.RedisCtx, key, cacheValue, 2*time.Minute).Err()
}
