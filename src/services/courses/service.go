package courses

import (
	DB "Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

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

// GetAllCourses - ดึงข้อมูลคอร์สทั้งหมดแบบแบ่งหน้าและกรองข้อมูล
func GetAllCourses(params models.PaginationParams, filters models.CourseFilters) ([]models.Course, int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := bson.M{}

	// ค้นหา
	if params.Search != "" {
		query["$or"] = []bson.M{
			{"name": bson.M{"$regex": primitive.Regex{Pattern: params.Search, Options: "i"}}},
			{"description": bson.M{"$regex": primitive.Regex{Pattern: params.Search, Options: "i"}}},
		}
	}

	// ฟิลเตอร์
	if filters.Type != "" {
		query["type"] = filters.Type
	}
	if filters.IsHardSkill != nil {
		query["isHardSkill"] = *filters.IsHardSkill
	}
	if filters.IsActive != nil {
		query["isActive"] = *filters.IsActive
	}

	// นับจำนวนทั้งหมด
	total, err := DB.CourseCollection.CountDocuments(ctx, query)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count documents: %v", err)
	}

	// Pagination
	skip := int64((params.Page - 1) * params.Limit)
	order := 1
	if params.Order == "desc" {
		order = -1
	}

	// Pagination
	findOptions := options.Find().
		SetSkip(skip).
		SetLimit(int64(params.Limit)).
		SetSort(bson.M{params.SortBy: order})

	cursor, err := DB.CourseCollection.Find(ctx, query, findOptions)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find documents: %v", err)
	}
	defer cursor.Close(ctx)

	var courses []models.Course
	if err := cursor.All(ctx, &courses); err != nil {
		return nil, 0, fmt.Errorf("failed to decode documents: %v", err)
	}

	return courses, total, nil
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
		"name":               update.Name,
		"certificateName":    update.CertificateName,
		"certificateNameEng": update.CertificateNameEN,
		"link":               update.Link,
		"issuer":             update.Issuer,
		"type":               update.Type,
		"hour":               update.Hour,
		"isHardSkill":        update.IsHardSkill,
		"isActive":           update.IsActive,
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
