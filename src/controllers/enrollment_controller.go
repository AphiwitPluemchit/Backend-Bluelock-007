package controllers

import (
	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/src/services/enrollments"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CreateEnrollment godoc
// @Summary      ลงทะเบียนกิจกรรม
// @Description  นักศึกษาสามารถลงทะเบียนกิจกรรมได้
// @Tags         enrollments
// @Accept       json
// @Produce      json
// @Param        enrollment body models.Enrollment true "Enrollment data"
// @Success      201  {object}  models.SuccessResponse
// @Failure      400  {object}  models.ErrorResponse
// @Failure      409  {object}  models.ErrorResponse
// @Router       /enrollments [post]
// ✅ 1. Student ลงทะเบียนกิจกรรม
func CreateEnrollment(c *fiber.Ctx) error {
	var req struct {
		ProgramItemID string  `json:"programItemId"`
		StudentID     string  `json:"studentId"`
		Food          *string `json:"food"` // ✅ รับชื่ออาหาร ถ้ามี
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input format"})
	}

	programItemID, _ := primitive.ObjectIDFromHex(req.ProgramItemID)
	studentID, _ := primitive.ObjectIDFromHex(req.StudentID)

	err := enrollments.RegisterStudent(programItemID, studentID, req.Food) // ✅ ส่ง food ไปด้วย
	if err != nil {
		return c.Status(http.StatusConflict).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{"message": "Enrollment successful"})
}

type bulkEnrollItem struct {
	StudentCode string  `json:"studentCode"`
	Food        *string `json:"food"` // ต่อคนเลือกได้
}

type bulkEnrollReq struct {
	ProgramItemID string           `json:"programItemId"`
	Students      []bulkEnrollItem `json:"students"`
}

// ✅ 1.b Student ลงทะเบียนกิจกรรมแบบ bulk: { studentCode, food } ต่อคน
func CreateBulkEnrollment(c *fiber.Ctx) error {
	var req bulkEnrollReq
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input format"})
	}
	if req.ProgramItemID == "" || len(req.Students) == 0 {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "programItemId and students are required"})
	}

	programItemID, err := primitive.ObjectIDFromHex(req.ProgramItemID)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid programItemId"})
	}

	// Convert []bulkEnrollItem to []enrollments.BulkEnrollItem
	students := make([]enrollments.BulkEnrollItem, len(req.Students))
	for i, s := range req.Students {
		students[i] = enrollments.BulkEnrollItem{
			StudentCode: s.StudentCode,
			Food:        s.Food,
		}
	}

	result, err := enrollments.RegisterStudentsByCodes(c.Context(), programItemID, students)
	if err != nil {
		// error ระดับระบบ — ส่ง payload ผลลัพธ์บางส่วนกลับไปด้วย
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error":  err.Error(),
			"result": result,
		})
	}

	return c.Status(http.StatusOK).JSON(result)
}

// GetEnrollmentsByStudent godoc
// @Summary      ดึงรายการกิจกรรมที่นักศึกษาลงทะเบียนไว้
// @Description  ให้นักศึกษาดูรายการกิจกรรมที่ลงทะเบียนไว้ทั้งหมด
// @Tags         enrollments
// @Produce      json
// @Param        studentId path string true "Student ID"
// @Success      200  {array}   models.Enrollment
// @Failure      400  {object}  models.ErrorResponse
// @Failure      500  {object}  models.ErrorResponse
// @Router       /enrollments/student/{studentId} [get]
// ✅ 2. Student ดูกิจกรรมที่ลงทะเบียนไปแล้ว
func GetEnrollmentsByStudent(c *fiber.Ctx) error {
	// 🔍 แปลง studentId จาก path param
	studentID, err := primitive.ObjectIDFromHex(c.Params("studentId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid studentId format"})
	}

	// ✅ 1. ตั้งค่าพารามิเตอร์แบ่งหน้า
	params := models.DefaultPagination()
	params.Page, _ = strconv.Atoi(c.Query("page", strconv.Itoa(params.Page)))
	params.Limit, _ = strconv.Atoi(c.Query("limit", strconv.Itoa(params.Limit)))
	params.Search = c.Query("search", "")
	params.SortBy = c.Query("sortBy", "name")
	params.Order = c.Query("order", "asc")

	// ✅ 2. แปลง Query skill เป็น array
	skillFilter := strings.Split(c.Query("skills"), ",")
	if len(skillFilter) == 1 && skillFilter[0] == "" {
		skillFilter = []string{}
	}

	// ✅ 3. เรียก service
	programs, total, totalPages, err := enrollments.GetEnrollmentsByStudent(studentID, params, skillFilter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// ✅ 4. ส่ง response แบบเดียวกับ /programs
	return c.JSON(fiber.Map{
		"data": programs,
		"meta": fiber.Map{
			"page":       params.Page,
			"limit":      params.Limit,
			"total":      total,
			"totalPages": totalPages,
		},
	})
}

