package controllers

import (
	"log"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/src/services/submission"
)

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

// createSubmission
func CreateSubmission(c *fiber.Ctx) error {
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

	submissions := models.Submission{
		FormID:    formOID,
		UserID:    userOID,
		Responses: resps,
	}

	log.Printf("[submission] IN form=%s user=%s responses=%d", submissions.FormID.Hex(), submissions.UserID.Hex(), len(submissions.Responses))

	created, err := submission.CreateSubmission(c.Context(), &submissions)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(created)
}

// GetSubmission handles getting a submission by ID
func GetSubmission(c *fiber.Ctx) error {
	id, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid ID format"})
	}

	subm, err := submission.GetSubmissionByID(c.Context(), id)
if err != nil {
    return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Submission not found"})
}
return c.JSON(subm)
}


func GetSubmissionsByForm(c *fiber.Ctx) error {
  formID, err := primitive.ObjectIDFromHex(c.Params("formId"))
  if err != nil {
    return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid form ID"})
  }

  // ⬇️ รับ query เพิ่มเติม (optional)
  limit := int64(0)
  if v := c.Query("limit"); v != "" {
    if n, convErr := strconv.ParseInt(v, 10, 64); convErr == nil && n > 0 {
      limit = n
    }
  }
  sortField := c.Query("sort") // e.g. "createdAt" หรือ "-createdAt"

  submissions, err := submission.GetSubmissionsByFormID(c.Context(), formID, limit, sortField)
  if err != nil {
    return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
  }
  return c.JSON(submissions)
}

// DeleteSubmission handles submission deletion
func DeleteSubmission(c *fiber.Ctx) error {
	id, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid ID format"})
	}

	if err := submission.DeleteSubmission(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
// ===== Analytics ภายใต้ submission controller =====
func GetFormBlocksAnalytics(c *fiber.Ctx) error {
	formID, err := primitive.ObjectIDFromHex(c.Params("formId"))
	if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid form ID"})
	}
	items, err := submission.GetFormBlocksAnalytics(c.Context(), formID)
	if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(items)
}

func GetBlockAnalytics(c *fiber.Ctx) error {
	formID, err := primitive.ObjectIDFromHex(c.Params("formId"))
	if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid form ID"})
	}
	blockID, err := primitive.ObjectIDFromHex(c.Params("blockId"))
	if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid block ID"})
	}
	items, err := submission.GetBlockAnalytics(c.Context(), formID, blockID)
	if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(items)
}