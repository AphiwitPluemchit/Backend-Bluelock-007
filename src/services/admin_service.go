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

// GetAllAdmins - ดึงข้อมูลผู้ใช้ทั้งหมด
func GetAllAdmins() ([]models.Admin, error) {
	var admins []models.Admin
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := adminCollection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var admin models.Admin
		if err := cursor.Decode(&admin); err != nil {
			return nil, err
		}
		admins = append(admins, admin)
	}

	return admins, nil
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
