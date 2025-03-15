package routes

import (
	"github.com/gofiber/fiber/v2"
)

func InitRoutes(app *fiber.App) {
	// เรียกใช้ฟังก์ชัน InitUserRoutes และ InitOrderRoutes
	authRoutes(app)
	activityRoutes(app)
	adminRoutes(app)
	checkInOutRoutes(app)
	enrollmentRoutes(app)
	evaluationScoreRoutes(app)
	foodRoutes(app)
	formEvaluationRoutes(app)
	studentRoutes(app)
	suggestionRoutes(app)

	// Route เช็คว่า API ทำงานอยู่
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("✅ API is running...")
	})

}

// func JWTMiddleware() fiber.Handler {
// 	return func(c *fiber.Ctx) error {
// 		authHeader := c.Get("Authorization")
// 		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
// 			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Missing or invalid token"})
// 		}

// 		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
// 		claims, err := utils.ParseJWT(tokenStr)
// 		if err != nil {
// 			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token"})
// 		}

// 		// ⏩ บันทึกไว้ใช้ใน route ถัดไป
// 		c.Locals("userId", claims.UserID)
// 		c.Locals("role", claims.Role)
// 		c.Locals("email", claims.Email)

// 		return c.Next()
// 	}
// }
