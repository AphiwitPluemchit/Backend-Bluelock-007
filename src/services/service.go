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

	// Ensure all collections exist (Mongo will also auto-create on first insert, but this makes it explicit)
	if err := DB.EnsureCollections(collection, []string{
		"activitys",
		"activityItems",
		"admins",
		"checkInOuts",
		"enrollments",
		"foods",
		"qrTokens",
		"qrClaims",
		"forms",
		"questions",
		"submissions",
		"students",
		"users",
		"courses",
		"uploadCertificates",
	}); err != nil {
		log.Fatal("Failed ensuring collections:", err)
	}

	DB.ActivityCollection = DB.GetCollection(collection, "activitys")
	DB.ActivityItemCollection = DB.GetCollection(collection, "activityItems")
	DB.AdminCollection = DB.GetCollection(collection, "admins")
	DB.CheckinCollection = DB.GetCollection(collection, "checkInOuts")
	DB.EnrollmentCollection = DB.GetCollection(collection, "enrollments")
	DB.FoodCollection = DB.GetCollection(collection, "foods")
	DB.QrTokenCollection = DB.GetCollection(collection, "qrTokens")
	DB.QrClaimCollection = DB.GetCollection(collection, "qrClaims")
	DB.FormCollection = DB.GetCollection(collection, "forms")
	DB.QuestionCollection = DB.GetCollection(collection, "questions")
	DB.SubmissionCollection = DB.GetCollection(collection, "submissions")
	DB.StudentCollection = DB.GetCollection(collection, "students")
	DB.UserCollection = DB.GetCollection(collection, "users")
	DB.CourseCollection = DB.GetCollection(collection, "courses")
	DB.UploadCertificateCollection = DB.GetCollection(collection, "uploadCertificates")

	if DB.RedisURI != "" {
		DB.InitAsynq()
	}

}
