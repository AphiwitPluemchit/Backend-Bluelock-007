package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"Backend-Bluelock-007/src/models"
 	forms "Backend-Bluelock-007/src/services/forms"
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

	// สร้าง Form ID
	form.ID = primitive.NewObjectID()

	// Loop ทุก block → set ID + formId
	for i := range form.Blocks {
		form.Blocks[i].ID = primitive.NewObjectID()
		form.Blocks[i].FormID = form.ID

		// Loop ทุก choice → set ID + blockId
		for j := range form.Blocks[i].Choices {
			form.Blocks[i].Choices[j].ID = primitive.NewObjectID()
			form.Blocks[i].Choices[j].BlockID = form.Blocks[i].ID
		}

		// Loop ทุก row → set ID + blockId
		for j := range form.Blocks[i].Rows {
			form.Blocks[i].Rows[j].ID = primitive.NewObjectID()
			form.Blocks[i].Rows[j].BlockID = form.Blocks[i].ID
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	insertResult, err := forms.InsetForm(ctx, &form)
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

// GetAllForms godoc
// @Summary      Get all forms
// @Description  ดึงข้อมูลฟอร์มทั้งหมด
// @Tags         forms
// @Produce      json
// @Success      200   {array}   models.Form
// @Failure      500   {object}  map[string]interface{}
// @Router       /forms [get]
// @Security     ApiKeyAuth
func GetAllForms(c *fiber.Ctx) error {
	allForms, err := forms.GetAllForms(context.Background())
	if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to get forms",
			})
	}
	return c.JSON(allForms)
}