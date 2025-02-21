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

// CreateStudent - เพิ่มข้อมูลผู้ใช้ใน MongoDB
func CreateStudent(student *models.Student) error {
	student.ID = primitive.NewObjectID() // กำหนด ID อัตโนมัติ
	_, err := studentCollection.InsertOne(context.Background(), student)
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
