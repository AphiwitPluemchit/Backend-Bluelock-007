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

// LoginRequest represents the login request payload
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginUser godoc
// @Summary Login user
// @Description Authenticate user with email and password, includes rate limiting and security measures
// @Tags auth
// @Accept json
// @Produce json
// @Param loginRequest body LoginRequest true "Login credentials"
// @Success 200 {object} map[string]interface{} "Login successful with token and user info"
// @Failure 400 {object} map[string]interface{} "Bad request - invalid format or missing credentials"
// @Failure 401 {object} map[string]interface{} "Unauthorized - invalid credentials"
// @Failure 429 {object} map[string]interface{} "Too many requests - rate limited"
// @Failure 500 {object} map[string]interface{} "Internal server error - token generation failed"
// @Router /auth/login [post]
func LoginUser(c *fiber.Ctx) error {
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

	// 6. Generate JWT token (ใช้ RefID เป็น userID ใน JWT)
	token, err := utils.GenerateJWT(user.RefID.Hex(), user.Email, user.Role)
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

	// 9. Return response with user data
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

// GoogleLogin godoc
// @Summary Initiate Google OAuth login
// @Description Start Google OAuth authentication flow and return authorization URL
// @Tags auth
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "OAuth URL generated successfully"
// @Router /auth/google [get]
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

// GoogleCallback godoc
// @Summary Handle Google OAuth callback
// @Description Process Google OAuth callback, authenticate user, and redirect with token
// @Tags auth
// @Accept json
// @Produce json
// @Param code query string true "Authorization code from Google"
// @Param state query string false "State parameter for security"
// @Success 302 "Redirect to frontend with token"
// @Failure 302 "Redirect to frontend with error"
// @Router /auth/google/callback [get]
// GoogleCallback - handle Google OAuth callback
func GoogleCallback(c *fiber.Ctx) error {
	code := c.Query("code")
	errorParam := c.Query("error")
	frontendURL := os.Getenv("FRONTEND_URL")

	// 1. Handle OAuth error
	if errorParam != "" {
		return c.Redirect(fmt.Sprintf("%s/auth/callback?error=%s", frontendURL, errorParam))
	}

	// 2. Validate authorization code
	if code == "" {
		return c.Redirect(fmt.Sprintf("%s/auth/callback?error=missing_code", frontendURL))
	}

	// 3. Process Google login
	user, err := services.ProcessGoogleLogin(code)
	if err != nil {
		return c.Redirect(fmt.Sprintf("%s/auth/callback?error=%s", frontendURL, err.Error()))
	}

	// 4. Generate JWT token (ใช้ RefID เป็น userID ใน JWT)
	token, err := utils.GenerateJWT(user.RefID.Hex(), user.Email, user.Role)
	if err != nil {
		return c.Redirect(fmt.Sprintf("%s/auth/callback?error=token_generation_failed", frontendURL))
	}

	// 5. Log successful login
	services.LogLoginAttempt(user.Email, c.IP(), true)

	// 6. Redirect to frontend with token only (ใช้ /auth/me เพื่อดึงข้อมูล user)
	redirectURL := fmt.Sprintf("%s/auth/callback?token=%s", frontendURL, token)
	return c.Redirect(redirectURL)
}

// GetProfile godoc
// @Summary Get user profile
// @Description Get current user profile information from JWT token
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "User profile retrieved successfully"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 500 {object} map[string]interface{} "Failed to fetch profile"
// @Router /auth/me [get]
// GetProfile - ดึงข้อมูลโปรไฟล์ผู้ใช้
func GetProfile(c *fiber.Ctx) error {
	// 1. Get user info from JWT middleware context
	userID := c.Locals("userId").(string)
	role := c.Locals("role").(string)

	if userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User not authenticated",
			"code":  "NOT_AUTHENTICATED",
		})
	}

	// 2. Get full user profile from database
	user, err := services.GetUserProfile(userID, role)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to fetch profile: %v", err),
			"code":  "PROFILE_FETCH_ERROR",
		})
	}

	// 3. Return user profile
	return c.JSON(fiber.Map{
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
		"message": "Profile retrieved successfully",
	})
}

// LogoutUser godoc
// @Summary Logout user
// @Description Logout user by blacklisting token and updating session
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Logout successful"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Router /auth/logout [post]
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
