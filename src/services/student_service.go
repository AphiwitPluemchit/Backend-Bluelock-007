package services

import (
	"Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
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
			yearPrefixes := generateStudentCodeFilter(intYears)
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

// GetStudentByCode - ดึงข้อมูลนักศึกษาด้วยรหัส code
func GetStudentByCode(code string) (*models.Student, error) {
	var student models.Student
	err := studentCollection.FindOne(context.Background(), bson.M{"code": code}).Decode(&student)
	if err != nil {
		return nil, err
	}

	return &student, nil
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

// ✅ สร้าง Student พร้อมเพิ่ม User (ใช้ ID เดียวกัน)
func CreateStudent(student *models.Student) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exists, err := isStudentExists(student.Code, student.Email)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("student already exists")
	}

	hashedPassword, err := hashPassword(student.Password)
	if err != nil {
		return errors.New("failed to hash password")
	}
	student.Password = hashedPassword
	student.ID = primitive.NewObjectID() // ใช้ ID เดียวกันกับ User

	_, err = studentCollection.InsertOne(ctx, student)
	if err != nil {
		return err
	}

	user := models.User{
		ID:        student.ID, // ใช้ ID เดียวกัน
		Email:     student.Email,
		Password:  student.Password,
		Role:      "Student",
		StudentID: &student.ID,
		AdminID:   nil,
	}
	userCollection := database.GetCollection("BluelockDB", "users")
	_, err = userCollection.InsertOne(ctx, user)
	if err != nil {
		studentCollection.DeleteOne(ctx, bson.M{"_id": student.ID})
		return err
	}

	return nil
}

// UpdateStudent - อัปเดตข้อมูลผู้ใช้
func UpdateStudent(id string, student *models.Student) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid student ID")
	}

	filter := bson.M{"_id": objID}
	update := bson.M{"$set": student}

	_, err = studentCollection.UpdateOne(context.Background(), filter, update)
	return err
}

// ✅ ลบ Student พร้อมลบ User ที่เกี่ยวข้อง
func DeleteStudent(id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid student ID")
	}

	userCollection := database.GetCollection("BluelockDB", "users")
	_, err = userCollection.DeleteOne(context.Background(), bson.M{"_id": objID})
	if err != nil {
		return err
	}

	_, err = studentCollection.DeleteOne(context.Background(), bson.M{"_id": objID})
	return err
}

// ✅ UpdateStatusToZero - เปลี่ยนสถานะนิสิตเป็น 0
func UpdateStatusToZero(studentID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// แปลง studentID จาก string เป็น ObjectID
	objectID, err := primitive.ObjectIDFromHex(studentID)
	if err != nil {
		return err
	}

	// ค้นหานิสิตตาม ID และอัพเดตสถานะเป็น 0
	filter := bson.M{"_id": objectID}
	update := bson.M{"$set": bson.M{"status": 0}}

	// Update นิสิทธ์ใน MongoDB
	_, err = studentCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	return nil
}
