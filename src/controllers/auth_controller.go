package controllers

import (
	"Backend-Bluelock-007/src/services"
	"Backend-Bluelock-007/src/utils"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/oauth2"
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

// GoogleLogin - เริ่มต้น Google OAuth flow
func GoogleLogin(c *fiber.Ctx) error {
	config := services.GetGoogleOAuthConfig()

	// Generate state parameter for security
	state := utils.GenerateRandomString(32)

	// Store state in session or cache (for production, use Redis)
	// For now, we'll skip state validation for simplicity

	url := config.AuthCodeURL(state, oauth2.AccessTypeOffline)

	return c.JSON(fiber.Map{
		"url": url,
	})
}

// GoogleCallback - handle Google OAuth callback
func GoogleCallback(c *fiber.Ctx) error {
	fmt.Printf("🔍 Google Callback received - IP: %s\n", c.IP())
	fmt.Printf("🔍 Query params: %s\n", c.Request().URI().QueryString())

	code := c.Query("code")
	state := c.Query("state")
	errorParam := c.Query("error")

	fmt.Printf("🔍 Code: %s\n", code)
	fmt.Printf("🔍 State: %s\n", state)
	fmt.Printf("🔍 Error: %s\n", errorParam)

	if errorParam != "" {
		fmt.Printf("❌ Google OAuth error: %s\n", errorParam)
		frontendURL := os.Getenv("FRONTEND_URL")
		redirectURL := fmt.Sprintf("%s/auth/callback?error=%s", frontendURL, errorParam)
		return c.Redirect(redirectURL)
	}

	if code == "" {
		fmt.Printf("❌ No authorization code provided\n")
		frontendURL := os.Getenv("FRONTEND_URL")
		redirectURL := fmt.Sprintf("%s/auth/callback?error=missing_code", frontendURL)
		return c.Redirect(redirectURL)
	}

	fmt.Printf("🔄 Processing Google login...\n")
	// Process Google login
	user, err := services.ProcessGoogleLogin(code)
	if err != nil {
		fmt.Printf("❌ Google login failed: %v\n", err)
		frontendURL := os.Getenv("FRONTEND_URL")
		redirectURL := fmt.Sprintf("%s/auth/callback?error=%s", frontendURL, err.Error())
		return c.Redirect(redirectURL)
	}

	fmt.Printf("✅ User authenticated: %s (%s)\n", user.Email, user.Role)

	// Generate JWT token
	token, err := utils.GenerateJWT(user.ID.Hex(), user.Email, user.Role)
	if err != nil {
		fmt.Printf("❌ Token generation failed: %v\n", err)
		frontendURL := os.Getenv("FRONTEND_URL")
		redirectURL := fmt.Sprintf("%s/auth/callback?error=token_generation_failed", frontendURL)
		return c.Redirect(redirectURL)
	}

	fmt.Printf("✅ JWT token generated successfully\n")

	// Log successful login
	services.LogLoginAttempt(user.Email, c.IP(), true)

	// Redirect to frontend with token
	frontendURL := os.Getenv("FRONTEND_URL")
	redirectURL := fmt.Sprintf("%s/auth/callback?token=%s", frontendURL, token)

	fmt.Printf("🔄 Redirecting to: %s\n", redirectURL)
	return c.Redirect(redirectURL)
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
