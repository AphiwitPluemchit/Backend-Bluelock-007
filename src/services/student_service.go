package services

import (
	"Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"context"
	"errors"
	"log"
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

// ✅ CreateStudent - เพิ่ม Student ลงใน MongoDB
func CreateStudent(student *models.Student) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1️⃣ ตรวจสอบว่ามี Student ที่ `code` หรือ `email` ซ้ำหรือไม่
	exists, err := isStudentExists(student.Code, student.Email)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("student with the same code or email already exists")
	}

	// 2️⃣ เข้ารหัส Password ก่อนบันทึก
	hashedPassword, err := hashPassword(student.Password)
	if err != nil {
		return errors.New("failed to hash password")
	}
	student.Password = hashedPassword

	// 3️⃣ กำหนดค่า `ID` อัตโนมัติ
	student.ID = primitive.NewObjectID()

	// 4️⃣ บันทึกข้อมูลลง MongoDB
	_, err = studentCollection.InsertOne(ctx, student)
	return err
}

// GetAllStudents - ดึงข้อมูลผู้ใช้ทั้งหมด
func GetAllStudents() ([]models.Student, error) {
	var students []models.Student
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := studentCollection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var student models.Student
		if err := cursor.Decode(&student); err != nil {
			return nil, err
		}
		students = append(students, student)
	}

	return students, nil
}

// GetStudentByID - ดึงข้อมูลผู้ใช้ตาม ID
func GetStudentByID(id string) (*models.Student, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid student ID")
	}

	var student models.Student
	err = studentCollection.FindOne(context.Background(), bson.M{"_id": objID}).Decode(&student)
	if err != nil {
		return nil, err
	}

	return &student, nil
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

// DeleteStudent - ลบข้อมูลผู้ใช้
func DeleteStudent(id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid student ID")
	}

	_, err = studentCollection.DeleteOne(context.Background(), bson.M{"_id": objID})
	return err
}
