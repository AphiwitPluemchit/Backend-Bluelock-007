package services

import (
	"context"

	"Backend-Bluelock-007/src/database" // สมมติว่าเชื่อมต่อ DB ไว้ที่นี่
	"Backend-Bluelock-007/src/models"

	"go.mongodb.org/mongo-driver/mongo"
)

// InsetForm แทรก form ลงฐานข้อมูล (สะกดผิด intentionally ตามคำขอ)
func InsetForm(ctx context.Context, form *models.Form) (*mongo.InsertOneResult, error) {
	return database.FormCollection.InsertOne(ctx, form)
}
