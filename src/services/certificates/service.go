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

// UpdateUploadCertificateStatus อัพเดทสถานะของ certificate และจัดการชั่วโมงให้อัตโนมัติ
// ใช้โดย Admin เพื่อ approve/reject certificate
func UpdateUploadCertificateStatus(id string, newStatus models.StatusType, remark string) (*models.UploadCertificate, error) {
	ctx := context.Background()
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid upload certificate ID")
	}

	// 1. ดึงข้อมูล certificate เดิม
	var oldCert models.UploadCertificate
	err = DB.UploadCertificateCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&oldCert)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("upload certificate not found")
		}
		return nil, err
	}

	// 2. ตรวจสอบว่าสถานะเปลี่ยนจริงหรือไม่
	if oldCert.Status == newStatus {
		fmt.Printf("No status change for certificate %s (already %s)\n", id, newStatus)
		return &oldCert, nil // ไม่มีการเปลี่ยนแปลง
	}

	// Validation: ตรวจสอบว่าเป็น duplicate certificate หรือไม่
	if oldCert.IsDuplicate {
		fmt.Printf("Warning: Attempting to change status of duplicate certificate %s\n", id)
		// Allow status change but won't affect hours
	}

	// 3. ตรวจสอบ business rules และจัดการชั่วโมง
	fmt.Printf("📝 Status change detected: %s -> %s for certificate %s\n", oldCert.Status, newStatus, id)

	// สร้าง copy ของ oldCert เพื่อใช้ในการคำนวณชั่วโมง (เพราะจะใช้ข้อมูลเดิม)
	certForHours := oldCert

	// กรณีที่ 1: pending -> approved (Admin อนุมัติ)
	if oldCert.Status == models.StatusPending && newStatus == models.StatusApproved {
		fmt.Println("▶️ Adding hours for pending -> approved")
		if err := addCertificateHours(ctx, &certForHours); err != nil {
			return nil, fmt.Errorf("failed to add hours: %v", err)
		}
	}

	// กรณีที่ 2: approved -> rejected (Admin ปฏิเสธ certificate ที่เคยอนุมัติแล้ว)
	if oldCert.Status == models.StatusApproved && newStatus == models.StatusRejected {
		fmt.Println("▶️ Removing hours for approved -> rejected")
		if err := removeCertificateHours(ctx, &certForHours); err != nil {
			return nil, fmt.Errorf("failed to remove hours: %v", err)
		}
	}

	// กรณีที่ 3: rejected -> approved (Admin เปลี่ยนใจอนุมัติ)
	if oldCert.Status == models.StatusRejected && newStatus == models.StatusApproved {
		fmt.Println("▶️ Adding hours for rejected -> approved")
		if err := addCertificateHours(ctx, &certForHours); err != nil {
			return nil, fmt.Errorf("failed to add hours: %v", err)
		}
	}

	// กรณีที่ 4: approved -> pending (Admin ถอนการอนุมัติ ต้องรอพิจารณาใหม่)
	if oldCert.Status == models.StatusApproved && newStatus == models.StatusPending {
		fmt.Println("▶️ Removing hours for approved -> pending")
		if err := removeCertificateHours(ctx, &certForHours); err != nil {
			return nil, fmt.Errorf("failed to remove hours: %v", err)
		}
	}

	// กรณีที่ 5: pending -> rejected (Admin ปฏิเสธตั้งแต่แรก - ไม่ต้องลบชั่วโมงเพราะไม่เคยเพิ่ม)
	// แต่ยังต้องบันทึก history record
	if oldCert.Status == models.StatusPending && newStatus == models.StatusRejected {
		fmt.Println("▶️ Rejecting pending certificate (no hours to remove)")
		if err := recordCertificateRejection(ctx, &certForHours, remark); err != nil {
			fmt.Printf("Warning: Failed to record certificate rejection: %v\n", err)
		}
	}

	// กรณีที่ 6: rejected -> pending (Admin เปลี่ยนใจให้พิจารณาใหม่ - ไม่ต้องทำอะไร)
	// บันทึก history record ด้วยสถานะ pending
	if oldCert.Status == models.StatusRejected && newStatus == models.StatusPending {
		fmt.Println("▶️ Moving rejected certificate back to pending (no hours change)")
		if err := recordCertificatePending(ctx, &certForHours, remark); err != nil {
			fmt.Printf("Warning: Failed to record certificate pending status: %v\n", err)
		}
	}

	// 4. อัพเดทสถานะและข้อมูลอื่นๆ
	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"status":          newStatus,
			"remark":          remark,
			"changedStatusAt": now,
		},
	}

	_, err = DB.UploadCertificateCollection.UpdateOne(ctx, bson.M{"_id": objID}, update)
	if err != nil {
		return nil, fmt.Errorf("failed to update certificate status: %v", err)
	}

	// 5. ดึงข้อมูล certificate ที่อัพเดทแล้ว
	var updatedCert models.UploadCertificate
	err = DB.UploadCertificateCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&updatedCert)
	if err != nil {
		return nil, err
	}

	fmt.Printf("✅ Certificate %s status updated successfully: %s -> %s\n", id, oldCert.Status, newStatus)
	return &updatedCert, nil
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

	// Debug logs
	fmt.Println("GetUploadCertificates called with params:")
	fmt.Printf("  StudentID=%s CourseID=%s Status=%s\n", params.StudentID, params.CourseID, params.Status)

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
	fmt.Printf("  Pagination: page=%d limit=%d search=%s sortBy=%s order=%s\n", pagination.Page, pagination.Limit, pagination.Search, pagination.SortBy, pagination.Order)

	// 3) Build pipeline
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: filter}},
	}

	// print filter for debugging
	fmt.Println("  Mongo filter:", filter)

	pipeline = append(pipeline,
		bson.D{{Key: "$lookup", Value: bson.M{
			"from":         "Students", // ใช้ชื่อ collection ตามที่เชื่อมต่อใน DB
			"localField":   "studentId",
			"foreignField": "_id",
			"as":           "student",
		}}},
		bson.D{{Key: "$unwind", Value: bson.M{
			"path":                       "$student",
			"preserveNullAndEmptyArrays": true, // สำคัญมาก กันเอกสารถูกทิ้งหมด
		}}},
		// ทำ field ชื่อให้แบนและมีค่า default เพื่อใช้ sort/search ง่าย
		bson.D{{Key: "$addFields", Value: bson.M{
			"student":     bson.M{"$ifNull": []interface{}{"$student", bson.M{}}}, // เก็บ object student หรือ {} แทน null
			"studentName": bson.M{"$ifNull": []interface{}{"$student.name", ""}},
		}}},
	)

	// ควร join เฉพาะตอน "ต้องใช้" (ค้นหาด้วยชื่อ หรือ sort ด้วย studentName)
	needJoin := pagination.Search != "" || strings.EqualFold(pagination.SortBy, "studentname")
	fmt.Println("  needJoin for student lookup:", needJoin)
	if needJoin {
		if pagination.Search != "" {
			pipeline = append(pipeline,
				bson.D{{Key: "$match", Value: bson.M{
					"student.name": bson.M{
						"$regex": primitive.Regex{Pattern: pagination.Search, Options: "i"},
					},
				}}},
			)
		}
	}

	// 👉 join course (ปกติเรามักอยากโชว์เสมอ)
	pipeline = append(pipeline,
		bson.D{{Key: "$lookup", Value: bson.M{
			"from":         "Courses",  // ชื่อคอลเลกชันของคุณ (ตรงกับ DB)
			"localField":   "courseId", // อิงจาก UploadCertificate.CourseId
			"foreignField": "_id",
			"as":           "course",
		}}},
		bson.D{{Key: "$unwind", Value: bson.M{
			"path": "$course", "preserveNullAndEmptyArrays": true,
		}}},
		bson.D{{Key: "$addFields", Value: bson.M{
			"course": bson.M{"$ifNull": []interface{}{"$course", bson.M{}}}, // เก็บ object course หรือ {} แทน null
		}}},
	)

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

	fmt.Printf("  Sorting: field=%s order=%d\n", sortByField, sortOrder)

	// Debug: print pipeline (best-effort)
	pipelineBytes, _ := bson.MarshalExtJSON(pipeline, true, true)
	fmt.Println("  Aggregation pipeline:", string(pipelineBytes))

	rows, meta, err := models.AggregatePaginateGlobal[models.UploadCertificate](
		ctx, DB.UploadCertificateCollection, pipeline, pagination.Page, pagination.Limit,
	)
	if err != nil {
		return nil, models.PaginationMeta{}, err
	}

	// Debug: number of returned rows
	fmt.Printf("  Aggregation returned %d rows\n", len(rows))
	return rows, meta, nil
}

