package courses

import (
	DB "Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var ctx = context.Background()

// CreateCourse - สร้างคอร์สใหม่
func CreateCourse(course *models.Course) (*models.Course, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	course.ID = primitive.NewObjectID()
	_, err := DB.CourseCollection.InsertOne(ctx, course)
	if err != nil {
		return nil, err
	}
	return course, nil
}

// GetAllCourses - ดึงคอร์สทั้งหมด
func GetAllCourses() ([]models.Course, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cursor, err := DB.CourseCollection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	var courses []models.Course
	if err = cursor.All(ctx, &courses); err != nil {
		return nil, err
	}
	return courses, nil
}

// GetCourseByID - ดึงคอร์สตาม ID
func GetCourseByID(id primitive.ObjectID) (*models.Course, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var course models.Course
	err := DB.CourseCollection.FindOne(ctx, bson.M{"_id": id}).Decode(&course)
	if err != nil {
		return nil, err
	}
	return &course, nil
}

// UpdateCourse - อัปเดตคอร์ส
func UpdateCourse(id primitive.ObjectID, update models.Course) (*models.Course, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	updateData := bson.M{
		"name":        update.Name,
		"description": update.Description,
		"date":        update.Date,
		"issuer":      update.Issuer,
		"type":        update.Type,
		"hour":        update.Hour,
		"isHardSkill": update.IsHardSkill,
		"isActive":    update.IsActive,
	}
	_, err := DB.CourseCollection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": updateData})
	if err != nil {
		return nil, err
	}
	return GetCourseByID(id)
}

// DeleteCourse - ลบคอร์ส
func DeleteCourse(id primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := DB.CourseCollection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}
