package controllers

import (
	"Backend-Bluelock-007/src/services"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CreateEnrollment godoc
// @Summary      Student ลงทะเบียนกิจกรรม
// @Description  ให้นักศึกษาลงทะเบียนเข้าร่วมกิจกรรม
// @Tags         enrollments
// @Accept       json
// @Produce      json
// @Param        body body models.RequestCreateEnrollment true "ข้อมูลสำหรับการลงทะเบียนกิจกรรม"
// @Success      201  {object}  models.Enrollment
// @Failure      400  {object}  models.ErrorResponse
// @Failure      409  {object}  models.ErrorResponse
// @Failure      500  {object}  models.ErrorResponse
// @Router       /enrollments [post]

// ✅ 1. Student ลงทะเบียนกิจกรรม
func CreateEnrollment(c *fiber.Ctx) error {
	var req struct {
		ActivityItemID string `json:"activityItemId"`
		StudentID      string `json:"studentId"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input format"})
	}

	activityItemID, _ := primitive.ObjectIDFromHex(req.ActivityItemID)
	studentID, _ := primitive.ObjectIDFromHex(req.StudentID)

	err := services.RegisterStudent(activityItemID, studentID)
	if err != nil {
		return c.Status(http.StatusConflict).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{"message": "Enrollment successful"})
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
	studentID, err := primitive.ObjectIDFromHex(c.Params("studentId"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid studentId format"})
	}

	enrollments, err := services.GetEnrollmentsByStudent(studentID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(enrollments)
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

	err = services.UnregisterStudent(enrollmentID)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Enrollment deleted successfully"})
}

// GetStudentsByActivity godoc
// @Summary      ดูนักศึกษาที่ลงทะเบียนในกิจกรรม
// @Description  แอดมินสามารถดูรายชื่อนักศึกษาที่ลงทะเบียนในกิจกรรมได้
// @Tags         enrollments
// @Produce      json
// @Param        activityItemId path string true "Activity Item ID"
// @Success      200  {array}   models.StudentEnrollment
// @Failure      400  {object}  models.ErrorResponse
// @Failure      404  {object}  models.ErrorResponse
// @Router       /enrollments/activity/{activityItemId} [get]

// ✅ 4. Admin ดู Student ที่ลงทะเบียนในกิจกรรม
func GetStudentsByActivity(c *fiber.Ctx) error {
	activityId, err := primitive.ObjectIDFromHex(c.Params("activityId"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid activityItemId format"})
	}

	enrollmentData, err := services.GetStudentsByActivity(activityId)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(enrollmentData)
}

// GetEnrollmentByStudentAndActivity godoc
// @Summary      ดูรายละเอียดของกิจกรรมที่นักศึกษาลงทะเบียนไว้ (เฉพาะ 1 รายการ)
// @Description  นักศึกษาสามารถดูรายละเอียดของกิจกรรมที่ลงทะเบียนไว้
// @Tags         enrollments
// @Produce      json
// @Param        studentId path string true "Student ID"
// @Param        activityItemId path string true "Activity Item ID"
// @Success      200  {object}  models.Enrollment
// @Failure      400  {object}  models.ErrorResponse
// @Failure      404  {object}  models.ErrorResponse
// @Router       /enrollments/student/{studentId}/activity/{activityItemId} [get]

// ✅ 5. Student ดูกิจกรรมที่ลงทะเบียนไว้ (1 ตัว)
func GetEnrollmentByStudentAndActivity(c *fiber.Ctx) error {
	studentID, _ := primitive.ObjectIDFromHex(c.Params("studentId"))
	activityItemID, _ := primitive.ObjectIDFromHex(c.Params("activityItemId"))

	enrollment, err := services.GetEnrollmentByStudentAndActivity(studentID, activityItemID)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{"error": "Enrollment not found"})
	}

	return c.JSON(enrollment)
}
