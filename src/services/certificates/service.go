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

	"strconv"

	"github.com/chromedp/chromedp"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Thresholds controlled by environment variables. Defaults kept for backward compatibility.
var (
	nameApproveThreshold   = 80
	courseApproveThreshold = 80
	pendingThreshold       = 50
)

func init() {
	// Ensure .env is loaded for this package's init so environment-controlled
	// thresholds are picked up even if other packages load .env later.
	if err := godotenv.Load(); err != nil {
		// Not fatal; if .env not present we'll fall back to system env/defaults
		fmt.Println("⚠️ services: .env not found or failed to load")
	}
	if v := os.Getenv("NAME_APPROVE"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			nameApproveThreshold = parsed
		}
	}
	if v := os.Getenv("COURSE_APPROVE"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			courseApproveThreshold = parsed
		}
	}
	if v := os.Getenv("PENDING"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			pendingThreshold = parsed
		}
	}
}

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
		// ถ้าสถานะไม่เปลี่ยน แต่ remark เปลี่ยน ให้ update remark
		if oldCert.Remark != remark {
			fmt.Printf("Updating remark for certificate %s (status remains %s)\n", id, newStatus)
			now := time.Now()
			update := bson.M{
				"$set": bson.M{
					"remark":          remark,
					"changedStatusAt": now,
				},
			}
			_, err = DB.UploadCertificateCollection.UpdateOne(ctx, bson.M{"_id": objID}, update)
			if err != nil {
				return nil, fmt.Errorf("failed to update remark: %v", err)
			}

			// ดึงข้อมูล certificate ที่อัพเดทแล้ว
			var updatedCert models.UploadCertificate
			err = DB.UploadCertificateCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&updatedCert)
			if err != nil {
				return nil, err
			}
			return &updatedCert, nil
		}

		fmt.Printf("No status or remark change for certificate %s (already %s)\n", id, newStatus)
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
		fmt.Println("▶️ Adding hours for pending -> approved 1")

		certForHours.Remark = "อนุมัติโดยเจ้าหน้าที่"

		if err := updateCertificateHoursApproved(ctx, &certForHours); err != nil {
			return nil, fmt.Errorf("failed to add hours: %v", err)
		}
	}

	// กรณีที่ 2: approved -> rejected (Admin ปฏิเสธ certificate ที่เคยอนุมัติแล้ว)
	if oldCert.Status == models.StatusApproved && newStatus == models.StatusRejected {
		fmt.Println("▶️ Removing hours for approved -> rejected 2")

		if remark == "" {
			certForHours.Remark = "ปฏิเสธโดยเจ้าหน้าที่"
		} else {
			certForHours.Remark = remark
		}

		// fmt remark
		fmt.Printf("▶️ Old Remark: %s\n", oldCert.Remark)
		fmt.Printf("▶️ Remark for hours removal: %s\n", certForHours.Remark)

		if err := updateCertificateHoursRejected(ctx, &certForHours); err != nil {
			return nil, fmt.Errorf("failed to remove hours: %v", err)
		}
	}

	// กรณีที่ 3: rejected -> approved (Admin เปลี่ยนใจอนุมัติ)
	if oldCert.Status == models.StatusRejected && newStatus == models.StatusApproved {
		fmt.Println("▶️ Adding hours for rejected -> approved 3")

		certForHours.Remark = "อนุมัติโดยเจ้าหน้าที่"

		if err := updateCertificateHoursApproved(ctx, &certForHours); err != nil {
			return nil, fmt.Errorf("failed to add hours: %v", err)
		}
	}

	// กรณีที่ 4: approved -> pending (Admin ถอนการอนุมัติ ต้องรอพิจารณาใหม่)
	if oldCert.Status == models.StatusApproved && newStatus == models.StatusPending {
		fmt.Println("▶️ Removing hours for approved -> pending 4")
		if remark == "" {
			certForHours.Remark = "รอพิจารณาใหม่โดยเจ้าหน้าที่"
		} else {
			certForHours.Remark = remark
		}

		// ลบชั่วโมงที่เคยได้รับการอนุมัติ
		if err := updateCertificateHoursRejected(ctx, &certForHours); err != nil {
			return nil, fmt.Errorf("failed to remove hours: %v", err)
		}

		// บันทึก history record ด้วยสถานะ pending
		if err := recordCertificatePending(ctx, &certForHours, certForHours.Remark); err != nil {
			fmt.Printf("Warning: Failed to record certificate pending status: %v\n", err)
		}
	}

	// กรณีที่ 5: pending -> rejected (Admin ปฏิเสธตั้งแต่แรก - ไม่ต้องลบชั่วโมงเพราะไม่เคยเพิ่ม)
	// แต่ยังต้องบันทึก history record
	if oldCert.Status == models.StatusPending && newStatus == models.StatusRejected {
		fmt.Println("▶️ Rejecting pending certificate (no hours to remove) 5")

		if remark == "" {
			certForHours.Remark = "ปฏิเสธโดยเจ้าหน้าที่"
		} else {
			certForHours.Remark = remark
		}

		if err := recordCertificateRejection(ctx, &certForHours, remark); err != nil {
			fmt.Printf("Warning: Failed to record certificate rejection: %v\n", err)
		}
	}

	// กรณีที่ 6: rejected -> pending (Admin เปลี่ยนใจให้พิจารณาใหม่ - ไม่ต้องทำอะไร)
	// บันทึก history record ด้วยสถานะ pending
	if oldCert.Status == models.StatusRejected && newStatus == models.StatusPending {
		fmt.Println("▶️ Moving rejected certificate back to pending (no hours change) 6")

		if remark == "" {
			certForHours.Remark = "รอพิจารณาใหม่โดยเจ้าหน้าที่"
		} else {
			certForHours.Remark = remark
		}

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
	// Support multiple statuses separated by comma (e.g. status=pending,approved)
	if params.Status != "" {
		statuses := strings.Split(params.Status, ",")
		if len(statuses) == 1 {
			filter["status"] = params.Status
		} else {
			// Trim spaces and use $in
			for i := range statuses {
				statuses[i] = strings.TrimSpace(statuses[i])
			}
			filter["status"] = bson.M{"$in": statuses}
		}
	}

	// 2) Clean pagination
	pagination = models.CleanPagination(pagination)

	// 3) Build pipeline
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: filter}},
	}

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
	// If filtering by major or year is requested, we must join students to filter by their fields
	if params.Major != "" || params.Year != "" {
		needJoin = true
	}

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
		// If major filter provided, add a match for student.major
		if params.Major != "" {
			// support comma-separated majors or single major
			majors := strings.Split(params.Major, ",")
			if len(majors) == 1 {
				pipeline = append(pipeline,
					bson.D{{Key: "$match", Value: bson.M{
						"student.major": bson.M{"$regex": primitive.Regex{Pattern: strings.TrimSpace(majors[0]), Options: "i"}},
					}}},
				)
			} else {
				// build $in with regexes for case-insensitive matching
				var regexes []interface{}
				for _, m := range majors {
					m = strings.TrimSpace(m)
					if m == "" {
						continue
					}
					regexes = append(regexes, primitive.Regex{Pattern: m, Options: "i"})
				}
				if len(regexes) > 0 {
					pipeline = append(pipeline,
						bson.D{{Key: "$match", Value: bson.M{
							"student.major": bson.M{"$in": regexes},
						}}},
					)
				}
			}
		}
		// If year filter provided, filter by student code prefix (first 2 digits)
		if params.Year != "" {
			// support comma-separated years (e.g., "68,67,66")
			years := strings.Split(params.Year, ",")
			if len(years) == 1 {
				// Single year: match student.code starting with the year prefix
				yearPrefix := strings.TrimSpace(years[0])
				pipeline = append(pipeline,
					bson.D{{Key: "$match", Value: bson.M{
						"student.code": bson.M{"$regex": primitive.Regex{Pattern: "^" + yearPrefix, Options: "i"}},
					}}},
				)
			} else {
				// Multiple years: use $or with multiple regex patterns
				var orConditions []bson.M
				for _, y := range years {
					y = strings.TrimSpace(y)
					if y == "" {
						continue
					}
					orConditions = append(orConditions, bson.M{
						"student.code": bson.M{"$regex": primitive.Regex{Pattern: "^" + y, Options: "i"}},
					})
				}
				if len(orConditions) > 0 {
					pipeline = append(pipeline,
						bson.D{{Key: "$match", Value: bson.M{"$or": orConditions}}},
					)
				}
			}
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

	rows, meta, err := models.AggregatePaginateGlobal[models.UploadCertificate](
		ctx, DB.UploadCertificateCollection, pipeline, pagination.Page, pagination.Limit,
	)
	if err != nil {
		return nil, models.PaginationMeta{}, err
	}

	// Debug: number of returned rows
	return rows, meta, nil
}

