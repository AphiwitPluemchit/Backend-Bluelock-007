package controllers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

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

// GetFormByid godoc
// @Summary      Get form by ID
// @Description  ดึงข้อมูลฟอร์มตามรหัส
// @Tags         forms
// @Produce      json
// @Param        id   path      string  true  "Form ID"
// @Success      200  {object}  models.Form
// @Failure      400  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Router       /forms/{id} [get]
// @Security     ApiKeyAuth
func GetFormByID(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ID is required",
		})
	}

	// ตั้ง timeout ให้คำขอครั้งนี้
	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	form, err := forms.GetFormByID(ctx, id)
	if err != nil {
		// ไม่พบเอกสาร
		if errors.Is(err, mongo.ErrNoDocuments) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Form not found",
			})
		}
		// id ไม่ถูกต้อง (บริการจะส่ง error นี้กลับมา)
		if errors.Is(err, forms.ErrInvalidObjectID) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid form id",
			})
		}
		// ข้อผิดพลาดอื่น ๆ
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get form",
		})
	}

	return c.JSON(form)
}

// UpdateForm godoc
// @Summary      Update a form
// @Description  อัปเดตข้อมูลฟอร์มตามรหัส
// @Tags         forms
// @Accept       json
// @Produce      json
// @Param        id    path      string      true  "Form ID"
// @Param        form  body      models.Form true  "Form object"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]interface{}
// @Failure      404   {object}  map[string]interface{}
// @Failure      500   {object}  map[string]interface{}
// @Router       /forms/{id} [patch]
// @Security     ApiKeyAuth
func UpdateForm(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ID is required"})
	}

	var form models.Form
	if err := c.BodyParser(&form); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	result, err := forms.UpdateForm(ctx, id, &form)
	if errors.Is(err, forms.ErrInvalidObjectID) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid form id"})
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update form"})
	}

	if result.MatchedCount == 0 { // ✅ ไม่พบเอกสาร
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Form not found"})
	}

	return c.JSON(fiber.Map{
		"message":    "Form updated successfully",
		"matched":    result.MatchedCount,
		"modified":   result.ModifiedCount,
	})
}



// DeleteFormByid godoc
// @Summary      Delete a form by ID
// @Description  ลบฟอร์มตาม ObjectID
// @Tags         forms
// @Param        id   path      string  true  "Form ID"
// @Success      200  {object}  map[string]interface{}  "Form deleted successfully"
// @Failure      400  {object}  map[string]interface{}  "Invalid ID"
// @Failure      404  {object}  map[string]interface{}  "Form not found"
// @Failure      500  {object}  map[string]interface{}  "Failed to delete form"
// @Router       /forms/{id} [delete]
// @Security     ApiKeyAuth
func DeleteFormByid(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "ID is required",
		})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := forms.DeleteFormByID(ctx, id)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete form",
		})
	}
	if result.DeletedCount == 0 {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "Form not found",
		})
	}
	return c.JSON(fiber.Map{
		"message":      "Form deleted successfully",
		"deletedCount": result.DeletedCount,
	})
}
