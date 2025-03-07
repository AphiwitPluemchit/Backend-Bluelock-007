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

var foodCollection *mongo.Collection

func init() {
	// เชื่อมต่อกับ MongoDB
	if err := database.ConnectMongoDB(); err != nil {
		log.Fatal("MongoDB connection error:", err)
	}

	foodCollection = database.GetCollection("BluelockDB", "foods")
	if foodCollection == nil {
		log.Fatal("Failed to get the foods collection")
	}
}

// CreateFoods - ลบข้อมูลเก่าทั้งหมด และเพิ่มข้อมูลใหม่
func CreateFoods(foods []models.Food) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 🔥 ลบข้อมูลเก่าทั้งหมด
	_, err := foodCollection.DeleteMany(ctx, bson.M{})
	if err != nil {
		return err
	}

	// เตรียมข้อมูลใหม่
	var foodDocs []interface{}
	for i := range foods {
		foods[i].ID = primitive.NewObjectID() // สร้าง ObjectID ใหม่
		foodDocs = append(foodDocs, foods[i])
	}

	// ✅ เพิ่มข้อมูลใหม่ทั้งหมด
	_, err = foodCollection.InsertMany(ctx, foodDocs)
	return err
}

// GetAllFoods - ดึงข้อมูลอาหารทั้งหมดเป็น Array
func GetAllFoods() ([]models.Food, error) {
	var foods []models.Food
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := foodCollection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var food models.Food
		if err := cursor.Decode(&food); err != nil {
			return nil, err
		}
		foods = append(foods, food)
	}

	return foods, nil
}

// GetFoodByID - ดึงข้อมูลผู้ใช้ตาม ID
func GetFoodByID(id string) (*models.Food, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid food ID")
	}

	var food models.Food
	err = foodCollection.FindOne(context.Background(), bson.M{"_id": objID}).Decode(&food)
	if err != nil {
		return nil, err
	}

	return &food, nil
}

// UpdateFood - อัปเดตข้อมูลผู้ใช้
func UpdateFood(id string, food *models.Food) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid food ID")
	}

	filter := bson.M{"_id": objID}
	update := bson.M{"$set": food}

	_, err = foodCollection.UpdateOne(context.Background(), filter, update)
	return err
}

// DeleteFood - ลบข้อมูลผู้ใช้
func DeleteFood(id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid food ID")
	}

	_, err = foodCollection.DeleteOne(context.Background(), bson.M{"_id": objID})
	return err
}