func ThaiMooc(publicPageURL string, student models.Student, course models.Course) (*FastAPIResp, error) {
	// Use a cancellable context with timeout to avoid hanging on bad URLs
	timeout := 180 * time.Second
	if v := os.Getenv("THAIMOOC_TIMEOUT"); v != "" {
		if parsed, err := time.ParseDuration(v); err == nil {
			timeout = parsed
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
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
		// If it's a context deadline, persist an auto-rejected certificate and record history
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			fmt.Printf("ThaiMooc timeout after %v for URL %s\n", timeout, publicPageURL)
			if e := saveTimeoutRejection(context.Background(), publicPageURL, student, course, "ระบบปฏิเสธเนื่องจากหมดเวลาในการเข้าถึง URL"); e != nil {
				fmt.Printf("Warning: failed to save timeout rejection: %v\n", e)
			}
		}
		return nil, err
	}
	if pdfSrc == "" {
		return nil, errors.New("pdf <embed> not found or empty src")
	}
	// ตัดพารามิเตอร์ viewer ออก (#toolbar/navpanes/scrollbar)
	if i := strings.IndexByte(pdfSrc, '#'); i >= 0 {
		pdfSrc = pdfSrc[:i]
	}

	// Download PDF into memory (no disk write)
	pdfBytes, err := DownloadPDFToBytes(ctx, pdfSrc)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			fmt.Printf("ThaiMooc timeout after %v for URL %s\n", timeout, publicPageURL)
			if e := saveTimeoutRejection(context.Background(), publicPageURL, student, course, "Auto-rejected due to timeout while downloading PDF"); e != nil {
				fmt.Printf("Warning: failed to save timeout rejection: %v\n", e)
			}
		}
		return nil, err
	}

	// Ensure the FastAPI call respects the same context/timeout and send bytes
	response, err := callThaiMoocFastAPIWithContext(ctx,
		FastAPIURL(),
		pdfBytes,                 // pdf bytes
		student.Name,             // student_th
		student.EngName,          // student_en
		course.CertificateName,   // course_name
		course.CertificateNameEN, // course_name_en
	)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			fmt.Printf("ThaiMooc timeout after %v for URL %s\n", timeout, publicPageURL)
			if e := saveTimeoutRejection(context.Background(), publicPageURL, student, course, "Auto-rejected due to timeout while calling FastAPI"); e != nil {
				fmt.Printf("Warning: failed to save timeout rejection: %v\n", e)
			}
		}
		return nil, err
	}
	return response, nil
}

