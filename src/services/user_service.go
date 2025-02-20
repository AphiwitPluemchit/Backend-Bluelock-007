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

var userCollection *mongo.Collection

func init() {
	// เชื่อมต่อกับ MongoDB
	if err := database.ConnectMongoDB(); err != nil {
		log.Fatal("MongoDB connection error:", err)
	}

	userCollection = database.GetCollection("BluelockDB", "users")
	if userCollection == nil {
		log.Fatal("Failed to get the users collection")
	}
}

// CreateUser - เพิ่มข้อมูลผู้ใช้ใน MongoDB
func CreateUser(user *models.User) error {
	user.ID = primitive.NewObjectID() // กำหนด ID อัตโนมัติ
	_, err := userCollection.InsertOne(context.Background(), user)
	return err
}

// GetAllUsers - ดึงข้อมูลผู้ใช้ทั้งหมด
func GetAllUsers() ([]models.User, error) {
	var users []models.User
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := userCollection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var user models.User
		if err := cursor.Decode(&user); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

// GetUserByID - ดึงข้อมูลผู้ใช้ตาม ID
func GetUserByID(id string) (*models.User, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	var user models.User
	err = userCollection.FindOne(context.Background(), bson.M{"_id": objID}).Decode(&user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// UpdateUser - อัปเดตข้อมูลผู้ใช้
func UpdateUser(id string, user *models.User) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid user ID")
	}

	filter := bson.M{"_id": objID}
	update := bson.M{"$set": user}

	_, err = userCollection.UpdateOne(context.Background(), filter, update)
	return err
}

// DeleteUser - ลบข้อมูลผู้ใช้
func DeleteUser(id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid user ID")
	}

	_, err = userCollection.DeleteOne(context.Background(), bson.M{"_id": objID})
	return err
}
