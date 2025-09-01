package services

import (
	DB "Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/src/services/courses"
	"Backend-Bluelock-007/src/services/students"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func CreateUploadCertificate(uploadCertificate *models.UploadCertificate) (*mongo.InsertOneResult, error) {
	ctx := context.Background()
	return DB.UploadCertificateCollection.InsertOne(ctx, uploadCertificate)
}

func UpdateUploadCertificate(id string, uploadCertificate *models.UploadCertificate) (*mongo.UpdateResult, error) {
	ctx := context.Background()
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid upload certificate ID")
	}
	return DB.UploadCertificateCollection.UpdateOne(ctx, bson.M{"_id": objID}, bson.M{"$set": uploadCertificate})
}

func IsVerifiedDuplicate(ctx context.Context, url string) (bool, error) {
	if url == "" {
		return false, nil
	}

	filter := bson.M{
		"url":           url,
		"isNameMatch":   true,
		"isCourseMatch": true,
	}

	err := DB.UploadCertificateCollection.FindOne(ctx, filter).Err()
	switch err {
	case nil:
		// พบแล้ว → เป็นซ้ำ
		return true, nil
	case mongo.ErrNoDocuments:
		// ไม่พบ → ไม่ซ้ำ
		return false, nil
	default:
		// error อื่นจาก DB
		return false, err
	}
}

func VerifyURL(publicPageURL string, studentId string, courseId string) (bool, error) {

	student, course, err := CheckStudentCourse(studentId, courseId)
	if err != nil {
		return false, err
	}

	fmt.Println("student name: ", student.Name)
	fmt.Println("course name: ", course.Name)
	fmt.Println("course type: ", course.Type)

	if course.Type == "buumooc" {
		return BuuMooc(publicPageURL, student, course)
	}

	// if course.Type == "thaimooc" {
	// 	return ThaiMooc(publicPageURL)
	// }

	return false, errors.New("invalid course type")

}

func ThaiMooc(publicPageURL string) (bool, error) {
	ctx := context.Background()
	// สร้าง browser context (headless)
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true), // ถ้ารันใน container เป็น root ให้เปิดอันนี้
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	)
	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx, opts...)
	defer allocCancel()

	// สร้างแท็บใหม่
	taskCtx, taskCancel := chromedp.NewContext(allocCtx)
	defer taskCancel()

	var pdfSrc string
	err := chromedp.Run(taskCtx,
		chromedp.Navigate(publicPageURL),
		// รอจน network เงียบลงหน่อย
		chromedp.Sleep(500*time.Millisecond),
		// รอให้ <embed type="application/pdf"> โผล่ใน DOM
		chromedp.WaitVisible(`embed[type="application/pdf"]`, chromedp.ByQuery),
		// ดึงค่า attribute src
		chromedp.AttributeValue(`embed[type="application/pdf"]`, "src", &pdfSrc, nil, chromedp.ByQuery),
	)
	if err != nil {
		return false, err
	}
	if pdfSrc == "" {
		return false, errors.New("pdf <embed> not found or empty src")
	}
	// ตัดพารามิเตอร์ viewer ออก (#toolbar/navpanes/scrollbar)
	if i := strings.IndexByte(pdfSrc, '#'); i >= 0 {
		pdfSrc = pdfSrc[:i]
	}

	fmt.Println("pdfSrc: ", pdfSrc)

	filePath, err := DownloadPDF(pdfSrc)
	if err != nil {
		return false, err
	}

	fmt.Println("Downloaded PDF to:", filePath)

	return true, nil
}

func BuuMooc(publicPageURL string, student models.Student, course models.Course) (bool, error) {
	studentNameTh := student.Name
	studentNameEng := student.EngName

	// get html from publicPageURL and log
	resp, err := http.Get(publicPageURL)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}
	// fmt.Println(string(body))

	// remove "นาย" "นาง" "นางสาว" and "Miss" "Mr."
	studentNameTh = strings.ReplaceAll(studentNameTh, "นาย", "")
	studentNameTh = strings.ReplaceAll(studentNameTh, "นางสาว", "")
	studentNameTh = strings.ReplaceAll(studentNameTh, "นาง", "")
	studentNameEng = strings.ReplaceAll(studentNameEng, "Miss", "")
	studentNameEng = strings.ReplaceAll(studentNameEng, "Mr.", "")

	// match student name th  and eng or both
	if !strings.Contains(string(body), studentNameTh) && !strings.Contains(string(body), studentNameEng) {
		return false, errors.New("student name : " + studentNameTh + " or " + studentNameEng + " not found")
	}

	// match course name
	if !strings.Contains(string(body), course.Name) {
		return false, errors.New("course name : " + course.Name + " not found")
	}

	return true, nil
}

func DownloadPDF(pdfSrc string) (bool, error) {
	// Create a new HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create a new request
	req, err := http.NewRequest("GET", pdfSrc, nil)
	if err != nil {
		return false, fmt.Errorf("error creating request: %v", err)
	}

	// Set headers to mimic a browser request
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("error downloading PDF: %v", err)
	}
	defer resp.Body.Close()

	// Check if the response status code is OK
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Create the uploads directory if it doesn't exist
	uploadDir := "uploads/certificates"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return false, fmt.Errorf("error creating upload directory: %v", err)
	}

	// Create a unique filename for the downloaded PDF
	fileName := fmt.Sprintf("%s/%d.pdf", uploadDir, time.Now().UnixNano())

	// Create the file
	file, err := os.Create(fileName)
	if err != nil {
		return false, fmt.Errorf("error creating file: %v", err)
	}
	defer file.Close()

	// Write the response body to the file
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return false, fmt.Errorf("error saving PDF: %v", err)
	}

	return true, nil
}

func CheckStudentCourse(studentId string, courseId string) (models.Student, models.Course, error) {

	studentObjectID, err := primitive.ObjectIDFromHex(studentId)
	if err != nil {
		return models.Student{}, models.Course{}, err
	}

	courseObjectID, err := primitive.ObjectIDFromHex(courseId)
	if err != nil {
		return models.Student{}, models.Course{}, err
	}

	// find student
	student, err := students.GetStudentById(studentObjectID)
	if err != nil {
		return models.Student{}, models.Course{}, err
	}

	// find course
	course, err := courses.GetCourseByID(courseObjectID)
	if err != nil {
		return models.Student{}, models.Course{}, err
	}

	return *student, *course, nil
}