// saveTimeoutRejection creates an UploadCertificate record marked rejected and records rejection history.
func saveTimeoutRejection(ctx context.Context, publicPageURL string, student models.Student, course models.Course, reason string) error {
	uc := models.UploadCertificate{}
	uc.ID = primitive.NewObjectID()
	uc.IsDuplicate = false
	uc.StudentId = student.ID
	uc.CourseId = course.ID
	uc.UploadAt = time.Now()
	uc.NameMatch = 0
	uc.NameEngMatch = 0
	uc.CourseMatch = 0
	uc.CourseEngMatch = 0
	uc.Status = models.StatusRejected
	uc.Remark = reason
	uc.Url = publicPageURL

	saved, err := CreateUploadCertificate(&uc)
	if err != nil {
		return fmt.Errorf("failed to save timeout-rejected upload certificate: %v", err)
	}

	if err := recordCertificateRejection(context.Background(), saved, "URL ไม่พบข้อมูล หรืออาจมีปัญหาทางการเข้าถึง"); err != nil {
		// log but don't fail
		fmt.Printf("Warning: Failed to record certificate rejection history for timeout-rejected certificate %s: %v\n", saved.ID.Hex(), err)
	}
	fmt.Printf("Saved timeout-rejected certificate %s for URL %s\n", saved.ID.Hex(), publicPageURL)
	return nil
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

	response, err := callBUUMoocFastAPI(
		FastAPIURL(),
		string(body),             // html ที่ดึงมา
		studentNameTh,            // student_th
		studentNameEng,           // student_en
		course.CertificateName,   // course_name (ใช้ชื่อจาก certificate)
		course.CertificateNameEN, // course_name_en
	)
	if err != nil {
		return nil, err
	}
	return response, nil
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

// callThaiMoocFastAPIWithContext runs callThaiMoocFastAPI but returns early if ctx is done.
func callThaiMoocFastAPIWithContext(ctx context.Context, url string, pdfBytes []byte, studentTh string, studentEn string, courseName string, courseNameEn string) (*FastAPIResp, error) {
	type respWrap struct {
		resp *FastAPIResp
		err  error
	}
	ch := make(chan respWrap, 1)

	go func() {
		r, e := callThaiMoocFastAPI(url, pdfBytes, studentTh, studentEn, courseName, courseNameEn)
		ch <- respWrap{resp: r, err: e}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case out := <-ch:
		return out.resp, out.err
	}
}

// DownloadPDFToBytes downloads a PDF from the given URL into memory and returns bytes.
func DownloadPDFToBytes(ctx context.Context, pdfSrc string) ([]byte, error) {
	req, err := http.NewRequest("GET", pdfSrc, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req = req.WithContext(ctx)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error downloading PDF: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading PDF body: %v", err)
	}
	return b, nil
}

func checkDuplicateURL(publicPageURL string, studentId primitive.ObjectID, courseId primitive.ObjectID, excludeID *primitive.ObjectID) (bool, *models.UploadCertificate, error) {
	ctx := context.Background()

	var result models.UploadCertificate
	// Consider approved uploads as duplicates by default. For pending uploads,
	// allow a short grace window when the pending upload belongs to the same
	// student and course and was created just now -- this avoids race where a
	// pending record is created locally then immediately re-checked and treated
	// as a duplicate.
	filter := bson.M{"url": publicPageURL, "status": bson.M{"$in": bson.A{models.StatusApproved, models.StatusPending}}}
	err := DB.UploadCertificateCollection.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil, nil // URL is unique (no document found)
		}
		return false, nil, err // Some other error occurred
	}
	// If the found document is pending, only ignore it when it's the same
	// upload we're currently processing (excludeID). This avoids race where
	// the background job finds its own pending record and rejects itself.
	if result.Status == models.StatusPending {
		if excludeID != nil && result.ID == *excludeID {
			return false, nil, nil
		}
		// Otherwise treat pending as a duplicate (fall through)
	}

	// copy result to new object remove _id and mark as duplicate rejection record
	// newResult := models.UploadCertificate{}
	// newResult.IsDuplicate = true
	// newResult.StudentId = studentId
	// newResult.CourseId = courseId
	// newResult.UploadAt = time.Now()
	// newResult.NameMatch = 0
	// newResult.CourseMatch = 0
	// newResult.Status = models.StatusRejected
	// newResult.Remark = "Certificate URL already exists"
	// newResult.Url = publicPageURL
	// newResult.ID = primitive.NewObjectID()

	// createDuplicate, err := CreateUploadCertificate(&newResult)
	// if err != nil {
	// 	return false, nil, err
	// }

	// if err := recordCertificateRejection(context.Background(), createDuplicate, "Auto-rejected based on matching scores"); err != nil {
	// 	fmt.Printf("Warning: Failed to record certificate rejection for auto-rejected certificate %s: %v\n", createDuplicate.ID.Hex(), err)
	// }

	return true, &result, nil // URL already exists
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

	// Decide status using available course scores: take the max of Thai/EN course score
	courseMax := max(courseScore, courseScoreEn)

	// log thresholds and scores
	fmt.Printf("  Thresholds: NAME_APPROVE=%d, COURSE_APPROVE=%d, PENDING=%d\n", nameApproveThreshold, courseApproveThreshold, pendingThreshold)
	fmt.Printf("  Scores: nameMax=%d (TH=%d, EN=%d), courseMax=%d (TH=%d, EN=%d)\n",
		nameMax, nameScoreTh, nameScoreEn,
		courseMax, courseScore, courseScoreEn,
	)

	// Decide status using centralized thresholds from environment
	// - If both nameMax and courseMax >= NAME_APPROVE & COURSE_APPROVE => Approved
	// - Else if both nameMax and courseMax >= PENDING => Pending
	// - Otherwise => Rejected
	if nameMax >= nameApproveThreshold && courseMax >= courseApproveThreshold {
		uploadCertificate.Status = models.StatusApproved
	} else if nameMax >= pendingThreshold && courseMax >= pendingThreshold {
		uploadCertificate.Status = models.StatusPending
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

	// If FastAPI explicitly returned usedOcr, persist it. Otherwise leave nil (don't overwrite existing defaults).
	if res.UsedOCR != nil {
		uploadCertificate.UseOcr = res.UsedOCR
	}

	saved, err := CreateUploadCertificate(&uploadCertificate)
	if err != nil {
		return nil, err
	}

	// ถ้าสถานะเป็น approved ให้บันทึกชั่วโมงทันที (auto-approved)
	if saved.Status == models.StatusApproved {
		if err := updateCertificateHoursApproved(context.Background(), saved); err != nil {
			fmt.Printf("Warning: Failed to add certificate hours for auto-approved certificate %s: %v\n", saved.ID.Hex(), err)
		}
	}

	// ถ้าสถานะเป็น rejected ให้บันทึก history record ด้วย
	if saved.Status == models.StatusRejected {
		fmt.Println("Auto-rejected certificate, recording rejection history")
		if err := recordCertificateRejection(context.Background(), saved, "Auto-rejected based on matching scores"); err != nil {
			fmt.Printf("Warning: Failed to record certificate rejection for auto-rejected certificate %s: %v\n", saved.ID.Hex(), err)
		}
	}

	return saved, nil
}

