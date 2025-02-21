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

var foodVoteCollection *mongo.Collection

func init() {
	// เชื่อมต่อกับ MongoDB
	if err := database.ConnectMongoDB(); err != nil {
		log.Fatal("MongoDB connection error:", err)
	}

	foodVoteCollection = database.GetCollection("BluelockDB", "foodVotes")
	if foodVoteCollection == nil {
		log.Fatal("Failed to get the foodVotes collection")
	}
}

// CreateFoodVote - เพิ่มข้อมูลผู้ใช้ใน MongoDB
func CreateFoodVote(foodVote *models.FoodVote) error {
	foodVote.ID = primitive.NewObjectID() // กำหนด ID อัตโนมัติ
	_, err := foodVoteCollection.InsertOne(context.Background(), foodVote)
	return err
}

// GetAllFoodVotes - ดึงข้อมูลผู้ใช้ทั้งหมด
func GetAllFoodVotes() ([]models.FoodVote, error) {
	var foodVotes []models.FoodVote
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := foodVoteCollection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var foodVote models.FoodVote
		if err := cursor.Decode(&foodVote); err != nil {
			return nil, err
		}
		foodVotes = append(foodVotes, foodVote)
	}

	return foodVotes, nil
}

// GetFoodVoteByID - ดึงข้อมูลผู้ใช้ตาม ID
func GetFoodVoteByID(id string) (*models.FoodVote, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid foodVote ID")
	}

	var foodVote models.FoodVote
	err = foodVoteCollection.FindOne(context.Background(), bson.M{"_id": objID}).Decode(&foodVote)
	if err != nil {
		return nil, err
	}

	return &foodVote, nil
}

// UpdateFoodVote - อัปเดตข้อมูลผู้ใช้
func UpdateFoodVote(id string, foodVote *models.FoodVote) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid foodVote ID")
	}

	filter := bson.M{"_id": objID}
	update := bson.M{"$set": foodVote}

	_, err = foodVoteCollection.UpdateOne(context.Background(), filter, update)
	return err
}

// DeleteFoodVote - ลบข้อมูลผู้ใช้
func DeleteFoodVote(id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid foodVote ID")
	}

	_, err = foodVoteCollection.DeleteOne(context.Background(), bson.M{"_id": objID})
	return err
}
