package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/src/services/forms"
)
// CreateForm godoc
// @Summary      Create a new form
// @Description  รับข้อมูลฟอร์มจาก client และบันทึกลงฐานข้อมูล
// @Tags         forms
// @Accept       json
// @Produce      json
// @Param        form  body  models.Form  true  "Form object"
// @Success      201   {object}  map[string]interface{}  "Form created successfully"
// @Failure      400   {object}  map[string]interface{}  "Invalid input"
// @Failure      500   {object}  map[string]interface{}  "Failed to insert form"
// @Router       /forms [post]
// @Security     ApiKeyAuth
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

	insertResult, err := services.InsetForm(ctx, &form)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to insert form",
		})
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"message":    "Form created successfully",
		"insertedId": insertResult.InsertedID,
		"form":       form,
	})
}
