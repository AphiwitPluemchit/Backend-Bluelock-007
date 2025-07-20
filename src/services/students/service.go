package students

import (
	"Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/src/services/activities"
	"context"
	"errors"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

var studentCollection *mongo.Collection

func init() {
	// เชื่อมต่อกับ MongoDB
	if err := database.ConnectMongoDB(); err != nil {
		log.Fatal("MongoDB connection error:", err)
	}

	studentCollection = database.GetCollection("BluelockDB", "students")
	if studentCollection == nil {
		log.Fatal("Failed to get the students collection")
	}
}

// GetStudentsWithFilter - ดึงข้อมูลนิสิตทั้งหมดที่ผ่านการ filter ตามเงื่อนไขที่ระบุ
func GetStudentsWithFilter(params models.PaginationParams, majors []string, studentYears []string, studentStatus []string) ([]bson.M, int64, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pipeline := mongo.Pipeline{}

	// 🔍 Step : Search filter (name, code)
	if params.Search != "" {
		regex := bson.M{"$regex": params.Search, "$options": "i"}
		pipeline = append(pipeline, bson.D{{Key: "$match", Value: bson.M{
			"$or": bson.A{
				bson.M{"name": regex},
				bson.M{"code": regex},
			},
		}}})
	}

	// 🔍 Step : Filter by major
	if len(majors) > 0 {
		pipeline = append(pipeline, bson.D{{Key: "$match", Value: bson.M{
			"major": bson.M{"$in": majors},
		}}})
	}

	// 🔍 Step : Filter by major
	if len(studentStatus) > 0 {
		intStatus := make([]int, 0)
		for _, s := range studentStatus {
			if v, err := strconv.Atoi(s); err == nil {
				intStatus = append(intStatus, v)
			}
		}
		if len(intStatus) > 0 {
			pipeline = append(pipeline, bson.D{{Key: "$match", Value: bson.M{
				"status": bson.M{"$in": intStatus},
			}}})
		}
	}
	// 🔍 Step : Filter by studentYears
	if len(studentYears) > 0 {
		// แปลง string เป็น int
		intYears := make([]int, 0)
		for _, y := range studentYears {
			if v, err := strconv.Atoi(y); err == nil {
				intYears = append(intYears, v)
			}
		}
		if len(intYears) > 0 {
			yearPrefixes := activities.GenerateStudentCodeFilter(intYears)
			var regexFilters []bson.M
			for _, prefix := range yearPrefixes {
				regexFilters = append(regexFilters, bson.M{"code": bson.M{"$regex": "^" + prefix}})
			}
			pipeline = append(pipeline, bson.D{{Key: "$match", Value: bson.M{
				"$or": regexFilters,
			}}})
		}
	}
	log.Println(pipeline)
	// 🔢 Count pipeline (before pagination)
	countPipeline := append(pipeline, bson.D{{Key: "$count", Value: "total"}})
	countCursor, err := studentCollection.Aggregate(ctx, countPipeline)
	if err != nil {
		return nil, 0, 0, err
	}
	var countResult struct {
		Total int64 `bson:"total"`
	}
	if countCursor.Next(ctx) {
		_ = countCursor.Decode(&countResult)
	}
	total := countResult.Total

	// 🔗 Lookup email จาก users collection
	pipeline = append(pipeline, bson.D{{Key: "$lookup", Value: bson.M{
		"from":         "users",
		"localField":   "_id",
		"foreignField": "refId",
		"as":           "user",
	}}})

	// 📌 Project เฉพาะฟิลด์ที่ต้องการ
	pipeline = append(pipeline, bson.D{{Key: "$project", Value: bson.M{
		"_id":       0,
		"id":        "$_id",
		"code":      1,
		"name":      1,
		"engName":   1,
		"status":    1,
		"softSkill": 1,
		"hardSkill": 1,
		"major":     1,
		"email":     bson.M{"$arrayElemAt": bson.A{"$user.email", 0}},
	}}})

	// 🔁 Sort, skip, limit
	sort := 1
	if strings.ToLower(params.Order) == "desc" {
		sort = -1
	}
	pipeline = append(pipeline,
		bson.D{{Key: "$sort", Value: bson.M{params.SortBy: sort}}},
		bson.D{{Key: "$skip", Value: (params.Page - 1) * params.Limit}},
		bson.D{{Key: "$limit", Value: params.Limit}},
	)

	// 🚀 Run main pipeline
	cursor, err := studentCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, 0, 0, err
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		return nil, 0, 0, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(params.Limit)))
	return results, total, totalPages, nil
}

