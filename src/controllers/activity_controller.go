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
	var activity models.Activity
	// แปลง JSON เป็น struct
	if err := c.BodyParser(&activity); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	// บันทึก Activity + Items
	err := services.CreateActivity(activity)
	if err != nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":  "Activity and items created successfully",
		"activity": activity,
	})
}

// GetAllActivities - Controller สำหรับดึงข้อมูลกิจกรรมทั้งหมด
func GetAllActivities(c *fiber.Ctx) error {
	// เรียกใช้ service เพื่อดึงข้อมูลกิจกรรมทั้งหมด
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

// GetActivityByID - ดึงข้อมูลกิจกรรมตาม ID
func GetActivityByID(c *fiber.Ctx) error {
	id := c.Params("id")
	activityID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid ID format"})
	}

	activity, err := services.GetActivityByID(activityID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Activity not found"})
	}

	return c.Status(fiber.StatusOK).JSON(activity)
}

// UpdateActivity - อัพเดตข้อมูลกิจกรรม
func UpdateActivity(c *fiber.Ctx) error {
	id := c.Params("id")
	activityID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid ID format"})
	}

	var activity models.Activity
	if err := c.BodyParser(&activity); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	updatedActivity, err := services.UpdateActivity(activityID, activity)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(updatedActivity)
}

// DeleteActivity - ลบกิจกรรม
func DeleteActivity(c *fiber.Ctx) error {
	id := c.Params("id")
	activityID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid ID format"})
	}

	err = services.DeleteActivity(activityID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Activity deleted successfully"})
}
