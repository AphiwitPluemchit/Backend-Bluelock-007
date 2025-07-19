package routes

import (
	"github.com/gofiber/fiber/v2"
)

func InitRoutes(router fiber.Router) {
	// Group API routes under /api
	// เรียกใช้ฟังก์ชัน InitUserRoutes และ InitOrderRoutes
	authRoutes(router)
	activityRoutes(router)
	adminRoutes(router)
	checkInOutRoutes(router)
	enrollmentRoutes(router)
	evaluationScoreRoutes(router)
	foodRoutes(router)
	formEvaluationRoutes(router)
	studentRoutes(router)
	suggestionRoutes(router)
	ocrRoutes(router)
	courseRoutes(router) // 👈 เพิ่มตรงนี้

	// Route เช็คว่า API ทำงานอยู่
	router.Get("/api", func(c *fiber.Ctx) error {
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