// GetStudentByCode - ดึงข้อมูลนักศึกษาด้วยรหัส code พร้อม email
func GetStudentByCode(code string) (bson.M, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"code": code}}},
		{{Key: "$lookup", Value: bson.M{
			"from":         "users",
			"localField":   "_id",
			"foreignField": "refId",
			"as":           "user",
		}}},
		{{Key: "$project", Value: bson.M{
			"_id":       0,
			"id":        "$_id",
			"code":      1,
			"name":      1,
			"engName":   1,
			"status":    1,
			"softSkill": 1,
			"hardSkill": 1,
			"major":     1,
			"email":     bson.M{"$arrayElemAt": bson.A{"$user.email", 0}},
		}}},
	}

	cursor, err := studentCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, errors.New("student not found")
	}

	return results[0], nil
}

// ✅ ฟังก์ชันเข้ารหัส Password
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// ✅ ตรวจสอบว่ามี Student ที่ `code` หรือ `email` ซ้ำกันหรือไม่
func isStudentExists(code, email string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	count, err := studentCollection.CountDocuments(ctx, bson.M{
		"$or": []bson.M{
			{"code": code},
			{"email": email},
		},
	})

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// ✅ สร้าง Student พร้อมเพิ่ม User
func CreateStudent(userInput *models.User, studentInput *models.Student) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 🔍 ตรวจว่าซ้ำหรือไม่
	exists, err := isStudentExists(studentInput.Code, userInput.Email)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("student already exists")
	}

	// ✅ เข้ารหัสรหัสผ่าน
	hashedPassword, err := hashPassword(userInput.Password)
	if err != nil {
		return errors.New("failed to hash password")
	}
	userInput.Password = hashedPassword

	// ✅ สร้าง student ก่อน
	studentInput.ID = primitive.NewObjectID()
	_, err = studentCollection.InsertOne(ctx, studentInput)
	if err != nil {
		return err
	}

	// ✅ สร้าง user โดยใช้ refId ไปยัง student
	userInput.ID = primitive.NewObjectID()
	userInput.Role = "Student"
	userInput.RefID = studentInput.ID // 👈 จุดสำคัญ
	userInput.Email = strings.ToLower(strings.TrimSpace(userInput.Email))
	userInput.IsActive = true

	userCollection := database.GetCollection("BluelockDB", "users")
	_, err = userCollection.InsertOne(ctx, userInput)
	if err != nil {
		// rollback
		studentCollection.DeleteOne(ctx, bson.M{"_id": studentInput.ID})
		return err
	}

	return nil
}

// UpdateStudent - อัปเดตข้อมูล Student และ sync ไปยัง User
func UpdateStudent(id string, student *models.Student, email string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid student ID")
	}

	// ✅ อัปเดต student
	filter := bson.M{"_id": objID}
	update := bson.M{"$set": student}
	if _, err := studentCollection.UpdateOne(context.Background(), filter, update); err != nil {
		return err
	}

	// ✅ Sync ทั้ง name และ email ไปยัง user
	userCollection := database.GetCollection("BluelockDB", "users")
	_, err = userCollection.UpdateOne(context.Background(),
		bson.M{"refId": objID, "role": "student"},
		bson.M{"$set": bson.M{
			"name":  student.Name,
			"email": email, // ✅ เพิ่ม email
		}})
	return err
}

// DeleteStudent - ลบ Student พร้อมลบ User ที่อ้างถึง
func DeleteStudent(id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid student ID")
	}

	userCollection := database.GetCollection("BluelockDB", "users")

	// ลบ user ที่ refId เป็น student.id และ role เป็น "student"
	_, err = userCollection.DeleteOne(context.Background(), bson.M{
		"refId": objID,
		"role":  "student",
	})
	if err != nil {
		return err
	}

	// ลบ student
	_, err = studentCollection.DeleteOne(context.Background(), bson.M{"_id": objID})
	return err
}

// ✅ UpdateStatusToZero - เปลี่ยนสถานะ isActive ใน users เป็น false และ status = 0
func UpdateStatusToZero(studentID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// แปลง studentID จาก string เป็น ObjectID
	objectID, err := primitive.ObjectIDFromHex(studentID)
	if err != nil {
		return err
	}

	// ✅ อัปเดต status เป็น 0 ใน students collection
	filter := bson.M{"_id": objectID}
	update := bson.M{"$set": bson.M{"status": 0}}

	if _, err := studentCollection.UpdateOne(ctx, filter, update); err != nil {
		return err
	}

	// ✅ อัปเดต isActive เป็น false ใน users collection
	userCollection := database.GetCollection("BluelockDB", "users")
	_, err = userCollection.UpdateOne(ctx, bson.M{
		"refId": objectID,
		"role":  "Student",
	}, bson.M{
		"$set": bson.M{"isActive": false},
	})
	if err != nil {
		return err
	}
	return nil
}

