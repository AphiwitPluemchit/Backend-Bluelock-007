package services

import (
	"context"

	"Backend-Bluelock-007/src/database" // สมมติว่าเชื่อมต่อ DB ไว้ที่นี่
	"Backend-Bluelock-007/src/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// InsetForm แทรก form ลงฐานข้อมูล (สะกดผิด intentionally ตามคำขอ)
func InsetForm(ctx context.Context, form *models.Form) (*mongo.InsertOneResult, error) {
	return database.FormCollection.InsertOne(ctx, form)
}

func GetAllForms(ctx context.Context) ([]models.Form, error) {
	cursor, err := database.FormCollection.Find(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var forms []models.Form
	if err := cursor.All(ctx, &forms); err != nil {
		return nil, err
	}
	return forms, nil
}