// DeleteEnrollment godoc
// @Summary      ยกเลิกการลงทะเบียนกิจกรรม
// @Description  นักศึกษาสามารถยกเลิกการลงทะเบียนกิจกรรมได้
// @Tags         enrollments
// @Param        enrollmentId path string true "Enrollment ID"
// @Success      200  {object}  models.SuccessResponse
// @Failure      400  {object}  models.ErrorResponse
// @Failure      404  {object}  models.ErrorResponse
// @Router       /enrollments/{enrollmentId} [delete]
// ✅ 3. Student ยกเลิกการลงทะเบียน
func DeleteEnrollment(c *fiber.Ctx) error {
	enrollmentID, err := primitive.ObjectIDFromHex(c.Params("enrollmentId"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid enrollmentId format"})
	}

	err = enrollments.UnregisterStudent(enrollmentID)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Enrollment deleted successfully"})
}

// GetStudentsByProgram godoc
// @Summary      ดูนักศึกษาที่ลงทะเบียนในกิจกรรม
// @Description  แอดมินสามารถดูรายชื่อนักศึกษาที่ลงทะเบียนในกิจกรรมได้
// @Tags         enrollments
// @Produce      json
// @Param        programItemId path string true "Program Item ID"
// @Success      200  {array}   models.Enrollment
// @Failure      400  {object}  models.ErrorResponse
// @Failure      404  {object}  models.ErrorResponse
// @Router       /enrollments/program/{programItemId} [get]
// ✅ 4. Admin ดู Student ที่ลงทะเบียนในกิจกรรม
func GetStudentsByProgram(c *fiber.Ctx) error {
	programId, err := primitive.ObjectIDFromHex(c.Params("programId"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid programItemId format"})
	}

	enrollmentData, err := enrollments.GetStudentsByProgram(programId)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(enrollmentData)
}

// GetEnrollmentByStudentAndProgram godoc
// @Summary      ดูรายละเอียดของกิจกรรมที่นักศึกษาลงทะเบียนไว้ (เฉพาะ 1 รายการ)
// @Description  นักศึกษาสามารถดูรายละเอียดของกิจกรรมที่ลงทะเบียนไว้
// @Tags         enrollments
// @Produce      json
// @Param        studentId path string true "Student ID"
// @Param        programItemId path string true "Program Item ID"
// @Success      200  {object}  models.EnrollmentSummary
// @Failure      400  {object}  models.ErrorResponse
// @Failure      404  {object}  models.ErrorResponse
// @Router       /enrollments/student/{studentId}/programItem/{programItemId} [get]
// ✅ 5. Student ดูกิจกรรมที่ลงทะเบียนไว้ (1 ตัว)
func GetEnrollmentByStudentAndProgram(c *fiber.Ctx) error {
	studentID, _ := primitive.ObjectIDFromHex(c.Params("studentId"))
	programItemID, _ := primitive.ObjectIDFromHex(c.Params("programItemId"))

	enrollment, err := enrollments.GetEnrollmentByStudentAndProgram(studentID, programItemID)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "Enrollment not found"})
	}

	return c.JSON(enrollment)
}

