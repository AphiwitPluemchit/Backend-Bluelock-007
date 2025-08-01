package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var (
	client     *mongo.Client
	once       sync.Once // ✅ ป้องกันการรัน ConnectMongoDB() ซ้ำ
	connectErr error

	ActivityCollection     *mongo.Collection // Renamed: exported
	ActivityItemCollection *mongo.Collection // Renamed: exported
	EnrollmentCollection   *mongo.Collection
	StudentCollection      *mongo.Collection // Renamed: exported
	CourseCollection       *mongo.Collection // ✅ เพิ่มตัวแปรนี้
	FormCollection         *mongo.Collection
	QuestionCollection     *mongo.Collection
	SubmissionCollection   *mongo.Collection
	AdminCollection        *mongo.Collection
	CheckinCollection      *mongo.Collection
	FoodCollection         *mongo.Collection
	QrTokenCollection      *mongo.Collection
	QrClaimCollection      *mongo.Collection
	UserCollection         *mongo.Collection
)

// ConnectMongoDB เชื่อมต่อกับ MongoDB แค่ครั้งเดียว
func ConnectMongoDB() error {

	// โหลดค่า Environment Variables จากไฟล์ .env
	err := godotenv.Load()
	if err != nil {
		log.Println("⚠️ Warning: No .env file found")
	}

	// ดึงค่าจาก Environment Variable
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		log.Fatal("❌ MONGO_URI environment variable not set. Please create a .env file and set it.")
	}

	once.Do(func() { // ✅ Run only once
		clientOptions := options.Client().ApplyURI(mongoURI)

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
