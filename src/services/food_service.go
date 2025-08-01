package services

import (
	DB "Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Collections are now initialized in service.go

// CreateFoods - เพิ่มข้อมูลอาหารทีละตัว
func CreateFood(food *models.Food) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := DB.FoodCollection.InsertOne(ctx, food)
	return err
}

// GetAllFoods - ดึงข้อมูลอาหารทั้งหมดเป็น Array
func GetAllFoods() ([]models.Food, error) {
	var foods []models.Food
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := DB.FoodCollection.Find(ctx, bson.M{})
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
	err = DB.FoodCollection.FindOne(context.Background(), bson.M{"_id": objID}).Decode(&food)
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

	_, err = DB.FoodCollection.UpdateOne(context.Background(), filter, update)
	return err
}

// DeleteFood - ลบข้อมูลผู้ใช้
func DeleteFood(id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid food ID")
	}

	_, err = DB.FoodCollection.DeleteOne(context.Background(), bson.M{"_id": objID})
	return err
}
