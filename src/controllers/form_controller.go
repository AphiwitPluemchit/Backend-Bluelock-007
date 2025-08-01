package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"yourapp/models"
	"yourapp/services"
)

// CreateForm รับข้อมูล form จาก client และบันทึกลงฐานข้อมูล
func CreateForm(c *fiber.Ctx) error {
	var form models.Form

	if err := c.BodyParser(&form); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	// ตั้งค่า _id และเวลาสร้าง
	form.ID = primitive.NewObjectID()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := services.InsertForm(ctx, &form); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to insert form",
		})
	}

	return c.Status(http.StatusCreated).JSON(form)
}
