package services

import (
	"Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var orderCollection *mongo.Collection

// init ฟังก์ชันที่ใช้ในการเชื่อมต่อกับ MongoDB และกำหนดค่า collection
func init() {
	// เชื่อมต่อกับ MongoDB
	if err := database.ConnectMongoDB(); err != nil {
		log.Fatal("MongoDB connection error:", err)
	}

	// กำหนดค่า collection สำหรับคำสั่งซื้อ
	orderCollection = database.GetCollection("BluelockDB", "orders")
	if orderCollection == nil {
		log.Fatal("Failed to get the orders collection")
	}
}

// GetAllOrders ดึงข้อมูลคำสั่งซื้อทั้งหมด
func GetAllOrders() ([]models.Order, error) {
	var orders []models.Order
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := orderCollection.Find(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var order models.Order
		if err := cursor.Decode(&order); err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}
	return orders, nil
}

// GetOrderByID ดึงคำสั่งซื้อจาก MongoDB โดยใช้ ID
func GetOrderByID(id string) (*models.Order, error) {
	var order models.Order
	err := orderCollection.FindOne(context.Background(), bson.M{"id": id}).Decode(&order)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, err
		}
		return nil, err
	}
	return &order, nil
}

// CreateOrder สร้างคำสั่งซื้อใหม่ใน MongoDB
func CreateOrder(order *models.Order) error {
	_, err := orderCollection.InsertOne(context.Background(), order)
	return err
}
