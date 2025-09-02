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

func CreateUploadCertificate(uploadCertificate *models.UploadCertificate) (*models.UploadCertificate, error) {
	ctx := context.Background()
	result, err := DB.UploadCertificateCollection.InsertOne(ctx, uploadCertificate)
	if err != nil {
		return nil, err
	}

	// Create a filter to find the inserted document
	filter := bson.M{"_id": result.InsertedID}

	// Find and return the inserted document
	var insertedDoc models.UploadCertificate
	err = DB.UploadCertificateCollection.FindOne(ctx, filter).Decode(&insertedDoc)
	if err != nil {
		return nil, err
	}

	return &insertedDoc, nil
}

func UpdateUploadCertificate(id string, uploadCertificate *models.UploadCertificate) (*mongo.UpdateResult, error) {
	ctx := context.Background()
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid upload certificate ID")
	}
	return DB.UploadCertificateCollection.UpdateOne(ctx, bson.M{"_id": objID}, bson.M{"$set": uploadCertificate})
}

func GetUploadCertificate(id string) (*models.UploadCertificate, error) {
	ctx := context.Background()
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid upload certificate ID")
	}
	var result models.UploadCertificate
	err = DB.UploadCertificateCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
func GetUploadCertificates(params models.UploadCertificateQuery, pagination models.PaginationParams) ([]models.UploadCertificate, models.PaginationMeta, error) {
	ctx := context.Background()

	// 1) Build base filter
	filter := bson.M{}
	if params.StudentID != "" {
		studentID, err := primitive.ObjectIDFromHex(params.StudentID)
		if err != nil {
			return nil, models.PaginationMeta{}, errors.New("invalid student ID format")
		}
		filter["studentId"] = studentID
	}
	if params.CourseID != "" {
		courseID, err := primitive.ObjectIDFromHex(params.CourseID)
		if err != nil {
			return nil, models.PaginationMeta{}, errors.New("invalid course ID format")
		}
		filter["courseId"] = courseID
	}
	if params.Status != "" {
		// เก็บใน DB เป็น string ("pending/approved/rejected") ใช่หรือไม่ ตรวจด้วย mongosh อีกที
		filter["status"] = params.Status
	}

	// 2) Clean pagination
	pagination = models.CleanPagination(pagination)

	// 3) Build pipeline
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: filter}},
	}

	// ควร join เฉพาะตอน "ต้องใช้" (ค้นหาด้วยชื่อ หรือ sort ด้วย studentName)
	needJoin := pagination.Search != "" || strings.EqualFold(pagination.SortBy, "studentname")
	if needJoin {
		pipeline = append(pipeline,
			bson.D{{Key: "$lookup", Value: bson.M{
				"from":         "students", // <== ตรวจชื่อคอลเลกชันให้ถูกต้อง!
				"localField":   "studentId",
				"foreignField": "_id",
				"as":           "studentData",
			}}},
			bson.D{{Key: "$unwind", Value: bson.M{
				"path":                       "$studentData",
				"preserveNullAndEmptyArrays": true, // สำคัญมาก กันเอกสารถูกทิ้งหมด
			}}},
			// ทำ field ชื่อให้แบนและมีค่า default เพื่อใช้ sort/search ง่าย
			bson.D{{Key: "$addFields", Value: bson.M{
				"studentName": bson.M{"$ifNull": []interface{}{"$studentData.name", ""}},
			}}},
		)

		if pagination.Search != "" {
			pipeline = append(pipeline,
				bson.D{{Key: "$match", Value: bson.M{
					"studentName": bson.M{
						"$regex": primitive.Regex{Pattern: pagination.Search, Options: "i"},
					},
				}}},
			)
		}
	}

	// 4) Sorting
	sortByField := pagination.SortBy
	if strings.EqualFold(pagination.SortBy, "studentname") {
		sortByField = "studentName"
	}
	sortOrder := 1
	if strings.ToLower(pagination.Order) == "desc" {
		sortOrder = -1
	}
	// ใส่ tie-breaker ด้วย _id กัน sort ไม่เสถียร
	pipeline = append(pipeline, bson.D{{Key: "$sort", Value: bson.D{
		{Key: sortByField, Value: sortOrder},
	}}})

	rows, meta, err := models.AggregatePaginateGlobal[models.UploadCertificate](
		ctx, DB.UploadCertificateCollection, pipeline, pagination.Page, pagination.Limit,
	)
	if err != nil {
		return nil, models.PaginationMeta{}, err
	}
	return rows, meta, nil
}