func VerifyURL(publicPageURL string, studentId string, courseId string) (bool, bool, error) {
	fmt.Println("VerifyURL ")
	fmt.Println("studentId", studentId)
	fmt.Println("courseId", courseId)
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
		course.NameEN,  // course_name_en
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
	fmt.Println("Check Student Course")

	// find student
	student, err := students.GetStudentById(studentObjectID)
	if err != nil {
		return models.Student{}, models.Course{}, err
	}
	fmt.Println("studentId", studentId)

	// find course
	course, err := courses.GetCourseByID(courseObjectID)
	if err != nil {
		return models.Student{}, models.Course{}, err
	}
	fmt.Println("courseId", courseId)

	return *student, *course, err
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
	filter := bson.M{"url": publicPageURL, "status": models.StatusApproved}
	fmt.Println("checkDuplicateURL filter:", filter)
	err := DB.UploadCertificateCollection.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil, nil // URL is unique (no document found)
		}
		return false, nil, err // Some other error occurred
	}
	fmt.Println("Found existing approved upload certificate:", result)

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

	fmt.Println("Creating duplicate upload certificate:", newResult)
	createDuplicate, err := CreateUploadCertificate(&newResult)
	if err != nil {
		return false, nil, err
	}

	fmt.Println("Created duplicate upload certificate:", createDuplicate)

	return true, createDuplicate, nil // URL already exists
}