// Reference to avoid "unused function" staticcheck when function is kept for future use
var _ = saveUploadCertificate

// ProcessPendingUpload finds an existing UploadCertificate by its hex ID and performs
// the full verification (calling fastapi/browser as needed), updates the document with
// scores, status and records history or hours. This is intended to be called as a
// background job so the HTTP request can return immediately.
func ProcessPendingUpload(uploadIDHex string) error {
	ctx := context.Background()
	objID, err := primitive.ObjectIDFromHex(uploadIDHex)
	if err != nil {
		return fmt.Errorf("invalid upload id: %v", err)
	}

	var uc models.UploadCertificate
	if err := DB.UploadCertificateCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&uc); err != nil {
		return fmt.Errorf("upload certificate not found: %v", err)
	}

	// Only process if status is pending
	if uc.Status != models.StatusPending {
		fmt.Printf("Upload %s is not pending (status=%s), skipping background processing\n", uploadIDHex, uc.Status)
		return nil
	}

	// Load student and course
	student, course, err := CheckStudentCourse(uc.StudentId.Hex(), uc.CourseId.Hex())
	if err != nil {
		return fmt.Errorf("failed to load student/course: %v", err)
	}

	// Check duplicate URL against already approved certificates
	// Pass the current upload ID so the duplicate checker can ignore the same pending record
	isDuplicate, existUC, err := checkDuplicateURL(uc.Url, uc.StudentId, uc.CourseId, &uc.ID)
	if err != nil {
		return fmt.Errorf("duplicate check failed: %v", err)
	}

	if isDuplicate {
		// Update current upload as rejected duplicate
		update := bson.M{"$set": bson.M{
			"isDuplicate":     true,
			"status":          models.StatusRejected,
			"remark":          "ใบรับรองนี้ถูกปฏิเสธโดยอัตโนมัติ เนื่องจากมี URL ซ้ำกับใบรับรองที่มีอยู่แล้ว",
			"changedStatusAt": time.Now(),
		}}
		if _, err := DB.UploadCertificateCollection.UpdateOne(ctx, bson.M{"_id": objID}, update); err != nil {
			return fmt.Errorf("failed to mark duplicate upload: %v", err)
		}
		// Finalize pending history as rejected (reuse helper)
		if err := finalizePendingHistoryRejected(context.Background(), &uc, course, "Certificate URL already exists"); err != nil {
			// fallback: still attempt to record rejection
			fmt.Printf("Warning: failed to finalize pending history for duplicate %s: %v\n", uploadIDHex, err)
			if rerr := recordCertificateRejection(context.Background(), &uc, "Auto-rejected based on matching scores"); rerr != nil {
				fmt.Printf("Warning: failed to record rejection history for %s: %v\n", uploadIDHex, rerr)
			}
		}
		fmt.Printf("Marked upload %s as duplicate (created duplicate record %s)\n", uploadIDHex, existUC.ID.Hex())
		return nil
	}

	// Perform verification depending on course type
	var res *FastAPIResp
	switch course.Type {
	case "buumooc":
		res, err = BuuMooc(uc.Url, student, course)
	case "thaimooc":
		res, err = ThaiMooc(uc.Url, student, course)
	default:
		return fmt.Errorf("invalid course type: %s", course.Type)
	}
	if err != nil {
		// On timeout or other errors, mark rejected with remark
		remark := fmt.Sprintf("ระบบปฏิเสธใบรับรองอัตโนมัติ อาจเกิดปัญหาในการเข้าถึง URL: %v", err)
		update := bson.M{"$set": bson.M{"status": models.StatusRejected, "remark": remark, "changedStatusAt": time.Now()}}
		if _, uerr := DB.UploadCertificateCollection.UpdateOne(ctx, bson.M{"_id": objID}, update); uerr != nil {
			return fmt.Errorf("failed to update upload after error: %v (update err: %v)", err, uerr)
		}
		// finalize pending history as rejected (update existing pending record if any)
		if ferr := finalizePendingHistoryRejected(context.Background(), &uc, course, remark); ferr != nil {
			fmt.Printf("Warning: failed to finalize pending rejection history for %s: %v\n", uploadIDHex, ferr)
			// fallback: insert rejection history
			if rerr := recordCertificateRejection(context.Background(), &uc, remark); rerr != nil {
				fmt.Printf("Warning: failed to record rejection history for %s: %v\n", uploadIDHex, rerr)
			}
		}
		return nil
	}
	if res == nil {
		return fmt.Errorf("nil response from fastapi for upload %s", uploadIDHex)
	}

	// Prepare fields to update on the existing upload record
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
	courseMax := max(courseScore, courseScoreEn)

	// log thresholds and scores
	fmt.Printf("  Thresholds: NAME_APPROVE=%d, COURSE_APPROVE=%d, PENDING=%d\n", nameApproveThreshold, courseApproveThreshold, pendingThreshold)
	fmt.Printf("  Scores: nameMax=%d (TH=%d, EN=%d), courseMax=%d (TH=%d, EN=%d)\n",
		nameMax, nameScoreTh, nameScoreEn,
		courseMax, courseScore, courseScoreEn,
	)

	newStatus := models.StatusRejected
	remark := "ระบบปฏิเสธใบรับรองอัตโนมัติ ตามคะแนนการตรวจสอบ"
	if nameMax >= nameApproveThreshold && courseMax >= courseApproveThreshold {
		newStatus = models.StatusApproved
		remark = "ใบรับรองได้รับการอนุมัติ"
	} else if nameMax >= pendingThreshold && courseMax >= pendingThreshold {
		newStatus = models.StatusPending
		remark = "ใบรับรองรอให้เจ้าหน้าที่ตรวจสอบ"
	}

	updateFields := bson.M{
		"nameMatch":       nameMax,
		"nameEngMatch":    nameScoreEn,
		"courseMatch":     courseScore,
		"courseEngMatch":  courseScoreEn,
		"status":          newStatus,
		"remark":          remark,
		"usedOcr":         res.UsedOCR,
		"changedStatusAt": time.Now(),
	}

	_, err = DB.UploadCertificateCollection.UpdateOne(ctx, bson.M{"_id": objID}, bson.M{"$set": updateFields})
	if err != nil {
		return fmt.Errorf("failed to update upload certificate: %v", err)
	}

	// Re-fetch updated doc for history/hours operations
	var updated models.UploadCertificate
	if err := DB.UploadCertificateCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&updated); err != nil {
		return fmt.Errorf("failed to fetch updated upload: %v", err)
	}

	// Update pending hour-history to final status and update student hours if approved
	// Finalize hour history and student hours using helper functions for clarity
	switch updated.Status {
	case models.StatusApproved:
		if err := finalizePendingHistoryApproved(context.Background(), &updated, course); err != nil {
			fmt.Printf("Warning: finalize approved history failed for %s: %v\n", uploadIDHex, err)
		}
	case models.StatusRejected:
		if err := finalizePendingHistoryRejected(context.Background(), &updated, course, "Auto-rejected based on matching scores"); err != nil {
			fmt.Printf("Warning: finalize rejected history failed for %s: %v\n", uploadIDHex, err)
		}
	default:
		// pending -> leave pending history as-is
	}

	return nil
}

