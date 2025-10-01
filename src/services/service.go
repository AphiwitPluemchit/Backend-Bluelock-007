package services

import (
	DB "Backend-Bluelock-007/src/database"
	"log"
)

var collection = "V2"

// var collection = "BluelockDB" // หรือ "UAT"
func init() {
	if err := DB.ConnectMongoDB(); err != nil {
		log.Fatal("MongoDB connection error:", err)
	}

	// Ensure all collections exist (Mongo will also auto-create on first insert, but this makes it explicit)
	if err := DB.EnsureCollections(collection, []string{
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
		"View_Summary_Check_In_Out_Reports",
	}); err != nil {
		log.Fatal("Failed ensuring collections:", err)
	}

	DB.ProgramCollection = DB.GetCollection(collection, "Programs")
	DB.ProgramItemCollection = DB.GetCollection(collection, "Program_Items")
	DB.AdminCollection = DB.GetCollection(collection, "Admins")
	DB.EnrollmentCollection = DB.GetCollection(collection, "Enrollments")
	DB.FoodCollection = DB.GetCollection(collection, "Foods")
	DB.QrTokenCollection = DB.GetCollection(collection, "Qr_Tokens")
	DB.QrClaimCollection = DB.GetCollection(collection, "Qr_Claims")
	DB.FormCollection = DB.GetCollection(collection, "Forms")
	DB.SubmissionCollection = DB.GetCollection(collection, "Submissions")
	DB.StudentCollection = DB.GetCollection(collection, "Students")
	DB.UserCollection = DB.GetCollection(collection, "Users")
	DB.CourseCollection = DB.GetCollection(collection, "Courses")
	DB.UploadCertificateCollection = DB.GetCollection(collection, "Upload_Certificates")
	DB.HourChangeHistoryCollection = DB.GetCollection(collection, "Hour_Change_Histories")
	DB.SummaryCheckInOutReportsCollection = DB.GetCollection(collection, "View_Summary_Check_In_Out_Reports")

	if DB.RedisURI != "" {
		DB.InitAsynq()
	}

}
