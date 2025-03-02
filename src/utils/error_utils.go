// error_utils.go
package utils

import (
	"Backend-Bluelock-007/src/models"

	"github.com/gofiber/fiber/v2"
)

func HandleError(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(models.ErrorResponse{
		Status:  status,
		Message: message,
	})
}