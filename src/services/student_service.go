package services

import (
	"Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"context"
	"errors"
	"log"
	"math"
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

// GetAllStudents - ดึงข้อมูลผู้ใช้ทั้งหมด
func GetStudentsWithFilter(params models.PaginationParams, majors []string, years []string) ([]bson.M, int64, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{}
	if params.Search != "" {
		filter["name"] = bson.M{"$regex": params.Search, "$options": "i"}
	}
	if len(years) > 0 {
		filter["status"] = bson.M{"$in": years}
	}

	sort := bson.D{{Key: params.SortBy, Value: 1}}
	if params.Order == "desc" {
		sort = bson.D{{Key: params.SortBy, Value: -1}}
	}

	skip := int64((params.Page - 1) * params.Limit)
	limit := int64(params.Limit)

	total, err := studentCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, 0, err
	}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: filter}},
		{{Key: "$lookup", Value: bson.M{
			"from":         "majors",
			"localField":   "majorId",
			"foreignField": "_id",
			"as":           "majorInfo",
		}}},
		{{Key: "$addFields", Value: bson.M{
			"majorNames": bson.M{
				"$map": bson.M{
					"input": "$majorInfo",
					"as":    "m",
					"in":    "$$m.majorName",
				},
			},
			"studentYears": bson.A{"$status"},
		}}},
		{{Key: "$project", Value: bson.M{
			"password":  0,
			"majorId":   0,
			"majorInfo": 0,
		}}},
	}

	if len(majors) > 0 {
		pipeline = append(pipeline, bson.D{{Key: "$match", Value: bson.M{
			"majorNames": bson.M{"$in": majors},
		}}})
	}

	pipeline = append(pipeline,
		bson.D{{Key: "$sort", Value: sort}},
		bson.D{{Key: "$skip", Value: skip}},
		bson.D{{Key: "$limit", Value: limit}},
	)

	cursor, err := studentCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, 0, 0, err
	}
	defer cursor.Close(ctx)

	var students []bson.M
	if err := cursor.All(ctx, &students); err != nil {
		return nil, 0, 0, err
	}
	totalPages := int(math.Ceil(float64(total) / float64(params.Limit)))
	return students, total, totalPages, nil
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

// ✅ สร้าง Student พร้อมเพิ่ม User
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
	student.ID = primitive.NewObjectID()

	_, err = studentCollection.InsertOne(ctx, student)
	if err != nil {
		return err
	}

	user := models.User{
		ID:        primitive.NewObjectID(),
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
	_, err = userCollection.DeleteOne(context.Background(), bson.M{"studentId": objID})
	if err != nil {
		return err
	}

	_, err = studentCollection.DeleteOne(context.Background(), bson.M{"_id": objID})
	return err
}