// updateCertificateHoursApproved อัพเดท student hours และ hour history เมื่อ certificate ได้รับการอนุมัติ
func updateCertificateHoursApproved(ctx context.Context, certificate *models.UploadCertificate) error {
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

	// 5. อัพเดทหรือสร้าง hour history record
	// หา history record สำหรับ certificate นี้ (pending หรือ rejected)
	histFilter := bson.M{
		"sourceType": "certificate",
		"sourceId":   certificate.ID,
		"studentId":  certificate.StudentId,
		"status":     bson.M{"$in": []string{string(models.HCStatusPending), string(models.HCStatusRejected)}},
	}

	histUpdate := bson.M{
		"$set": bson.M{
			"status":     models.HCStatusApproved,
			"hourChange": course.Hour, // เพิ่มชั่วโมง
			"remark":     "อนุมัติใบรับรอง",
			"changeAt":   time.Now(),
			"title":      course.Name,
			"skillType":  skillType,
		},
	}

	updateResult, err := DB.HourChangeHistoryCollection.UpdateOne(ctx, histFilter, histUpdate)
	if err != nil {
		return fmt.Errorf("failed to update hour history: %v", err)
	}

	// ถ้าไม่มี record เดิม ให้สร้างใหม่
	if updateResult.MatchedCount == 0 {
		hourChange := models.HourChangeHistory{
			ID:           primitive.NewObjectID(),
			StudentID:    certificate.StudentId,
			SkillType:    skillType,
			Status:       models.HCStatusApproved,
			HourChange:   course.Hour,
			Remark:       "อนุมัติใบรับรอง",
			ChangeAt:     time.Now(),
			Title:        course.Name,
			SourceType:   "certificate",
			SourceID:     certificate.ID,
			EnrollmentID: nil,
		}

		_, err = DB.HourChangeHistoryCollection.InsertOne(ctx, hourChange)
		if err != nil {
			fmt.Printf("Warning: Failed to insert hour history: %v\n", err)
		}
		fmt.Printf("📝 Created new hour history for certificate %s\n", certificate.ID.Hex())
	} else {
		fmt.Printf("📝 Updated existing hour history (pending/rejected -> approved) for certificate %s\n", certificate.ID.Hex())
	}

	fmt.Printf("✅ Added %d hours (%s skill) to student %s for certificate %s\n",
		course.Hour, skillType, student.Code, certificate.ID.Hex())

	return nil
}

