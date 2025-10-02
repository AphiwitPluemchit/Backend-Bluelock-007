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

// ✅Student ดูกิจกรรมที่ลงทะเบียนไปแล้ว
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

// ✅ 1.b Student หลายคน ลงทะเบียนกิจกรรมแบบ bulk: { studentCode, food } ต่อคน
func RegisterStudentsByCodes(c *fiber.Ctx) error {
	var req models.BulkEnrollRequest
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
	students := make([]models.BulkEnrollItem, len(req.Students))
	for i, s := range req.Students {
		students[i] = models.BulkEnrollItem{
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

// ✅Student ลงทะเบียนกิจกรรม
func RegisterStudent(c *fiber.Ctx) error {
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

// Student ยกเลิกการลงทะเบียน
func UnregisterStudent(c *fiber.Ctx) error {
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

// Student ดูกิจกรรมที่ลงทะเบียนไว้ (1 ตัว)
func GetEnrollmentByStudentAndProgram(c *fiber.Ctx) error {
	studentID, _ := primitive.ObjectIDFromHex(c.Params("studentId"))
	programItemID, _ := primitive.ObjectIDFromHex(c.Params("programItemId"))

	enrollment, err := enrollments.GetEnrollmentByStudentAndProgram(studentID, programItemID)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "Enrollment not found"})
	}

	return c.JSON(enrollment)
}

// ตรวจสอบว่านักศึกษาลงทะเบียนในกิจกรรมหรือไม่ และส่งข้อมูลกิจกรรม
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
	dateStr := c.Query("date") // รูปแบบ 2006-01-02
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
	if dateStr == "" {
		dateStr = c.Get("date")
	}
	log.Println(majorFilter)
	log.Println(statusFilter)
	log.Println(studentYearsFilter)
	log.Println(dateStr)
	student, total, err := enrollments.GetEnrollmentByProgramItemID(itemID, pagination, majorFilter, statusFilter, studentYearsFilter, dateStr)
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
	dateStr := c.Query("date")
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

	if dateStr == "" {
		dateStr = c.Get("date")
	}

	students, total, err := enrollments.GetEnrollmentsByProgramID(aID, pagination, majorFilter, statusFilter, studentYearsFilter, dateStr)
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
