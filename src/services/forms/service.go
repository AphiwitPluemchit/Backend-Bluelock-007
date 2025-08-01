package services

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"yourapp/models"
	"yourapp/database" // สมมติว่าเชื่อมต่อ DB ไว้ที่นี่
)

// ชื่อ collection ของฟอร์ม
const formCollectionName = "forms"

// InsertForm แทรก form ลงฐานข้อมูล
func InsertForm(ctx context.Context, form *models.Form) error {
	collection := database.MongoClient.Database("your_db_name").Collection(formCollectionName)

	// สามารถ custom options หรือ index ได้ตรงนี้
	opts := options.InsertOne()

	_, err := collection.InsertOne(ctx, form, opts)
	return err
}
