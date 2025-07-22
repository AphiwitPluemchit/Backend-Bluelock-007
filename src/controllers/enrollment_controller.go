package controllers

import (
	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/src/services/enrollments"
	"net/http"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func CreateEnrollment(c *fiber.Ctx) error {
	var req struct {
		ActivityItemID string  `json:"activityItemId"`
		StudentID      string  `json:"studentId"`
		Food           *string `json:"food"` // ✅ รับชื่ออาหาร ถ้ามี
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input format"})
	}

	activityItemID, _ := primitive.ObjectIDFromHex(req.ActivityItemID)
	studentID, _ := primitive.ObjectIDFromHex(req.StudentID)

	err := enrollments.RegisterStudent(activityItemID, studentID, req.Food) // ✅ ส่ง food ไปด้วย
	if err != nil {
		return c.Status(http.StatusConflict).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{"message": "Enrollment successful"})
}

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
	activities, total, totalPages, err := enrollments.GetEnrollmentsByStudent(studentID, params, skillFilter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// ✅ 4. ส่ง response แบบเดียวกับ /activities
	return c.JSON(fiber.Map{
		"data": activities,
		"meta": fiber.Map{
			"page":       params.Page,
			"limit":      params.Limit,
			"total":      total,
			"totalPages": totalPages,
		},
	})
}

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

func GetStudentsByActivity(c *fiber.Ctx) error {
	activityId, err := primitive.ObjectIDFromHex(c.Params("activityId"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid activityItemId format"})
	}

	enrollmentData, err := enrollments.GetStudentsByActivity(activityId)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(enrollmentData)
}

func GetEnrollmentByStudentAndActivity(c *fiber.Ctx) error {
	studentID, _ := primitive.ObjectIDFromHex(c.Params("studentId"))
	activityItemID, _ := primitive.ObjectIDFromHex(c.Params("activityItemId"))

	enrollment, err := enrollments.GetEnrollmentByStudentAndActivity(studentID, activityItemID)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "Enrollment not found"})
	}

	return c.JSON(enrollment)
}

func CheckEnrollmentByStudentAndActivity(c *fiber.Ctx) error {
	studentIDHex := c.Params("studentId")
	activityIDHex := c.Params("activityId")

	studentID, err := primitive.ObjectIDFromHex(studentIDHex)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid studentId"})
	}

	activityID, err := primitive.ObjectIDFromHex(activityIDHex)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid activityId"})
	}

	enrollment, err := enrollments.GetStudentEnrollmentInActivity(studentID, activityID)
	if err != nil {
		if err.Error() == "Student not enrolled in this activity" {
			return c.JSON(fiber.Map{
				"isEnrolled": false,
				"message":    "Student not enrolled in this activity",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"isEnrolled": true,
		"enrollment": enrollment,
		"message":    "Student is enrolled in this activity",
	})
}

// GetStudentEnrollmentInActivity godoc
// @Summary      ดึงข้อมูล Enrollment ของ Student ใน Activity
// @Description  ดึงข้อมูล Enrollment ที่ Student ลงทะเบียนใน Activity นี้ (รวม activity และ activityItem details)
// @Tags         enrollments
// @Produce      json
// @Param        studentId path string true "Student ID"
// @Param        activityId path string true "Activity ID"
// @Success      200  {object}  bson.M
// @Failure      400  {object}  models.ErrorResponse
// @Failure      404  {object}  models.ErrorResponse
// @Failure      500  {object}  models.ErrorResponse
// @Router       /enrollments/student/{studentId}/activity/{activityId} [get]
func GetStudentEnrollmentInActivity(c *fiber.Ctx) error {
	studentIDHex := c.Params("studentId")
	activityIDHex := c.Params("activityId")

	studentID, err := primitive.ObjectIDFromHex(studentIDHex)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid studentId"})
	}

	activityID, err := primitive.ObjectIDFromHex(activityIDHex)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid activityId"})
	}

	enrollment, err := enrollments.GetStudentEnrollmentInActivity(studentID, activityID)
	if err != nil {
		if err.Error() == "Student not enrolled in this activity" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(enrollment)
}
