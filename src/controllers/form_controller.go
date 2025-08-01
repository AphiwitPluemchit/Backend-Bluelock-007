package controllers

import (
	"strconv"

	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/src/services/forms"
	"Backend-Bluelock-007/src/utils"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CreateForm handles POST /forms
func CreateForm(ctx *fiber.Ctx) error {
	var req models.CreateFormRequest

	if err := ctx.BodyParser(&req); err != nil {
		return utils.SendErrorResponse(ctx, "Invalid request body", 400)
	}

	// Validate request
	if err := utils.ValidateStruct(req); err != nil {
		return utils.SendErrorResponse(ctx, err.Error(), 400)
	}

	// Create form
	result, err := forms.CreateForm(ctx.Context(), &req)
	if err != nil {
		return utils.SendErrorResponse(ctx, err.Error(), 500)
	}

	return utils.SendSuccessResponse(ctx, "Form created successfully", result)
}

// DeleteForm handles DELETE /forms/:id
func DeleteForm(ctx *fiber.Ctx) error {
	formIDStr := ctx.Params("id")

	formID, err := primitive.ObjectIDFromHex(formIDStr)
	if err != nil {
		return utils.SendErrorResponse(ctx, "Invalid form ID", 400)
	}

	// Call service to delete form
	err = forms.DeleteForm(ctx.Context(), formID)
	if err != nil {
		if err.Error() == "form not found" {
			return utils.SendErrorResponse(ctx, "Form not found", 404)
		}
		return utils.SendErrorResponse(ctx, err.Error(), 500)
	}

	return utils.SendSuccessResponse(ctx, "Form deleted successfully", nil)
}

// GetForms handles GET /forms
func GetForms(ctx *fiber.Ctx) error {
	// Parse pagination parameters
	pageStr := ctx.Query("page", "1")
	limitStr := ctx.Query("limit", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 10
	}

	// Get forms
	result, err := forms.GetForms(ctx.Context(), page, limit)
	if err != nil {
		return utils.SendErrorResponse(ctx, err.Error(), 500)
	}

	return utils.SendSuccessResponse(ctx, "Forms retrieved successfully", result)
}

// GetFormByID handles GET /forms/:id
func GetFormByID(ctx *fiber.Ctx) error {
	formIDStr := ctx.Params("id")

	formID, err := primitive.ObjectIDFromHex(formIDStr)
	if err != nil {
		return utils.SendErrorResponse(ctx, "Invalid form ID", 400)
	}

	// Get form
	result, err := forms.GetFormByID(ctx.Context(), formID)
	if err != nil {
		if err.Error() == "form not found" {
			return utils.SendErrorResponse(ctx, "Form not found", 404)
		}
		return utils.SendErrorResponse(ctx, err.Error(), 500)
	}

	return utils.SendSuccessResponse(ctx, "Form retrieved successfully", result)
}

// SubmitForm handles POST /forms/:id/submissions
func SubmitForm(ctx *fiber.Ctx) error {
	formIDStr := ctx.Params("id")

	formID, err := primitive.ObjectIDFromHex(formIDStr)
	if err != nil {
		return utils.SendErrorResponse(ctx, "Invalid form ID", 400)
	}

	var req models.SubmitFormRequest

	if err := ctx.BodyParser(&req); err != nil {
		return utils.SendErrorResponse(ctx, "Invalid request body", 400)
	}

	// Validate request
	if err := utils.ValidateStruct(req); err != nil {
		return utils.SendErrorResponse(ctx, err.Error(), 400)
	}

	// Submit form
	result, err := forms.SubmitForm(ctx.Context(), formID, &req)
	if err != nil {
		if err.Error() == "form not found" {
			return utils.SendErrorResponse(ctx, "Form not found", 404)
		}
		return utils.SendErrorResponse(ctx, err.Error(), 400)
	}

	return utils.SendSuccessResponse(ctx, "Form submitted successfully", result)
}

// GetFormSubmissions handles GET /forms/:id/submissions
func GetFormSubmissions(ctx *fiber.Ctx) error {
	formIDStr := ctx.Params("id")

	formID, err := primitive.ObjectIDFromHex(formIDStr)
	if err != nil {
		return utils.SendErrorResponse(ctx, "Invalid form ID", 400)
	}

	// Parse pagination parameters
	pageStr := ctx.Query("page", "1")
	limitStr := ctx.Query("limit", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 10
	}

	// Get submissions
	result, err := forms.GetFormSubmissions(ctx.Context(), formID, page, limit)
	if err != nil {
		if err.Error() == "form not found" {
			return utils.SendErrorResponse(ctx, "Form not found", 404)
		}
		return utils.SendErrorResponse(ctx, err.Error(), 500)
	}

	return utils.SendSuccessResponse(ctx, "Submissions retrieved successfully", result)
}
