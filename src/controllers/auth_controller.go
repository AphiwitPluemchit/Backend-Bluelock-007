package controllers

import (
	"Backend-Bluelock-007/src/services"
	"Backend-Bluelock-007/src/utils"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// LoginUser - สำหรับ login ทั้ง student และ admin
func LoginUser(c *fiber.Ctx) error {
	type LoginRequest struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	// 1. Input validation
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request format",
			"code":  "INVALID_REQUEST",
		})
	}

	// 2. Validate required fields
	if req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Email and password are required",
			"code":  "MISSING_CREDENTIALS",
		})
	}

	// 3. Rate limiting
	if services.IsRateLimited(req.Email) {
		// คำนวณเวลาที่เหลือ
		remainingTime := services.GetRemainingCooldownTime(req.Email)
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"error": fmt.Sprintf("Too many login attempts. Please try again in %d minutes and %d seconds.",
				int(remainingTime.Minutes()),
				int(remainingTime.Seconds())%60),
			"code":          "RATE_LIMITED",
			"remainingTime": int(remainingTime.Seconds()),
		})
	}

	// 4. Authenticate user
	user, err := services.AuthenticateUser(req.Email, req.Password)
	if err != nil {
		// 5. Log failed attempt
		services.LogLoginAttempt(req.Email, c.IP(), false)

		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid credentials",
			"code":  "INVALID_CREDENTIALS",
		})
	}

	// 6. Generate tokens
	token, err := utils.GenerateJWT(user.ID.Hex(), user.Email, user.Role)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Token generation failed",
			"code":  "TOKEN_ERROR",
		})
	}

	// 7. Log successful login
	services.LogLoginAttempt(req.Email, c.IP(), true)

	// 8. Set security headers
	c.Set("X-Frame-Options", "DENY")
	c.Set("X-Content-Type-Options", "nosniff")

	// 9. Return response
	return c.JSON(fiber.Map{
		"token":     token,
		"expiresIn": 3600,
		"user": fiber.Map{
			"id":          user.RefID.Hex(),
			"code":        user.Code,
			"name":        user.Name,
			"email":       user.Email,
			"role":        user.Role,
			"studentYear": user.StudentYear,
			"major":       user.Major,
			"lastLogin":   time.Now(),
		},
		"message": "Login successful",
	})
}

// LogoutUser - สำหรับ logout user
func LogoutUser(c *fiber.Ctx) error {
	// 1. Get user from JWT middleware context
	userID := c.Locals("userId").(string)
	if userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User not authenticated",
			"code":  "NOT_AUTHENTICATED",
		})
	}

	// 2. Get token for blacklisting
	token := c.Get("Authorization")
	if token != "" {
		token = strings.TrimPrefix(token, "Bearer ")
		// 3. Add to blacklist
		services.AddToBlacklist(token, userID)
	}

	// 4. Update user session
	services.UpdateLastLogout(userID)

	// 5. Log logout
	services.LogLogout(userID, c.IP(), time.Now())

	// 6. Return response
	return c.JSON(fiber.Map{
		"message":      "Logout successful",
		"success":      true,
		"timestamp":    time.Now(),
		"sessionEnded": true,
	})
}