func VerifyURL(publicPageURL string, studentId string, courseId string) (bool, bool, error) {
	student, course, err := CheckStudentCourse(studentId, courseId)
	if err != nil {
		return false, false, err
	}

	isDuplicate, duplicateUpload, err := checkDuplicateURL(publicPageURL, student.ID, course.ID)
	if err != nil {
		return false, false, err
	}

	if isDuplicate {
		fmt.Println("Duplicate URL found", duplicateUpload)
		return false, true, nil
	}

	var res *FastAPIResp
	switch course.Type {
	case "buumooc":
		fmt.Println("Verify URL for BuuMooc")
		res, err = BuuMooc(publicPageURL, student, course)
	case "thaimooc":
		fmt.Println("Verify URL for ThaiMooc")
		res, err = ThaiMooc(publicPageURL, student, course)
	default:
		return false, false, fmt.Errorf("invalid course type: %s", course.Type)
	}
	if err != nil {
		fmt.Println("Error verifying URL", err)
		return false, false, err
	}
	if res == nil {
		fmt.Println("nil response from", course.Type)
		return false, false, fmt.Errorf("nil response from %s", course.Type)
	}

	uploadCertificate, err := saveUploadCertificate(publicPageURL, student.ID, course.ID, res)
	if err != nil {
		fmt.Println("Error saving upload certificate", err)
		return false, false, err
	}

	fmt.Println(uploadCertificate)

	return res.IsVerified, false, nil
}

