package controllers

import (
	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/src/services"
	"log"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CreateActivity - สร้างกิจกรรมใหม่
func CreateActivity(c *fiber.Ctx) error {
	var request struct {
		models.Activity
		ActivityItems []models.ActivityItem `json:"activityItems"`
	}

	// แปลง JSON เป็น struct
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	// บันทึก Activity + Items
	err := services.CreateActivity(request.Activity, request.ActivityItems)
	if err != nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Activity and ActivityItems created successfully",
	})
}

// GetAllActivities - Controller สำหรับดึงข้อมูลกิจกรรมทั้งหมด พร้อม ActivityItems
func GetAllActivities(c *fiber.Ctx) error {
	// เรียกใช้ service เพื่อดึงข้อมูลกิจกรรมทั้งหมด พร้อม ActivityItems
	activities, err := services.GetAllActivities()
	if err != nil {
		log.Println("Error retrieving activities:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Unable to fetch activities",
		})
	}

	// ส่งข้อมูลกิจกรรมทั้งหมดกลับ
	return c.Status(fiber.StatusOK).JSON(activities)
}

// GetActivityByID - ดึงข้อมูลกิจกรรมตาม ID พร้อม ActivityItems
func GetActivityByID(c *fiber.Ctx) error {
	id := c.Params("id")
	activityID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid ID format"})
	}

	// ดึงข้อมูล Activity พร้อม ActivityItems
	activity, activityItems, err := services.GetActivityByID(activityID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Activity not found"})
	}

	// ส่งข้อมูลกลับรวมทั้ง ActivityItems
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"activity":      activity,
		"activityItems": activityItems,
	})
}

// UpdateActivity - อัพเดตข้อมูลกิจกรรม พร้อม ActivityItems
func UpdateActivity(c *fiber.Ctx) error {
	id := c.Params("id")
	activityID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid ID format"})
	}

	var request struct {
		models.Activity
		ActivityItems []models.ActivityItem `json:"activityItems"`
	}
	// แปลง JSON เป็น struct
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	// อัปเดต Activity และ ActivityItems
	updatedActivity, updatedItems, err := services.UpdateActivity(activityID, request.Activity, request.ActivityItems)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"updatedActivity": updatedActivity,
		"updatedItems":    updatedItems,
	})
}

// DeleteActivity - ลบกิจกรรม พร้อม ActivityItems ที่เกี่ยวข้อง
func DeleteActivity(c *fiber.Ctx) error {
	id := c.Params("id")
	activityID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid ID format"})
	}

	// ลบ Activity พร้อม ActivityItems ที่เกี่ยวข้อง
	err = services.DeleteActivity(activityID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Activity and related ActivityItems deleted successfully"})
}
