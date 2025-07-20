// error_utils.go
package utils

import (
	"Backend-Bluelock-007/src/models"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

var validate = validator.New()

func HandleError(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(models.ErrorResponse{
		Status:  status,
		Message: message,
	})
}

// SendErrorResponse sends an error response
func SendErrorResponse(c *fiber.Ctx, message string, status int) error {
	return c.Status(status).JSON(models.ErrorResponse{
		Status:  status,
		Message: message,
	})
}

// SendSuccessResponse sends a success response with data
func SendSuccessResponse(c *fiber.Ctx, message string, data interface{}) error {
	return c.Status(200).JSON(fiber.Map{
		"status":  200,
		"message": message,
		"data":    data,
	})
}

// ValidateStruct validates a struct using validator
func ValidateStruct(s interface{}) error {
	return validate.Struct(s)
}
