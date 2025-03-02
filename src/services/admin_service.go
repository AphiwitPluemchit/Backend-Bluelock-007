package services

import (
	"Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"context"
	"errors"
	"log"
	"math"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var adminCollection *mongo.Collection

func init() {
	// เชื่อมต่อกับ MongoDB
	if err := database.ConnectMongoDB(); err != nil {
		log.Fatal("MongoDB connection error:", err)
	}

	adminCollection = database.GetCollection("BluelockDB", "admins")
	if adminCollection == nil {
		log.Fatal("Failed to get the admins collection")
	}
}

// CreateAdmin - เพิ่มข้อมูลผู้ใช้ใน MongoDB
func CreateAdmin(admin *models.Admin) error {
	admin.ID = primitive.NewObjectID()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := adminCollection.InsertOne(ctx, admin)
	if err != nil {
		log.Println("❌ Error inserting admin:", err)
		return errors.New("failed to insert admin")
	}
	return nil
}

// GetAllAdmins - ดึงข้อมูล Admin พร้อม Pagination, Search, Sort
func GetAllAdmins(params models.PaginationParams) ([]models.Admin, int64, int, error) {
	var admins []models.Admin
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// คำนวณค่าการ Skip
	skip := int64((params.Page - 1) * params.Limit)

	// กำหนดค่าเริ่มต้นของการ Sort
	sortField := params.SortBy
	if sortField == "" {
		sortField = "id" // ค่าเริ่มต้น sort ด้วย ID
	}
	sortOrder := 1 // ค่าเริ่มต้นเป็น ascending (1)
	if strings.ToLower(params.Order) == "desc" {
		sortOrder = -1
	}

	// ค้นหาเฉพาะที่มีข้อความตรงกับ search
	filter := bson.M{}
	if params.Search != "" {
		filter["$or"] = []bson.M{
			{"name": bson.M{"$regex": params.Search, "$options": "i"}},
			{"email": bson.M{"$regex": params.Search, "$options": "i"}},
		}
	}

	// นับจำนวนทั้งหมดก่อน
	total, err := adminCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, 0, err
	}

	// Query MongoDB พร้อมตัวเลือก
	findOptions := options.Find().
		SetSkip(skip).
		SetLimit(int64(params.Limit)).
		SetSort(bson.D{{Key: sortField, Value: sortOrder}})

	cursor, err := adminCollection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, 0, 0, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var admin models.Admin
		if err := cursor.Decode(&admin); err != nil {
			log.Println("Error decoding admin:", err)
			continue
		}
		admins = append(admins, admin)
	}

	// คำนวณจำนวนหน้าทั้งหมด
	totalPages := int(math.Ceil(float64(total) / float64(params.Limit)))

	return admins, total, totalPages, nil
}

// GetAdminByID - ดึงข้อมูลผู้ใช้ตาม ID
func GetAdminByID(id string) (*models.Admin, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid admin ID")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var admin models.Admin
	err = adminCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&admin)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, errors.New("admin not found")
	} else if err != nil {
		log.Println("❌ Error finding admin:", err)
		return nil, err
	}

	return &admin, nil
}

// UpdateAdmin - อัปเดตข้อมูลผู้ใช้
func UpdateAdmin(id string, admin *models.Admin) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid admin ID")
	}

	filter := bson.M{"_id": objID}
	update := bson.M{"$set": admin}

	_, err = adminCollection.UpdateOne(context.Background(), filter, update)
	return err
}

// DeleteAdmin - ลบข้อมูลผู้ใช้
func DeleteAdmin(id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid admin ID")
	}

	_, err = adminCollection.DeleteOne(context.Background(), bson.M{"_id": objID})
	return err
}