// updateCertificateHoursRejected อัพเดท student hours และ hour history เมื่อ certificate ถูกปฏิเสธหรือยกเลิก
func updateCertificateHoursRejected(ctx context.Context, certificate *models.UploadCertificate) error {
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

	// Log remarks
	fmt.Printf("▶️ Old Remark: %s\n", certificate.Remark)

	// 5. อัพเดทหรือสร้าง hour history record
	remark := "ปฏิเสธใบรับรอง"
	if certificate.Remark != "" {
		remark = certificate.Remark
	}

	fmt.Printf("▶️ New Remark for Hour History: %s\n", remark)

	// หา history record สำหรับ certificate นี้ (pending หรือ approved)
	histFilter := bson.M{
		"sourceType": "certificate",
		"sourceId":   certificate.ID,
		"studentId":  certificate.StudentId,
		"status":     bson.M{"$in": []string{string(models.HCStatusPending), string(models.HCStatusApproved)}},
	}

	// ตรวจสอบว่า record เดิมเป็นสถานะอะไร เพื่อใส่ hourChange ให้ถูกต้อง
	var existingHistory models.HourChangeHistory
	err = DB.HourChangeHistoryCollection.FindOne(ctx, histFilter).Decode(&existingHistory)

	var hourChangeValue int
	if err == nil {
		// มี record เดิม - ตรวจสอบว่าเดิมมีชั่วโมงหรือไม่
		if existingHistory.Status == models.HCStatusApproved {
			// เดิมเป็น approved (มีชั่วโมง) -> ลบชั่วโมง
			hourChangeValue = -hoursToRemove
		} else {
			// เดิมเป็น pending (ยังไม่มีชั่วโมง) -> ไม่มีการเปลี่ยนแปลง
			hourChangeValue = 0
		}
	} else {
		// ไม่มี record เดิม -> ไม่มีการเปลี่ยนแปลงชั่วโมง
		hourChangeValue = 0
	}

	histUpdate := bson.M{
		"$set": bson.M{
			"status":     models.HCStatusRejected,
			"hourChange": hourChangeValue,
			"remark":     remark,
			"changeAt":   time.Now(),
			"title":      course.Name,
			"skillType":  skillType,
		},
	}

	result, err := DB.HourChangeHistoryCollection.UpdateOne(ctx, histFilter, histUpdate)
	if err != nil {
		return fmt.Errorf("failed to update hour history: %v", err)
	}

	// ถ้าไม่มี record เดิม ให้สร้างใหม่
	if result.MatchedCount == 0 {
		hourChange := models.HourChangeHistory{
			ID:           primitive.NewObjectID(),
			StudentID:    certificate.StudentId,
			SkillType:    skillType,
			Status:       models.HCStatusRejected,
			HourChange:   -hoursToRemove, // ลบชั่วโมง (เพราะถ้าไม่มี record แสดงว่าเคย approved แล้ว)
			Remark:       remark,
			ChangeAt:     time.Now(),
			Title:        course.Name,
			SourceType:   "certificate",
			SourceID:     certificate.ID,
			EnrollmentID: nil,
		}

		_, err = DB.HourChangeHistoryCollection.InsertOne(ctx, hourChange)
		if err != nil {
			fmt.Printf("Warning: Failed to insert hour history: %v\n", err)
		}
		fmt.Printf("📝 Created new hour history for certificate %s\n", certificate.ID.Hex())
	} else {
		fmt.Printf("📝 Updated existing hour history (pending/approved -> rejected) for certificate %s (hourChange: %d)\n", certificate.ID.Hex(), hourChangeValue)
	}

	fmt.Printf("❌ Removed %d hours (%s skill) from student %s for certificate %s\n",
		hoursToRemove, skillType, student.Code, certificate.ID.Hex())

	return nil
}

