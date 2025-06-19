package services

import (
	DB "Backend-Bluelock-007/src/database"
	"log"
)

func init() {
	if err := DB.ConnectMongoDB(); err != nil {
		log.Fatal("MongoDB connection error:", err)
	}

	DB.ActivityCollection = DB.GetCollection("BluelockDB", "activitys")
	DB.ActivityItemCollection = DB.GetCollection("BluelockDB", "activityItems")
	DB.EnrollmentCollection = DB.GetCollection("BluelockDB", "enrollments")
	DB.StudentCollection = DB.GetCollection("BluelockDB", "students")
	if DB.ActivityCollection == nil || DB.ActivityItemCollection == nil {
		log.Fatal("Failed to get the required collections")
	}

	if DB.RedisURI != "" {
		DB.InitAsynq()
	}

}