func saveUploadCertificate(publicPageURL string, studentId primitive.ObjectID, courseId primitive.ObjectID, res *FastAPIResp) (*models.UploadCertificate, error) {
	var uploadCertificate models.UploadCertificate

	// Helper to dereference nullable scores; treat nil as 0
	getScore := func(p *int) int {
		if p == nil {
			return 0
		}
		return *p
	}

	nameScoreTh := getScore(res.NameScoreTh)
	nameScoreEn := getScore(res.NameScoreEn)
	courseScore := getScore(res.CourseScore)
	courseScoreEn := getScore(res.CourseScoreEn)

	nameMax := max(nameScoreTh, nameScoreEn)

	// Decide status using available course scores: prefer courseScore, fallback to courseScoreEn
	courseMax := max(courseScore, courseScoreEn)

	if nameMax >= 90 && courseMax >= 90 {
		uploadCertificate.Status = models.StatusApproved
	} else if nameMax > 75 {
		if courseMax > 75 {
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
	uploadCertificate.NameEngMatch = nameScoreEn
	uploadCertificate.CourseMatch = courseScore
	uploadCertificate.CourseEngMatch = courseScoreEn

	saved, err := CreateUploadCertificate(&uploadCertificate)
	if err != nil {
		return nil, err
	}

	// ถ้าสถานะเป็น approved ให้บันทึกชั่วโมงทันที (auto-approved)
	if saved.Status == models.StatusApproved {
		if err := addCertificateHours(context.Background(), saved); err != nil {
			fmt.Printf("Warning: Failed to add certificate hours for auto-approved certificate %s: %v\n", saved.ID.Hex(), err)
		}
	}

	return saved, nil
}

// addCertificateHours เพิ่มชั่วโมงให้กับนิสิตเมื่อ certificate ได้รับการอนุมัติ
func addCertificateHours(ctx context.Context, certificate *models.UploadCertificate) error {
	// Validation: ตรวจสอบว่า certificate ไม่ซ้ำ
	if certificate.IsDuplicate {
		fmt.Printf("Skipping hours addition for duplicate certificate %s\n", certificate.ID.Hex())
		return nil // ไม่ต้อง error แค่ไม่เพิ่มชั่วโมง
	}

	// 1. ดึงข้อมูล course เพื่อหาจำนวนชั่วโมงและประเภท skill
	course, err := courses.GetCourseByID(certificate.CourseId)
	if err != nil {
		return fmt.Errorf("course not found: %v", err)
	}

	if course.Hour <= 0 {
		fmt.Printf("Warning: Course %s has no hours defined (%d), skipping hours addition\n", course.ID.Hex(), course.Hour)
		return nil // ไม่ error แต่ไม่เพิ่มชั่วโมง
	}

	// Validation: ตรวจสอบว่า course active
	if !course.IsActive {
		return fmt.Errorf("cannot add hours for inactive course: %s", course.Name)
	}

	// 2. ดึงข้อมูล student
	student, err := students.GetStudentById(certificate.StudentId)
	if err != nil {
		return fmt.Errorf("student not found: %v", err)
	}

	// 3. กำหนด skill type
	skillType := "soft"
	if course.IsHardSkill {
		skillType = "hard"
	}

	// 4. เพิ่มชั่วโมงให้กับนิสิต
	var update bson.M
	switch skillType {
	case "soft":
		update = bson.M{
			"$inc": bson.M{
				"softSkill": course.Hour,
			},
		}
	case "hard":
		update = bson.M{
			"$inc": bson.M{
				"hardSkill": course.Hour,
			},
		}
	default:
		return fmt.Errorf("invalid skill type: %s", skillType)
	}

	_, err = DB.StudentCollection.UpdateOne(ctx, bson.M{"_id": certificate.StudentId}, update)
	if err != nil {
		return fmt.Errorf("failed to update student hours: %v", err)
	}

	// 5. บันทึกประวัติการเปลี่ยนแปลงชั่วโมง (status: approved)
	// สร้าง HourChangeHistory record โดยใช้ status "approved"
	hourChange := models.HourChangeHistory{
		ID:           primitive.NewObjectID(),
		StudentID:    certificate.StudentId,
		SkillType:    skillType,
		Status:       models.HCStatusApproved, // ใช้ "approved" เพราะ certificate ถูกอนุมัติ
		HourChange:   course.Hour,
		Remark:       "Certificate Approved",
		ChangeAt:     time.Now(),
		Title:        course.Name,
		SourceType:   "certificate",
		SourceID:     certificate.ID,
		EnrollmentID: nil, // certificate ไม่มี enrollmentID
	}

	_, err = DB.HourChangeHistoryCollection.InsertOne(ctx, hourChange)
	if err != nil {
		// Log warning but don't fail the operation
		fmt.Printf("Warning: Failed to save certificate history: %v\n", err)
	}

	fmt.Printf("✅ Added %d hours (%s skill) to student %s for certificate %s\n",
		course.Hour, skillType, student.Code, certificate.ID.Hex())

	return nil
}

// removeCertificateHours ลบชั่วโมงจากนิสิตเมื่อ certificate ถูกปฏิเสธหรือยกเลิก
func removeCertificateHours(ctx context.Context, certificate *models.UploadCertificate) error {
	// Validation: ตรวจสอบว่า certificate ไม่ซ้ำ
	if certificate.IsDuplicate {
		fmt.Printf("Skipping hours removal for duplicate certificate %s\n", certificate.ID.Hex())
		return nil // ไม่ต้อง error แค่ไม่ลบชั่วโมง
	}

	// 1. ดึงข้อมูล course
	course, err := courses.GetCourseByID(certificate.CourseId)
	if err != nil {
		return fmt.Errorf("course not found: %v", err)
	}

	if course.Hour <= 0 {
		fmt.Printf("Warning: Course %s has no hours defined (%d), skipping hours removal\n", course.ID.Hex(), course.Hour)
		return nil // ไม่ error แต่ไม่ลบชั่วโมง
	}

	// 2. ดึงข้อมูล student
	student, err := students.GetStudentById(certificate.StudentId)
	if err != nil {
		return fmt.Errorf("student not found: %v", err)
	}

	// 3. กำหนด skill type
	skillType := "soft"
	if course.IsHardSkill {
		skillType = "hard"
	}

	// 4. ลบชั่วโมงจากนิสิต (ไม่ให้ติดลบ)
	var update bson.M
	var hoursToRemove int

	switch skillType {
	case "soft":
		hoursToRemove = course.Hour
		if student.SoftSkill < course.Hour {
			hoursToRemove = student.SoftSkill
			fmt.Printf("Warning: Student %s has insufficient soft skill hours (%d < %d), removing only %d\n",
				student.Code, student.SoftSkill, course.Hour, hoursToRemove)
		}
		update = bson.M{
			"$inc": bson.M{
				"softSkill": -hoursToRemove,
			},
		}
	case "hard":
		hoursToRemove = course.Hour
		if student.HardSkill < course.Hour {
			hoursToRemove = student.HardSkill
			fmt.Printf("Warning: Student %s has insufficient hard skill hours (%d < %d), removing only %d\n",
				student.Code, student.HardSkill, course.Hour, hoursToRemove)
		}
		update = bson.M{
			"$inc": bson.M{
				"hardSkill": -hoursToRemove,
			},
		}
	default:
		return fmt.Errorf("invalid skill type: %s", skillType)
	}

	// Skip if no hours to remove
	if hoursToRemove <= 0 {
		fmt.Printf("No hours to remove for student %s\n", student.Code)
		return nil
	}

	_, err = DB.StudentCollection.UpdateOne(ctx, bson.M{"_id": certificate.StudentId}, update)
	if err != nil {
		return fmt.Errorf("failed to update student hours: %v", err)
	}

	// 5. บันทึกประวัติการเปลี่ยนแปลงชั่วโมง (status: rejected)
	// สร้าง HourChangeHistory record โดยใช้ status "rejected"
	remark := fmt.Sprintf("Certificate Rejected - Status changed to: %s", certificate.Status)
	if certificate.Remark != "" {
		remark = certificate.Remark // ใช้ remark จาก certificate ถ้ามี
	}

	hourChange := models.HourChangeHistory{
		ID:           primitive.NewObjectID(),
		StudentID:    certificate.StudentId,
		SkillType:    skillType,
		Status:       models.HCStatusRejected, // ใช้ "rejected" เพราะ certificate ถูกปฏิเสธ
		HourChange:   -hoursToRemove,          // ลบชั่วโมง (ค่าลบ)
		Remark:       remark,
		ChangeAt:     time.Now(),
		Title:        course.Name,
		SourceType:   "certificate",
		SourceID:     certificate.ID,
		EnrollmentID: nil, // certificate ไม่มี enrollmentID
	}

	_, err = DB.HourChangeHistoryCollection.InsertOne(ctx, hourChange)
	if err != nil {
		// Log warning but don't fail the operation
		fmt.Printf("Warning: Failed to save certificate history: %v\n", err)
	}

	fmt.Printf("❌ Removed %d hours (%s skill) from student %s for certificate %s\n",
		hoursToRemove, skillType, student.Code, certificate.ID.Hex())

	return nil
}

// recordCertificateRejection บันทึก hour history เมื่อ certificate ถูกปฏิเสธจาก pending
// ไม่มีการเปลี่ยนแปลงชั่วโมงจริง (hourChange = 0) แต่บันทึกเป็นประวัติ
func recordCertificateRejection(ctx context.Context, certificate *models.UploadCertificate, adminRemark string) error {
	// ไม่ต้องบันทึกถ้าเป็น duplicate
	if certificate.IsDuplicate {
		return nil
	}

	// ดึงข้อมูล course เพื่อหาประเภท skill
	course, err := courses.GetCourseByID(certificate.CourseId)
	if err != nil {
		return fmt.Errorf("course not found: %v", err)
	}

	skillType := "soft"
	if course.IsHardSkill {
		skillType = "hard"
	}

	remark := "Certificate Rejected"
	if adminRemark != "" {
		remark = adminRemark
	}

	// บันทึก hour history record โดยไม่มีการเปลี่ยนแปลงชั่วโมง
	hourChange := models.HourChangeHistory{
		ID:           primitive.NewObjectID(),
		StudentID:    certificate.StudentId,
		SkillType:    skillType,
		Status:       models.HCStatusRejected, // สถานะ rejected
		HourChange:   0,                       // ไม่มีการเปลี่ยนแปลงชั่วโมง
		Remark:       remark,
		ChangeAt:     time.Now(),
		Title:        course.Name,
		SourceType:   "certificate",
		SourceID:     certificate.ID,
		EnrollmentID: nil,
	}

	_, err = DB.HourChangeHistoryCollection.InsertOne(ctx, hourChange)
	if err != nil {
		return fmt.Errorf("failed to save certificate rejection history: %v", err)
	}

	fmt.Printf("📝 Recorded rejection for certificate %s (no hour changes)\n", certificate.ID.Hex())
	return nil
}

// recordCertificatePending บันทึก hour history เมื่อ certificate กลับไปสถานะ pending
// ไม่มีการเปลี่ยนแปลงชั่วโมงจริง (hourChange = 0) แต่บันทึกเป็นประวัติ
func recordCertificatePending(ctx context.Context, certificate *models.UploadCertificate, adminRemark string) error {
	// ไม่ต้องบันทึกถ้าเป็น duplicate
	if certificate.IsDuplicate {
		return nil
	}

	// ดึงข้อมูล course เพื่อหาประเภท skill
	course, err := courses.GetCourseByID(certificate.CourseId)
	if err != nil {
		return fmt.Errorf("course not found: %v", err)
	}

	skillType := "soft"
	if course.IsHardSkill {
		skillType = "hard"
	}

	remark := "Certificate Back to Pending Review"
	if adminRemark != "" {
		remark = adminRemark
	}

	// บันทึก hour history record โดยไม่มีการเปลี่ยนแปลงชั่วโมง
	hourChange := models.HourChangeHistory{
		ID:           primitive.NewObjectID(),
		StudentID:    certificate.StudentId,
		SkillType:    skillType,
		Status:       models.HCStatusPending, // สถานะ pending
		HourChange:   0,                      // ไม่มีการเปลี่ยนแปลงชั่วโมง
		Remark:       remark,
		ChangeAt:     time.Now(),
		Title:        course.Name,
		SourceType:   "certificate",
		SourceID:     certificate.ID,
		EnrollmentID: nil,
	}

	_, err = DB.HourChangeHistoryCollection.InsertOne(ctx, hourChange)
	if err != nil {
		return fmt.Errorf("failed to save certificate pending history: %v", err)
	}

	fmt.Printf("📝 Recorded pending status for certificate %s (no hour changes)\n", certificate.ID.Hex())
	return nil
}