func GetSammaryByCode(code string) (bson.M, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 🔍 ดึงข้อมูล student
	var student models.Student
	err := studentCollection.FindOne(ctx, bson.M{"code": code}).Decode(&student)
	if err != nil {
		return nil, errors.New("student not found")
	}

	// 🔄 Pipeline สำหรับหาประวัติ
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"studentId": student.ID}}},
		{{Key: "$lookup", Value: bson.M{
			"from":         "activityItems",
			"localField":   "activityItemId",
			"foreignField": "_id",
			"as":           "activityItem",
		}}},
		{{Key: "$unwind", Value: "$activityItem"}},
		{{Key: "$lookup", Value: bson.M{
			"from":         "activitys",
			"localField":   "activityItem.activityId",
			"foreignField": "_id",
			"as":           "activity",
		}}},
		{{Key: "$unwind", Value: "$activity"}},
		{{Key: "$project", Value: bson.M{
			"_id":              0,
			"registrationDate": "$registrationDate",
			"activity": bson.M{
				"id":            "$activity._id",
				"name":          "$activity.name",
				"type":          "$activity.type",
				"activityState": "$activity.activityState",
				"skill":         "$activity.skill",
				"activityItem": bson.M{
					"id":          "$activityItem._id",
					"name":        "$activityItem.name",
					"dates":       "$activityItem.dates",
					"hour":        "$activityItem.hour",
					"operator":    "$activityItem.operator",
					"description": "$activityItem.description",
				},
			},
		}}},
	}

	cursor, err := database.EnrollmentCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var history []bson.M
	if err := cursor.All(ctx, &history); err != nil {
		return nil, err
	}

	// ✅ return พร้อม history เต็ม
	return bson.M{
		"studentId": student.ID.Hex(),
		"code":      student.Code,
		"name":      student.Name,
		"major":     student.Major,
		"softSkill": student.SoftSkill,
		"hardSkill": student.HardSkill,
		"history":   history,
	}, nil
}

// Summary struct สำหรับ response
type SkillSummary struct {
	Completed    int `json:"completed"`
	NotCompleted int `json:"notCompleted"`
	Progress     int `json:"progress"` // %
}

type StudentSummary struct {
	Total          int          `json:"total"`
	Completed      int          `json:"completed"`
	NotCompleted   int          `json:"notCompleted"`
	CompletionRate int          `json:"completionRate"` // %
	SoftSkill      SkillSummary `json:"softSkill"`
	HardSkill      SkillSummary `json:"hardSkill"`
}

// GetStudentSummary - summary ตาม format ที่ต้องการ (เฉพาะนักเรียนที่มี status ไม่ใช่ 0)
func GetStudentSummary(majors []string, studentYears []string) (StudentSummary, error) {
	const softSkillTarget = 30
	const hordSkillTarget = 12

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 🔍 สร้าง filter สำหรับ query
	filter := bson.M{"status": bson.M{"$ne": 0}}

	// 🔍 Filter by major
	if len(majors) > 0 {
		filter["major"] = bson.M{"$in": majors}
	}

	// 🔍 Filter by studentYears
	if len(studentYears) > 0 {
		// แปลง string เป็น int
		intYears := make([]int, 0)
		for _, y := range studentYears {
			if v, err := strconv.Atoi(y); err == nil {
				intYears = append(intYears, v)
			}
		}
		if len(intYears) > 0 {
			yearPrefixes := activities.GenerateStudentCodeFilter(intYears)
			var regexFilters []bson.M
			for _, prefix := range yearPrefixes {
				regexFilters = append(regexFilters, bson.M{"code": bson.M{"$regex": "^" + prefix}})
			}
			filter["$or"] = regexFilters
		}
	}

	// 🔍 ดึงข้อมูลนักเรียนตาม filter
	cur, err := studentCollection.Find(ctx, filter)
	if err != nil {
		return StudentSummary{}, err
	}
	defer cur.Close(ctx)

	var students []models.Student
	if err := cur.All(ctx, &students); err != nil {
		return StudentSummary{}, err
	}

	total := len(students)
	completed := 0
	softCompleted := 0
	hardCompleted := 0

	for _, s := range students {
		if s.SoftSkill >= softSkillTarget {
			softCompleted++
		}
		if s.HardSkill >= hordSkillTarget {
			hardCompleted++
		}
		if s.SoftSkill >= softSkillTarget && s.HardSkill >= hordSkillTarget {
			completed++
		}
	}

	notCompleted := total - completed

	summary := StudentSummary{
		Total:          total,
		Completed:      completed,
		NotCompleted:   notCompleted,
		CompletionRate: percent(completed, total),
		SoftSkill: SkillSummary{
			Completed:    softCompleted,
			NotCompleted: total - softCompleted,
			Progress:     percent(softCompleted, total),
		},
		HardSkill: SkillSummary{
			Completed:    hardCompleted,
			NotCompleted: total - hardCompleted,
			Progress:     percent(hardCompleted, total),
		},
	}
	log.Printf("Student Summary (Status != 0): %+v", summary)
	return summary, nil
}

func percent(part, total int) int {
	if total == 0 {
		return 0
	}
	return int(float64(part) / float64(total) * 100)
}
