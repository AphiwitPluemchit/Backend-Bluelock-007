package services

import (
	DB "Backend-Bluelock-007/src/database"
	"log"
)

var collection = "BluelockDB"

// var collection = "BluelockDB" // หรือ "UAT"
func init() {
	if err := DB.ConnectMongoDB(); err != nil {
		log.Fatal("MongoDB connection error:", err)
	}

	DB.ActivityCollection = DB.GetCollection(collection, "activitys")
	DB.ActivityItemCollection = DB.GetCollection(collection, "activityItems")
	DB.AdminCollection = DB.GetCollection(collection, "admins")
	DB.CheckinCollection = DB.GetCollection(collection, "checkInOuts")
	DB.EnrollmentCollection = DB.GetCollection(collection, "enrollments")
	DB.FoodCollection = DB.GetCollection(collection, "foods")
	DB.QrClaimCollection = DB.GetCollection(collection, "qrClaims")
	DB.QrTokenCollection = DB.GetCollection(collection, "qrTokens")
	DB.FormCollection = DB.GetCollection(collection, "forms")
	DB.QuestionCollection = DB.GetCollection(collection, "questions")
	DB.SubmissionCollection = DB.GetCollection(collection, "submissions")
	DB.StudentCollection = DB.GetCollection(collection, "students")
	DB.UserCollection = DB.GetCollection(collection, "users")
	DB.CourseCollection = DB.GetCollection(collection, "courses")

	if DB.ActivityCollection == nil || DB.ActivityItemCollection == nil {
		log.Fatal("Failed to get the required collections")
	}

	if DB.RedisURI != "" {
		DB.InitAsynq()
	}

}
