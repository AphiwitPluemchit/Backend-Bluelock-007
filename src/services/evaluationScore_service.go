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

var evaluationScoreCollection *mongo.Collection

func init() {
	// เชื่อมต่อกับ MongoDB
	if err := database.ConnectMongoDB(); err != nil {
		log.Fatal("MongoDB connection error:", err)
	}

	evaluationScoreCollection = database.GetCollection("BluelockDB", "evaluationScores")
	if evaluationScoreCollection == nil {
		log.Fatal("Failed to get the evaluationScores collection")
	}
}

// CreateEvaluationScore - เพิ่มข้อมูลผู้ใช้ใน MongoDB
func CreateEvaluationScore(evaluationScore *models.EvaluationScore) error {
	evaluationScore.ID = primitive.NewObjectID() // กำหนด ID อัตโนมัติ
	_, err := evaluationScoreCollection.InsertOne(context.Background(), evaluationScore)
	return err
}

// GetAllEvaluationScores - ดึงข้อมูลผู้ใช้ทั้งหมด
func GetAllEvaluationScores() ([]models.EvaluationScore, error) {
	var evaluationScores []models.EvaluationScore
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := evaluationScoreCollection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var evaluationScore models.EvaluationScore
		if err := cursor.Decode(&evaluationScore); err != nil {
			return nil, err
		}
		evaluationScores = append(evaluationScores, evaluationScore)
	}

	return evaluationScores, nil
}

// GetEvaluationScoreByID - ดึงข้อมูลผู้ใช้ตาม ID
func GetEvaluationScoreByID(id string) (*models.EvaluationScore, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid evaluationScore ID")
	}

	var evaluationScore models.EvaluationScore
	err = evaluationScoreCollection.FindOne(context.Background(), bson.M{"_id": objID}).Decode(&evaluationScore)
	if err != nil {
		return nil, err
	}

	return &evaluationScore, nil
}

// UpdateEvaluationScore - อัปเดตข้อมูลผู้ใช้
func UpdateEvaluationScore(id string, evaluationScore *models.EvaluationScore) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid evaluationScore ID")
	}

	filter := bson.M{"_id": objID}
	update := bson.M{"$set": evaluationScore}

	_, err = evaluationScoreCollection.UpdateOne(context.Background(), filter, update)
	return err
}

// DeleteEvaluationScore - ลบข้อมูลผู้ใช้
func DeleteEvaluationScore(id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid evaluationScore ID")
	}

	_, err = evaluationScoreCollection.DeleteOne(context.Background(), bson.M{"_id": objID})
	return err
}