// CheckEnrollmentByStudentAndProgram godoc
// @Summary      ตรวจสอบว่านักศึกษาลงทะเบียนในกิจกรรมหรือไม่
// @Description  ตรวจสอบว่านักศึกษาได้ลงทะเบียนในกิจกรรมนี้หรือไม่ และส่งข้อมูลกิจกรรมที่คล้ายกับ program getOne
// @Tags         enrollments
// @Produce      json
// @Param        studentId path string true "Student ID"
// @Param        programId path string true "Program ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  models.ErrorResponse
// @Failure      500  {object}  models.ErrorResponse
// @Router       /enrollments/student/{studentId}/program/{programId}/check [get]
// ✅ 5. ตรวจสอบว่านักศึกษาลงทะเบียนในกิจกรรมหรือไม่ และส่งข้อมูลกิจกรรม
func CheckEnrollmentByStudentAndProgram(c *fiber.Ctx) error {
	studentIDHex := c.Params("studentId")
	programIDHex := c.Params("programId")
	log.Println("check")
	studentID, err := primitive.ObjectIDFromHex(studentIDHex)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid studentId"})
	}

	programID, err := primitive.ObjectIDFromHex(programIDHex)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid programId"})
	}

	programDetails, err := enrollments.GetEnrollmentProgramDetails(studentID, programID)
	if err != nil {
		if err.Error() == "Student not enrolled in this program" {
			return c.JSON(fiber.Map{
				"isEnrolled": false,
				"message":    "Student not enrolled in this program",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	log.Println(studentID, programDetails.ProgramItems[0].ID)
	enrollmentId, err := enrollments.GetEnrollmentId(studentID, programDetails.ProgramItems[0].ID)
	if err != nil {
		if err.Error() == "Student not enrolled in this program" {
			return c.JSON(fiber.Map{
				"isEnrolled": false,
				"message":    "Student not enrolled in this program",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{
		"isEnrolled":   true,
		"enrollmentId": enrollmentId.Hex(),
		"program":      programDetails,
		"message":      "Student is enrolled in this program",
	})
}

// GetStudentEnrollmentInProgram godoc
// @Summary      ดึงข้อมูล Enrollment ของ Student ใน Program
// @Description  ดึงข้อมูล Enrollment ที่ Student ลงทะเบียนใน Program นี้ (รวม program และ programItem details)
// @Tags         enrollments
// @Produce      json
// @Param        studentId path string true "Student ID"
// @Param        programId path string true "Program ID"
// @Success      200  {object}  models.Enrollment
// @Failure      400  {object}  models.ErrorResponse
// @Failure      404  {object}  models.ErrorResponse
// @Failure      500  {object}  models.ErrorResponse
// @Router       /enrollments/student/{studentId}/program/{programId} [get]
func GetStudentEnrollmentInProgram(c *fiber.Ctx) error {
	studentIDHex := c.Params("studentId")
	programIDHex := c.Params("programId")

	studentID, err := primitive.ObjectIDFromHex(studentIDHex)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid studentId"})
	}

	programID, err := primitive.ObjectIDFromHex(programIDHex)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid programId"})
	}

	enrollment, err := enrollments.GetStudentEnrollmentInProgram(studentID, programID)
	if err != nil {
		if err.Error() == "Student not enrolled in this program" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(enrollment)
}

// ✅ 6. Student ดูกิจกรรมที่ลงทะเบียนไปแล้ว (History)
func GetRegistrationHistoryStatus(c *fiber.Ctx) error {
	// 🔍 แปลง studentId จาก path param
	studentID, err := primitive.ObjectIDFromHex(c.Params("studentId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid studentId format"})
	}

	// ✅ 1. ตั้งค่าพารามิเตอร์แบ่งหน้า
	params := models.DefaultPagination()
	params.Page, _ = strconv.Atoi(c.Query("page", strconv.Itoa(params.Page)))
	params.Limit, _ = strconv.Atoi(c.Query("limit", strconv.Itoa(params.Limit)))
	params.Search = c.Query("search", "")
	params.SortBy = c.Query("sortBy", "name")
	params.Order = c.Query("order", "asc")

	// ✅ 2. แปลง Query skill เป็น array
	skillFilter := strings.Split(c.Query("skills"), ",")
	if len(skillFilter) == 1 && skillFilter[0] == "" {
		skillFilter = []string{}
	}

	// ✅ 3. เรียก service (service จะ filter ให้เหลือเฉพาะ programItems ที่นิสิตลง + format checkin/checkout เป็นเวลาไทย)
	programs, total, totalPages, err := enrollments.GetRegistrationHistoryStatus(studentID, params, skillFilter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// ✅ 4. ส่ง response
	return c.JSON(fiber.Map{
		"data": programs,
		"meta": fiber.Map{
			"page":       params.Page,
			"limit":      params.Limit,
			"total":      total,
			"totalPages": totalPages,
		},
	})
}

// GetRegistrationHistoryStatus godoc
// @Summary      ประวัติการลงทะเบียนโครงการ (แบ่งสถานะ)
// @Description  คืนกลุ่มสถานะ: ยังไม่เข้าร่วม, เข้าร่วมแล้ว, ลงทะเบียนแต่ไม่ได้เข้าร่วม โดยอิงจาก Hour_Change_Histories
// @Tags         enrollments
// @Produce      json
// @Param        studentId path string true "Student ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  models.ErrorResponse
// @Failure      500  {object}  models.ErrorResponse
// @Router       /enrollments/history-status/student/{studentId} [get]
func GetEnrollmentsHistoryByStudent(c *fiber.Ctx) error {
	studentID, err := primitive.ObjectIDFromHex(c.Params("studentId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid studentId format"})
	}

	status, err := enrollments.GetEnrollmentsHistoryByStudent(studentID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(status)
}

func GetEnrollmentByProgramItemID(c *fiber.Ctx) error {
	programItemID := c.Params("id")
	itemID, err := primitive.ObjectIDFromHex(programItemID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid ID format"})
	}

	// อ่านค่าพารามิเตอร์การแบ่งหน้า
	pagination := models.DefaultPagination()
	if err := c.QueryParser(&pagination); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid pagination parameters"})
	}
	log.Println(pagination)
	// รับค่า query param
	studentMajors := c.Query("major")
	studentStatus := c.Query("studentStatus")
	studentYears := c.Query("studentYear")

	var majorFilter []string
	if studentMajors != "" {
		majorFilter = strings.Split(studentMajors, ",")
	}

	var statusFilter []int
	if studentStatus != "" {
		statusValues := strings.Split(studentStatus, ",")
		for _, val := range statusValues {
			if num, err := strconv.Atoi(val); err == nil {
				statusFilter = append(statusFilter, num)
			}
		}
	}

	var studentYearsFilter []int
	if studentYears != "" {
		studentYearsValues := strings.Split(studentYears, ",")
		for _, val := range studentYearsValues {
			if num, err := strconv.Atoi(val); err == nil {
				studentYearsFilter = append(studentYearsFilter, num)
			}
		}
	}
	log.Println(majorFilter)
	log.Println(statusFilter)
	log.Println(studentYearsFilter)
	student, total, err := enrollments.GetEnrollmentByProgramItemID(itemID, pagination, majorFilter, statusFilter, studentYearsFilter)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "ProgramItem not found",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": student,
		"meta": fiber.Map{
			"currentPage": pagination.Page,
			"perPage":     pagination.Limit,
			"total":       total,
			"totalPages":  (total + int64(pagination.Limit) - 1) / int64(pagination.Limit),
		},
	})
}

// GET /enrollments/by-program/:id
func GetEnrollmentsByProgramID(c *fiber.Ctx) error {
	programID := c.Params("id")
	aID, err := primitive.ObjectIDFromHex(programID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid ID format"})
	}

	// อ่าน pagination
	pagination := models.DefaultPagination()
	if err := c.QueryParser(&pagination); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid pagination parameters"})
	}

	// ฟิลเตอร์
	studentMajors := c.Query("major")
	studentStatus := c.Query("studentStatus")
	studentYears := c.Query("studentYear")

	var majorFilter []string
	if studentMajors != "" {
		majorFilter = strings.Split(studentMajors, ",")
	}

	var statusFilter []int
	if studentStatus != "" {
		for _, v := range strings.Split(studentStatus, ",") {
			if num, err := strconv.Atoi(v); err == nil {
				statusFilter = append(statusFilter, num)
			}
		}
	}

	var studentYearsFilter []int
	if studentYears != "" {
		for _, v := range strings.Split(studentYears, ",") {
			if num, err := strconv.Atoi(v); err == nil {
				studentYearsFilter = append(studentYearsFilter, num)
			}
		}
	}

	students, total, err := enrollments.GetEnrollmentsByProgramID(aID, pagination, majorFilter, statusFilter, studentYearsFilter)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "Program not found or no program items",
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": students,
		"meta": fiber.Map{
			"currentPage": pagination.Page,
			"perPage":     pagination.Limit,
			"total":       total,
			"totalPages":  (total + int64(pagination.Limit) - 1) / int64(pagination.Limit),
		},
	})
}
