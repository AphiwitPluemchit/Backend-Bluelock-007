package controllers

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"Backend-Bluelock-007/src/models"
	submissionService "Backend-Bluelock-007/src/services/submission"
)

type SubmissionController struct {
	service *submissionService.SubmissionService
}

func NewSubmissionController(service *submissionService.SubmissionService) *SubmissionController {
	return &SubmissionController{service: service}
}

// ===== DTO ที่รับจาก Frontend เป็น string IDs =====
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
//createSubmission
func (sc *SubmissionController) CreateSubmission(c *fiber.Ctx) error {
	var in submissionIn
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input: " + err.Error()})
	}

	// แปลง string → ObjectID
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

	submission := models.Submission{
		FormID:    formOID,
		UserID:    userOID,
		Responses: resps,
	}

	log.Printf("[submission] IN form=%s user=%s responses=%d", submission.FormID.Hex(), submission.UserID.Hex(), len(submission.Responses))

	created, err := sc.service.CreateSubmission(c.Context(), &submission)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(created)
}

// GetSubmission handles getting a submission by ID
func (sc *SubmissionController) GetSubmission(c *fiber.Ctx) error {
	id, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid ID format"})
	}

	submission, err := sc.service.GetSubmissionByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Submission not found"})
	}

	return c.JSON(submission)
}

// GetSubmissionsByForm handles getting submissions by form ID
func (sc *SubmissionController) GetSubmissionsByForm(c *fiber.Ctx) error {
	formID, err := primitive.ObjectIDFromHex(c.Params("formId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid form ID"})
	}

	submissions, err := sc.service.GetSubmissionsByFormID(c.Context(), formID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(submissions)
}

// DeleteSubmission handles submission deletion
func (sc *SubmissionController) DeleteSubmission(c *fiber.Ctx) error {
	id, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid ID format"})
	}

	if err := sc.service.DeleteSubmission(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