// recordCertificateRejection อัพเดท hour history เมื่อ certificate ถูกปฏิเสธจาก pending
// ไม่มีการเปลี่ยนแปลงชั่วโมงจริง (hourChange = 0) แต่บันทึกเป็นประวัติ
func recordCertificateRejection(ctx context.Context, certificate *models.UploadCertificate, adminRemark string) error {
	// ดึงข้อมูล course เพื่อหาประเภท skill
	course, err := courses.GetCourseByID(certificate.CourseId)
	if err != nil {
		return fmt.Errorf("course not found: %v", err)
	}

	skillType := "soft"
	if course.IsHardSkill {
		skillType = "hard"
	}

	remark := "ปฏิเสธใบรับรอง"
	if adminRemark != "" {
		remark = adminRemark
	}

	// หา history record ที่ pending อยู่สำหรับ certificate นี้
	histFilter := bson.M{
		"sourceType": "certificate",
		"sourceId":   certificate.ID,
		"studentId":  certificate.StudentId,
		"status":     models.HCStatusPending, // หาเฉพาะตัวที่ pending อยู่
	}

	histUpdate := bson.M{
		"$set": bson.M{
			"status":     models.HCStatusRejected,
			"hourChange": 0, // ไม่มีการเปลี่ยนแปลงชั่วโมง
			"remark":     remark,
			"changeAt":   time.Now(),
			"title":      course.Name,
			"skillType":  skillType,
		},
	}

	result, err := DB.HourChangeHistoryCollection.UpdateOne(ctx, histFilter, histUpdate)
	if err != nil {
		return fmt.Errorf("failed to update hour history: %v", err)
	}

	// ถ้าไม่มี pending record ให้สร้างใหม่
	if result.MatchedCount == 0 {
		hourChange := models.HourChangeHistory{
			ID:           primitive.NewObjectID(),
			StudentID:    certificate.StudentId,
			SkillType:    skillType,
			Status:       models.HCStatusRejected,
			HourChange:   0,
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
		fmt.Printf("📝 Created new rejection history for certificate %s\n", certificate.ID.Hex())
	} else {
		fmt.Printf("📝 Updated existing pending history to rejected for certificate %s\n", certificate.ID.Hex())
	}

	return nil
}

// recordCertificatePending อัพเดท hour history เมื่อ certificate กลับไปสถานะ pending
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

	remark := "รอให้เจ้าหน้าที่ตรวจสอบ"
	if adminRemark != "" {
		remark = adminRemark
	}

	// หา history record ที่ rejected อยู่สำหรับ certificate นี้
	histFilter := bson.M{
		"sourceType": "certificate",
		"sourceId":   certificate.ID,
		"studentId":  certificate.StudentId,
		"status":     bson.M{"$in": []string{string(models.HCStatusRejected), string(models.HCStatusApproved)}}, // หาเฉพาะตัวที่ rejected อยู่
	}

	histUpdate := bson.M{
		"$set": bson.M{
			"status":     models.HCStatusPending,
			"hourChange": 0, // ไม่มีการเปลี่ยนแปลงชั่วโมง
			"remark":     remark,
			"changeAt":   time.Now(),
			"title":      course.Name,
			"skillType":  skillType,
		},
	}

	result, err := DB.HourChangeHistoryCollection.UpdateOne(ctx, histFilter, histUpdate)
	if err != nil {
		return fmt.Errorf("failed to update hour history: %v", err)
	}

	// ถ้าไม่มี rejected record ให้สร้างใหม่
	if result.MatchedCount == 0 {
		hourChange := models.HourChangeHistory{
			ID:           primitive.NewObjectID(),
			StudentID:    certificate.StudentId,
			SkillType:    skillType,
			Status:       models.HCStatusPending,
			HourChange:   0,
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
		fmt.Printf("📝 Created new pending history for certificate %s\n", certificate.ID.Hex())
	} else {
		fmt.Printf("📝 Updated existing rejected history to pending for certificate %s\n", certificate.ID.Hex())
	}

	return nil
}

// RecordUploadPending is an exported helper that controllers can call to record
// a pending-hour-history entry for a newly created upload certificate.
func RecordUploadPending(certificate *models.UploadCertificate, remark string) error {
	return recordCertificatePending(context.Background(), certificate, remark)
}

// finalizePendingHistoryApproved applies hours to the student (if applicable)
// and updates the pending HourChangeHistory for the given upload to approved.
func finalizePendingHistoryApproved(ctx context.Context, upload *models.UploadCertificate, course models.Course) error {
	// determine skill type
	skillType := "soft"
	if course.IsHardSkill {
		skillType = "hard"
	}

	// apply student hours if not duplicate
	if !upload.IsDuplicate && course.Hour > 0 && course.IsActive {
		var inc bson.M
		if skillType == "soft" {
			inc = bson.M{"$inc": bson.M{"softSkill": course.Hour}}
		} else {
			inc = bson.M{"$inc": bson.M{"hardSkill": course.Hour}}
		}
		if _, err := DB.StudentCollection.UpdateOne(ctx, bson.M{"_id": upload.StudentId}, inc); err != nil {
			return fmt.Errorf("failed to update student hours: %v", err)
		}
	}

	// Match any existing history for this upload (don't require status=pending)
	histFilter := bson.M{"sourceType": "certificate", "sourceId": upload.ID, "studentId": upload.StudentId}
	histUpdate := bson.M{"$set": bson.M{
		"status":     models.HCStatusApproved,
		"hourChange": course.Hour,
		"remark":     "อนุมัติใบรับรอง",
		"changeAt":   time.Now(),
		"title":      course.Name,
		"studentId":  upload.StudentId,
		"skillType":  skillType,
	}}

	res, _ := DB.HourChangeHistoryCollection.UpdateOne(ctx, histFilter, histUpdate)
	if res != nil && res.MatchedCount == 0 {
		// fallback: insert history
		_, err := DB.HourChangeHistoryCollection.InsertOne(ctx, models.HourChangeHistory{
			ID:         primitive.NewObjectID(),
			StudentID:  upload.StudentId,
			SkillType:  skillType,
			Status:     models.HCStatusApproved,
			HourChange: course.Hour,
			Remark:     "อนุมัติใบรับรอง",
			ChangeAt:   time.Now(),
			Title:      course.Name,
			SourceType: "certificate",
			SourceID:   upload.ID,
		})
		if err != nil {
			return fmt.Errorf("failed to insert approved history: %v", err)
		}
	}
	return nil
}

// finalizePendingHistoryRejected updates the pending HourChangeHistory to rejected.
// If none exists, it inserts a rejected history record.
func finalizePendingHistoryRejected(ctx context.Context, upload *models.UploadCertificate, course models.Course, remark string) error {
	skillType := "soft"
	if course.IsHardSkill {
		skillType = "hard"
	}

	// Match any existing history for this upload (don't require status=pending)
	histFilter := bson.M{"sourceType": "certificate", "sourceId": upload.ID, "studentId": upload.StudentId}
	histUpdate := bson.M{"$set": bson.M{
		"status":     models.HCStatusRejected,
		"hourChange": 0,
		"remark":     remark,
		"changeAt":   time.Now(),
		"title":      course.Name,
		"studentId":  upload.StudentId,
		"skillType":  skillType,
	}}

	res, _ := DB.HourChangeHistoryCollection.UpdateOne(ctx, histFilter, histUpdate)
	if res != nil && res.MatchedCount == 0 {
		_, err := DB.HourChangeHistoryCollection.InsertOne(ctx, models.HourChangeHistory{
			ID:         primitive.NewObjectID(),
			StudentID:  upload.StudentId,
			SkillType:  skillType,
			Status:     models.HCStatusRejected,
			HourChange: 0,
			Remark:     remark,
			ChangeAt:   time.Now(),
			Title:      course.Name,
			SourceType: "certificate",
			SourceID:   upload.ID,
		})
		if err != nil {
			return fmt.Errorf("failed to insert rejected history: %v", err)
		}
	}
	return nil
}
