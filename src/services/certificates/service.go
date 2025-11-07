package services

import (
	DB "Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/src/services/courses"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"strconv"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// ============================================================================
// CONFIGURATION - Environment variables และ thresholds
// ============================================================================

// Thresholds controlled by environment variables. Defaults kept for backward compatibility.
var (
	nameApproveThreshold   = 80 // เกณฑ์คะแนนชื่อสำหรับอนุมัติอัตโนมัติ
	courseApproveThreshold = 80 // เกณฑ์คะแนนคอร์สสำหรับอนุมัติอัตโนมัติ
	pendingThreshold       = 50 // เกณฑ์คะแนนสำหรับสถานะ pending
)

func init() {
	// โหลด environment variables จาก .env file
	if err := godotenv.Load(); err != nil {
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

// ============================================================================
// CRUD OPERATIONS - Create, Read, Update, Delete
// ============================================================================

// CreateUploadCertificate สร้าง certificate ใหม่พร้อม hour history
func CreateUploadCertificate(uploadCertificate *models.UploadCertificate) (*models.UploadCertificate, error) {
	ctx := context.Background()

	// สร้าง hour history record ก่อน
	course, err := courses.GetCourseByID(uploadCertificate.CourseId)
	if err != nil {
		return nil, fmt.Errorf("course not found: %v", err)
	}

	skillType := "soft"
	if course.IsHardSkill {
		skillType = "hard"
	}

	hourHistoryId := primitive.NewObjectID()
	hourHistory := models.HourChangeHistory{
		ID:           hourHistoryId,
		StudentID:    uploadCertificate.StudentId,
		SkillType:    skillType,
		Status:       models.HCStatusPending,
		HourChange:   0,
		Remark:       "รอให้เจ้าหน้าที่ตรวจสอบ",
		ChangeAt:     time.Now(),
		Title:        course.Name,
		SourceType:   "certificate",
		SourceID:     &uploadCertificate.ID,
		EnrollmentID: nil,
	}

	_, err = DB.HourChangeHistoryCollection.InsertOne(ctx, hourHistory)
	if err != nil {
		return nil, fmt.Errorf("failed to create hour history: %v", err)
	}

	// ตั้งค่า hourHistoryId ให้กับ certificate
	uploadCertificate.HourHistoryId = &hourHistoryId

	result, err := DB.UploadCertificateCollection.InsertOne(ctx, uploadCertificate)
	if err != nil {
		// ถ้าสร้าง certificate ไม่สำเร็จ ลบ hour history ที่สร้างไว้
		DB.HourChangeHistoryCollection.DeleteOne(ctx, bson.M{"_id": hourHistoryId})
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

	fmt.Printf("📝 Created certificate %s with hour history ID: %s\n", insertedDoc.ID.Hex(), hourHistoryId.Hex())

	return &insertedDoc, nil
}

// UpdateUploadCertificate อัพเดทข้อมูล certificate
func UpdateUploadCertificate(id string, uploadCertificate *models.UploadCertificate) (*mongo.UpdateResult, error) {
	ctx := context.Background()
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid upload certificate ID")
	}
	return DB.UploadCertificateCollection.UpdateOne(ctx, bson.M{"_id": objID}, bson.M{"$set": uploadCertificate})
}

// ============================================================================
// STATUS MANAGEMENT - จัดการสถานะของ certificate
// ============================================================================

// UpdateUploadCertificateStatus อัพเดทสถานะของ certificate และจัดการชั่วโมงอัตโนมัติ
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

// ============================================================================
// QUERY OPERATIONS - ค้นหาและดึงข้อมูล certificate
// ============================================================================

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
					"$or": []bson.M{
						{
							"student.name": bson.M{
								"$regex": primitive.Regex{Pattern: pagination.Search, Options: "i"},
							},
						},
						{
							"student.code": bson.M{
								"$regex": primitive.Regex{Pattern: pagination.Search, Options: "i"},
							},
						},
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

// Reference to avoid "unused function" staticcheck when function is kept for future use
// Note: saveUploadCertificate was removed during refactor; if needed re-add.

// ============================================================================
// OCR PIPELINE - ระบบประมวลผลและตรวจสอบใบรับรองด้วย OCR และ Auto-classification
// ============================================================================

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
		duplicateRemark := "ระบบปฏิเสธใบรับรองอัตโนมัติ: URL นี้ถูกใช้งานแล้วในใบรับรองที่ได้รับการอนุมัติก่อนหน้านี้"
		update := bson.M{"$set": bson.M{
			"isDuplicate":     true,
			"status":          models.StatusRejected,
			"remark":          duplicateRemark,
			"changedStatusAt": time.Now(),
		}}
		if _, err := DB.UploadCertificateCollection.UpdateOne(ctx, bson.M{"_id": objID}, update); err != nil {
			return fmt.Errorf("failed to mark duplicate upload: %v", err)
		}
		// Finalize pending history as rejected (reuse helper)
		if err := finalizePendingHistoryRejected(context.Background(), &uc, *course, duplicateRemark); err != nil {
			// fallback: still attempt to record rejection
			fmt.Printf("Warning: failed to finalize pending history for duplicate %s: %v\n", uploadIDHex, err)
			if rerr := recordCertificateRejection(context.Background(), &uc, duplicateRemark); rerr != nil {
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
		var remark string
		if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "context deadline exceeded") {
			remark = "ระบบปฏิเสธใบรับรองอัตโนมัติ: ไม่สามารถเข้าถึง URL ได้ภายในเวลาที่กำหนด (Timeout)"
		} else if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not found") {
			remark = "ระบบปฏิเสธใบรับรองอัตโนมัติ: ไม่พบหน้าเว็บที่ระบุ (404 Not Found)"
		} else if strings.Contains(err.Error(), "certificate has expired") || strings.Contains(err.Error(), "ssl") {
			remark = "ระบบปฏิเสธใบรับรองอัตโนมัติ: เกิดปัญหาด้านความปลอดภัยของการเชื่อมต่อ (SSL Error)"
		} else {
			remark = fmt.Sprintf("ระบบปฏิเสธใบรับรองอัตโนมัติ: เกิดข้อผิดพลาดในการตรวจสอบ (%v)", err)
		}
		update := bson.M{"$set": bson.M{"status": models.StatusRejected, "remark": remark, "changedStatusAt": time.Now()}}
		if _, uerr := DB.UploadCertificateCollection.UpdateOne(ctx, bson.M{"_id": objID}, update); uerr != nil {
			return fmt.Errorf("failed to update upload after error: %v (update err: %v)", err, uerr)
		}
		// finalize pending history as rejected (update existing pending record if any)
		if ferr := finalizePendingHistoryRejected(context.Background(), &uc, *course, remark); ferr != nil {
			fmt.Printf("Warning: failed to finalize pending rejection history for %s: %v\n", uploadIDHex, ferr)
			// fallback: insert rejection history
			if rerr := recordCertificateRejection(context.Background(), &uc, remark); rerr != nil {
				fmt.Printf("Warning: failed to record rejection history for %s: %v\n", uploadIDHex, rerr)
			}
		}

		return nil
	}

	// ------------- Success path: we have a response in res --------------
	// Update some diagnostic fields returned by the verifier
	updFields := bson.M{
		"autoVerified":    res.IsVerified,
		"isNameMatch":     res.IsNameMatch,
		"isCourseMatch":   res.IsCourseMatch,
		"nameScoreTh":     res.NameScoreTh,
		"nameScoreEn":     res.NameScoreEn,
		"courseScore":     res.CourseScore,
		"courseScoreEn":   res.CourseScoreEn,
		"usedOcr":         res.UsedOCR,
		"changedStatusAt": time.Now(),
	}

	// Decide status based on thresholds
	// default to rejected
	chosenStatus := models.StatusRejected
	remark := "ระบบประมวลผลอัตโนมัติ: ผลการตรวจสอบ"

	// helper to get score value (nil-safe)
	getInt := func(p *int) int {
		if p == nil {
			return 0
		}
		return *p
	}

	nameScore := getInt(res.NameScoreTh)
	if nameScore == 0 {
		nameScore = getInt(res.NameScoreEn)
	}
	courseScore := getInt(res.CourseScore)
	if courseScore == 0 {
		courseScore = getInt(res.CourseScoreEn)
	}

	if res.IsVerified && nameScore >= nameApproveThreshold && courseScore >= courseApproveThreshold {
		chosenStatus = models.StatusApproved
		remark = "อนุมัติอัตโนมัติ"
	} else if res.IsVerified && (nameScore >= pendingThreshold || courseScore >= pendingThreshold) {
		chosenStatus = models.StatusPending
		remark = "รอพิจารณา: ผลการตรวจสอบอัตโนมัติไม่ถึงเกณฑ์อนุมัติ"
	} else {
		chosenStatus = models.StatusRejected
		remark = "ระบบปฏิเสธใบรับรองอัตโนมัติ"
	}

	// Apply update to DB
	update := bson.M{"$set": updFields}
	update["$set"].(bson.M)["status"] = chosenStatus
	update["$set"].(bson.M)["remark"] = remark

	if _, uerr := DB.UploadCertificateCollection.UpdateOne(ctx, bson.M{"_id": objID}, update); uerr != nil {
		return fmt.Errorf("failed to update upload after verification: %v", uerr)
	}

	// Reload updated certificate
	if err := DB.UploadCertificateCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&uc); err != nil {
		return fmt.Errorf("failed to reload upload certificate after verification: %v", err)
	}

	// finalize history depending on chosenStatus
	switch chosenStatus {
	case models.StatusApproved:
		if err := finalizePendingHistoryApproved(context.Background(), &uc, *course); err != nil {
			fmt.Printf("Warning: failed to finalize pending history approved for %s: %v\n", uploadIDHex, err)
		}
	case models.StatusRejected:
		if err := finalizePendingHistoryRejected(context.Background(), &uc, *course, remark); err != nil {
			fmt.Printf("Warning: failed to finalize pending history rejected for %s: %v\n", uploadIDHex, err)
		}
	case models.StatusPending:
		if err := recordCertificatePending(context.Background(), &uc, remark); err != nil {
			fmt.Printf("Warning: failed to record pending history for %s: %v\n", uploadIDHex, err)
		}
	}

	return nil
}
