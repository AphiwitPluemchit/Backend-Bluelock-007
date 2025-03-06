package database

import (
	"context"
	"fmt"
	"log"
	"sync"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var (
	client     *mongo.Client
	once       sync.Once // ✅ ป้องกันการรัน ConnectMongoDB() ซ้ำ
	connectErr error
)

// ConnectMongoDB เชื่อมต่อกับ MongoDB แค่ครั้งเดียว
func ConnectMongoDB() error {
	once.Do(func() { // ✅ Run only once
		clientOptions := options.Client().ApplyURI("mongodb+srv://BluelockDB:BluelockDB@cluster0.m60i72z.mongodb.net/")

		client, connectErr = mongo.Connect(context.TODO(), clientOptions)
		if connectErr != nil {
			log.Fatal("❌ Failed to connect to MongoDB:", connectErr)
			return
		}

		// ตรวจสอบการเชื่อมต่อ
		connectErr = client.Ping(context.TODO(), readpref.Primary())
		if connectErr != nil {
			log.Fatal("❌ MongoDB ping failed:", connectErr)
			return
		}

		log.Println("✅ MongoDB connected successfully")
		ListDatabases()
	})

	return connectErr
}

// ListDatabases แสดงรายการ Database ทั้งหมด
func ListDatabases() {
	if client == nil {
		log.Fatal("❌ MongoDB client is nil")
	}

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
