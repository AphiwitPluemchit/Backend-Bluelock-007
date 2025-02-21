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

var formEvaluationCollection *mongo.Collection

func init() {
	// เชื่อมต่อกับ MongoDB
	if err := database.ConnectMongoDB(); err != nil {
		log.Fatal("MongoDB connection error:", err)
	}

	formEvaluationCollection = database.GetCollection("BluelockDB", "formEvaluations")
	if formEvaluationCollection == nil {
		log.Fatal("Failed to get the formEvaluations collection")
	}
}

// CreateFormEvaluation - เพิ่มข้อมูลผู้ใช้ใน MongoDB
func CreateFormEvaluation(formEvaluation *models.FormEvaluation) error {
	formEvaluation.ID = primitive.NewObjectID() // กำหนด ID อัตโนมัติ
	_, err := formEvaluationCollection.InsertOne(context.Background(), formEvaluation)
	return err
}

// GetAllFormEvaluations - ดึงข้อมูลผู้ใช้ทั้งหมด
func GetAllFormEvaluations() ([]models.FormEvaluation, error) {
	var formEvaluations []models.FormEvaluation
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := formEvaluationCollection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var formEvaluation models.FormEvaluation
		if err := cursor.Decode(&formEvaluation); err != nil {
			return nil, err
		}
		formEvaluations = append(formEvaluations, formEvaluation)
	}

	return formEvaluations, nil
}

// GetFormEvaluationByID - ดึงข้อมูลผู้ใช้ตาม ID
func GetFormEvaluationByID(id string) (*models.FormEvaluation, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid formEvaluation ID")
	}

	var formEvaluation models.FormEvaluation
	err = formEvaluationCollection.FindOne(context.Background(), bson.M{"_id": objID}).Decode(&formEvaluation)
	if err != nil {
		return nil, err
	}

	return &formEvaluation, nil
}

// UpdateFormEvaluation - อัปเดตข้อมูลผู้ใช้
func UpdateFormEvaluation(id string, formEvaluation *models.FormEvaluation) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid formEvaluation ID")
	}

	filter := bson.M{"_id": objID}
	update := bson.M{"$set": formEvaluation}

	_, err = formEvaluationCollection.UpdateOne(context.Background(), filter, update)
	return err
}

// DeleteFormEvaluation - ลบข้อมูลผู้ใช้
func DeleteFormEvaluation(id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid formEvaluation ID")
	}

	_, err = formEvaluationCollection.DeleteOne(context.Background(), bson.M{"_id": objID})
	return err
}
