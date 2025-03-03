package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var client *mongo.Client

// ConnectMongoDB เชื่อมต่อกับ MongoDB
func ConnectMongoDB() error {
	if client != nil {
		log.Println("✅ MongoDB already connected")
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().
		ApplyURI("mongodb+srv://BluelockDB:BluelockDB@cluster0.m60i72z.mongodb.net/").
		SetMaxPoolSize(50). // เพิ่มจำนวน connection pool
		SetConnectTimeout(5 * time.Second).
		SetServerSelectionTimeout(5 * time.Second)

	var err error
	client, err = mongo.Connect(ctx, clientOptions)
	if err != nil {
		return err
	}

	// ตรวจสอบการเชื่อมต่อ
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return err
	}

	log.Println("✅ MongoDB connected successfully")
	ListDatabases(ctx)
	return nil
}

// ListDatabases แสดงรายการ Database ทั้งหมด
func ListDatabases(ctx context.Context) {
	if client == nil {
		log.Fatal("❌ MongoDB client is nil")
	}

	dbs, err := client.ListDatabaseNames(ctx, bson.M{})
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
