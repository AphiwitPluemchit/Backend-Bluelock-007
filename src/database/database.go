package database

import (
	"context"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var client *mongo.Client

// ConnectMongoDB เชื่อมต่อกับ MongoDB และแสดงข้อมูลใน Database
func ConnectMongoDB() error {
	clientOptions := options.Client().ApplyURI("mongodb+srv://aphiwitrr:8bZ24ie8b7oTYoRk@cluster0.2sydc.mongodb.net/")

	var err error
	client, err = mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		return err
	}

	// ตรวจสอบการเชื่อมต่อ
	err = client.Ping(context.TODO(), readpref.Primary())
	if err != nil {
		return err
	}

	log.Println("✅ MongoDB connected successfully")

	// เรียกใช้ฟังก์ชันแสดงข้อมูล Database
	ListDatabases()

	return nil
}

// ListDatabases แสดงรายการ Database ทั้งหมด
func ListDatabases() {
	if client == nil {
		log.Fatal("❌ MongoDB client is nil")
	}

	// ดึงรายการ Database
	dbs, err := client.ListDatabaseNames(context.TODO(), bson.M{})
	if err != nil {
		log.Fatal("❌ Error listing databases:", err)
	}

	fmt.Println("📌 Databases in MongoDB:")
	for _, db := range dbs {
		fmt.Println(" -", db)
	}
}

// GetCollection รับ Collection จาก MongoDB
func GetCollection(dbName, collectionName string) *mongo.Collection {
	if client == nil {
		log.Fatal("❌ MongoDB client is nil")
	}
	return client.Database(dbName).Collection(collectionName)
}