func ThaiMooc(publicPageURL string, student models.Student, course models.Course) (*FastAPIResp, error) {
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
		return nil, err
	}
	if pdfSrc == "" {
		return nil, errors.New("pdf <embed> not found or empty src")
	}
	// ตัดพารามิเตอร์ viewer ออก (#toolbar/navpanes/scrollbar)
	if i := strings.IndexByte(pdfSrc, '#'); i >= 0 {
		pdfSrc = pdfSrc[:i]
	}

	filePath, err := DownloadPDF(pdfSrc)
	if err != nil {
		return nil, err
	}

	response, err := callThaiMoocFastAPI(
		FastAPIURL(),
		filePath,        // path ของ PDF ที่ดาวน์โหลดมา
		student.Name,    // student_th (ปรับตามฟิลด์จริง)
		student.EngName, // student_en
		course.Name,     // course_name (หรือ NameTH/NameEN ที่คุณต้องการ)
	)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func BuuMooc(publicPageURL string, student models.Student, course models.Course) (*FastAPIResp, error) {
	studentNameTh := student.Name
	studentNameEng := student.EngName

	// get html from publicPageURL and log
	resp, err := http.Get(publicPageURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// // remove "นาย" "นาง" "นางสาว" and "Miss" "Mr."
	// studentNameTh = strings.ReplaceAll(studentNameTh, "นาย", "")
	// studentNameTh = strings.ReplaceAll(studentNameTh, "นางสาว", "")
	// studentNameTh = strings.ReplaceAll(studentNameTh, "นาง", "")
	// studentNameEng = strings.ReplaceAll(studentNameEng, "Miss", "")
	// studentNameEng = strings.ReplaceAll(studentNameEng, "Mr.", "")

	// // match student name th  and eng or both
	// if !strings.Contains(string(body), studentNameTh) && !strings.Contains(string(body), studentNameEng) {
	// 	return nil, errors.New("This Certificate is not for student name : " + studentNameTh + " | " + studentNameEng)
	// }

	// // match course name
	// if !strings.Contains(string(body), course.Name) {
	// 	return nil, errors.New("This Certificate is not for course name : " + course.Name)
	// }

	response, err := callBUUMoocFastAPI(
		FastAPIURL(),
		string(body),   // html ที่ดึงมา
		studentNameTh,  // student_th
		studentNameEng, // student_en
		course.Name,    // course_name (ปรับตามฟิลด์จริง)
	)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func DownloadPDF(pdfSrc string) (string, error) {
	// Create a new HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create a new request
	req, err := http.NewRequest("GET", pdfSrc, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	// Set headers to mimic a browser request
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error downloading PDF: %v", err)
	}
	defer resp.Body.Close()

	// Check if the response status code is OK
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Create the uploads directory if it doesn't exist
	uploadDir := "uploads/certificates"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return "", fmt.Errorf("error creating upload directory: %v", err)
	}

	// Create a unique filename for the downloaded PDF
	fileName := fmt.Sprintf("%s/%d.pdf", uploadDir, time.Now().UnixNano())

	// Create the file
	file, err := os.Create(fileName)
	if err != nil {
		return "", fmt.Errorf("error creating file: %v", err)
	}
	defer file.Close()

	// Write the response body to the file
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", fmt.Errorf("error saving PDF: %v", err)
	}

	return fileName, nil
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

func FastAPIURL() string {
	if v := os.Getenv("FASTAPI_URL"); v != "" {
		return v
	}
	return "http://fastapi-ocr:8000"
}

func checkDuplicateURL(publicPageURL string, studentId primitive.ObjectID, courseId primitive.ObjectID) (bool, *models.UploadCertificate, error) {
	ctx := context.Background()

	var result models.UploadCertificate
	err := DB.UploadCertificateCollection.FindOne(ctx, bson.M{"url": publicPageURL, "status": models.StatusApproved}).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil, nil // URL is unique (no document found)
		}
		return false, nil, err // Some other error occurred
	}

	// copy result to new object remove _id
	newResult := models.UploadCertificate{}
	newResult.IsDuplicate = true
	newResult.StudentId = studentId
	newResult.CourseId = courseId
	newResult.UploadAt = time.Now()
	newResult.NameMatch = 0
	newResult.CourseMatch = 0
	newResult.Status = models.StatusRejected
	newResult.Remark = "Certificate URL already exists"
	newResult.Url = publicPageURL
	newResult.ID = primitive.NewObjectID()

	createDuplicate, err := CreateUploadCertificate(&newResult)
	if err != nil {
		return false, nil, err
	}

	return true, createDuplicate, nil // URL already exists
}

func saveUploadCertificate(publicPageURL string, studentId primitive.ObjectID, courseId primitive.ObjectID, res *FastAPIResp) (*models.UploadCertificate, error) {
	var uploadCertificate models.UploadCertificate

	nameMax := max(res.NameScoreTh, res.NameScoreEn)
	if nameMax >= 90 && res.CourseScore >= 90 {
		uploadCertificate.Status = models.StatusApproved
	} else if nameMax > 75 {
		if res.CourseScore > 75 {
			uploadCertificate.Status = models.StatusPending
		} else {
			uploadCertificate.Status = models.StatusRejected
		}
	} else {
		uploadCertificate.Status = models.StatusRejected
	}

	uploadCertificate.IsDuplicate = false
	uploadCertificate.Url = publicPageURL
	uploadCertificate.StudentId = studentId
	uploadCertificate.CourseId = courseId
	uploadCertificate.UploadAt = time.Now()
	uploadCertificate.NameMatch = nameMax
	uploadCertificate.CourseMatch = res.CourseScore

	saved, err := CreateUploadCertificate(&uploadCertificate)
	if err != nil {
		return nil, err
	}
	return saved, nil
}
