package controllers

import (
	"log"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"Backend-Bluelock-007/src/models"
	submissionSvc "Backend-Bluelock-007/src/services/submission"
)

// --------- Input DTOs ---------

type responseIn struct {
	ID         string  `json:"id,omitempty"`
	AnswerText *string `json:"answerText,omitempty"`
	BlockID    string  `json:"blockId"`
	ChoiceID   *string `json:"choiceId,omitempty"`
	RowID      *string `json:"rowId,omitempty"`
}

type submissionIn struct {
	FormID    string       `json:"formId"`
	UserID    string       `json:"userId"`
	Responses []responseIn `json:"responses"`
}

// --------- Create ---------

func CreateSubmission(c *fiber.Ctx) error {
	var in submissionIn
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input: " + err.Error()})
	}

	// validate & convert IDs
	formOID, err := primitive.ObjectIDFromHex(in.FormID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid formId"})
	}
	userOID, err := primitive.ObjectIDFromHex(in.UserID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid userId"})
	}

	resps := make([]models.Response, 0, len(in.Responses))
	for _, r := range in.Responses {
		blockOID, err := primitive.ObjectIDFromHex(r.BlockID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid blockId"})
		}

		var choiceOID *primitive.ObjectID
		if r.ChoiceID != nil && *r.ChoiceID != "" {
			tmp, err := primitive.ObjectIDFromHex(*r.ChoiceID)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid choiceId"})
			}
			choiceOID = &tmp
		}

		var rowOID *primitive.ObjectID
		if r.RowID != nil && *r.RowID != "" {
			tmp, err := primitive.ObjectIDFromHex(*r.RowID)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid rowId"})
			}
			rowOID = &tmp
		}

		resps = append(resps, models.Response{
			ID:         primitive.NewObjectID(),
			AnswerText: r.AnswerText,
			BlockID:    blockOID,
			ChoiceID:   choiceOID,
			RowID:      rowOID,
		})
	}

	submissions := models.Submission{
		FormID:    formOID,
		UserID:    userOID,
		Responses: resps,
	}

	log.Printf("[submission] IN form=%s user=%s responses=%d",
		submissions.FormID.Hex(), submissions.UserID.Hex(), len(submissions.Responses))

	created, err := submissionSvc.CreateSubmission(c.Context(), &submissions)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(created)
}

// --------- Read (by id) ---------

func GetSubmission(c *fiber.Ctx) error {
	id, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid ID format"})
	}

	submission, err := submissionSvc.GetSubmissionByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Submission not found"})
	}

	return c.JSON(submission)
}

// --------- Read (by form, legacy path) ---------

func GetSubmissionsByForm(c *fiber.Ctx) error {
	formID, err := primitive.ObjectIDFromHex(c.Params("formId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid form ID"})
	}

	submissions, err := submissionSvc.GetSubmissionsByFormID(c.Context(), formID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(submissions)
}

// --------- Read (by form with query) ---------
// ใช้กับ: GET /forms/:formId/submissions?limit=1&sort=latest
func GetSubmissionsByFormWithQuery(c *fiber.Ctx) error {
	formID, err := primitive.ObjectIDFromHex(c.Params("formId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid form ID"})
	}

	limit := 0
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}
	sortParam := c.Query("sort") // e.g., "latest"

	subs, err := submissionSvc.GetSubmissionsByFormIDWithQuery(c.Context(), formID, limit, sortParam)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(subs)
}

// ========== Analytics รวมไว้ในไฟล์นี้ได้เลย ==========

// GET /analytics/forms/:formId/blocks/:blockId
// ใช้ทำกราฟแท่งของบล็อกเดียว (single-choice หรือ grid)
func GetBlockAnalytics(c *fiber.Ctx) error {
	formOID, err := primitive.ObjectIDFromHex(c.Params("formId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid formId"})
	}
	blockOID, err := primitive.ObjectIDFromHex(c.Params("blockId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid blockId"})
	}

	items, err := submissionSvc.AggregateBlockCounts(c.Context(), formOID, blockOID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(items)
}

// GET /analytics/forms/:formId/blocks
// รวมทุกบล็อกในฟอร์ม (เรียกทีเดียว)
func GetFormBlocksAnalytics(c *fiber.Ctx) error {
	formOID, err := primitive.ObjectIDFromHex(c.Params("formId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid formId"})
	}
	items, err := submissionSvc.AggregateFormCounts(c.Context(), formOID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(items)
}

// --------- (ทางเลือก) endpoint สั้น ๆ ดึง latest ของฟอร์มเดียว ---------
// GET /forms/:formId/submissions/latest
func GetLatestSubmission(c *fiber.Ctx) error {
	formID, err := primitive.ObjectIDFromHex(c.Params("formId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid form ID"})
	}

	subs, err := submissionSvc.GetSubmissionsByFormIDWithQuery(c.Context(), formID, 1, "latest")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if len(subs) == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "No submissions"})
	}
	return c.JSON(subs[0])
}
