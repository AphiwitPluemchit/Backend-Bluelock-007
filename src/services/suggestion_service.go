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

var suggestionCollection *mongo.Collection

func init() {
	// เชื่อมต่อกับ MongoDB
	if err := database.ConnectMongoDB(); err != nil {
		log.Fatal("MongoDB connection error:", err)
	}

	suggestionCollection = database.GetCollection("BluelockDB", "suggestions")
	if suggestionCollection == nil {
		log.Fatal("Failed to get the suggestions collection")
	}
}

// CreateSuggestion - เพิ่มข้อมูลผู้ใช้ใน MongoDB
func CreateSuggestion(suggestion *models.Suggestion) error {
	suggestion.ID = primitive.NewObjectID() // กำหนด ID อัตโนมัติ
	_, err := suggestionCollection.InsertOne(context.Background(), suggestion)
	return err
}

// GetAllSuggestions - ดึงข้อมูลผู้ใช้ทั้งหมด
func GetAllSuggestions() ([]models.Suggestion, error) {
	var suggestions []models.Suggestion
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := suggestionCollection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var suggestion models.Suggestion
		if err := cursor.Decode(&suggestion); err != nil {
			return nil, err
		}
		suggestions = append(suggestions, suggestion)
	}

	return suggestions, nil
}

// GetSuggestionByID - ดึงข้อมูลผู้ใช้ตาม ID
func GetSuggestionByID(id string) (*models.Suggestion, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid suggestion ID")
	}

	var suggestion models.Suggestion
	err = suggestionCollection.FindOne(context.Background(), bson.M{"_id": objID}).Decode(&suggestion)
	if err != nil {
		return nil, err
	}

	return &suggestion, nil
}

// UpdateSuggestion - อัปเดตข้อมูลผู้ใช้
func UpdateSuggestion(id string, suggestion *models.Suggestion) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid suggestion ID")
	}

	filter := bson.M{"_id": objID}
	update := bson.M{"$set": suggestion}

	_, err = suggestionCollection.UpdateOne(context.Background(), filter, update)
	return err
}

// DeleteSuggestion - ลบข้อมูลผู้ใช้
func DeleteSuggestion(id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid suggestion ID")
	}

	_, err = suggestionCollection.DeleteOne(context.Background(), bson.M{"_id": objID})
	return err
}
