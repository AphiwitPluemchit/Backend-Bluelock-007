package controllers

import (
	services "Backend-Bluelock-007/src/services/certificates"

	"github.com/gofiber/fiber/v2"
)

// @Summary      Verify a URL
// @Description  Verify a URL
// @Tags         certificates
// @Accept       json
// @Produce      json
// @Param        url        query     string  true  "URL to verify example: https://learner.thaimooc.ac.th/credential-wallet/10793bb5-6e4f-4873-9309-f25f216a46c7/sahaphap.rit/public"
// @Param        studentId  query     string  true  "Student ID example: 685abc586c4acf57c7e2f104 (สหภาพ)"
// @Param        courseId   query     string  true  "Course ID example: ThaiMooc: 6890a889ebc423e6aeb5605a or BuuMooc: 68b5c6b7e30cd42f34959a5e (การออกแบบและนำเสนอ)"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]interface{}
// @Failure      500   {object}  map[string]interface{}
// @Router       /certificates/url-verify [get]
func VerifyURL(c *fiber.Ctx) error {
	url := c.Query("url")
	studentId := c.Query("studentId")
	courseId := c.Query("courseId")

	isVerified, isDuplicate, err := services.VerifyURL(url, studentId, courseId)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"isVerified":  isVerified,
		"isDuplicate": isDuplicate,
	})

}
