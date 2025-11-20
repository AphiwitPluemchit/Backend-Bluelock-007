package services

import (
	DB "Backend-Bluelock-007/src/database"
	"log"
)

// Use the database name provided by the database package (loaded from MONGO_DATABASE)
func init() {
	if err := DB.ConnectMongoDB(); err != nil {
		log.Fatal("MongoDB connection error:", err)
	}

	// Use the database name loaded by DB.ConnectMongoDB()
	dbName := DB.DatabaseName
	if dbName == "" {
		dbName = "bluelock"
	}

	// Ensure all collections exist (Mongo will also auto-create on first insert, but this makes it explicit)
	if err := DB.EnsureCollections(dbName, []string{
		"Programs",
		"Program_Items",
		"Admins",
		"Enrollments",
		"Foods",
		"Qr_Tokens",
		"Qr_Claims",
		"Forms",
		"Questions",
		"Submissions",
		"Students",
		"Users",
		"Courses",
		"Upload_Certificates",
		"Hour_Change_Histories",
	}); err != nil {
		log.Fatal("Failed ensuring collections:", err)
	}

	DB.ProgramCollection = DB.GetDefaultCollection("Programs")
	DB.ProgramItemCollection = DB.GetDefaultCollection("Program_Items")
	DB.AdminCollection = DB.GetDefaultCollection("Admins")
	DB.EnrollmentCollection = DB.GetDefaultCollection("Enrollments")
	DB.FoodCollection = DB.GetDefaultCollection("Foods")
	DB.QrTokenCollection = DB.GetDefaultCollection("Qr_Tokens")
	DB.QrClaimCollection = DB.GetDefaultCollection("Qr_Claims")
	DB.FormCollection = DB.GetDefaultCollection("Forms")
	DB.SubmissionCollection = DB.GetDefaultCollection("Submissions")
	DB.StudentCollection = DB.GetDefaultCollection("Students")
	DB.UserCollection = DB.GetDefaultCollection("Users")
	DB.CourseCollection = DB.GetDefaultCollection("Courses")
	DB.UploadCertificateCollection = DB.GetDefaultCollection("Upload_Certificates")
	DB.HourChangeHistoryCollection = DB.GetDefaultCollection("Hour_Change_Histories")

	// Note: Asynq initialization is now handled in main.go after Redis connection check

}
