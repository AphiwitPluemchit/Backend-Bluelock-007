package routes

import (
	"Backend-Bluelock-007/src/controllers"
	"Backend-Bluelock-007/src/middleware"

	"github.com/gofiber/fiber/v2"
)

// CheckInOutRoutes กำหนดเส้นทางสำหรับ CheckInOut API
func checkInOutRoutes(router fiber.Router) {
	checkInOutRoutes := router.Group("/checkInOuts")
	checkInOutRoutes.Use(middleware.AuthJWT)
	// checkInOutRoutes.Post("/generate-link", controllers.GenerateLink)
	// checkInOutRoutes.Post("/checkin/:uuid", controllers.Checkin)   // ดึงผู้ใช้ทั้งหมด
	// checkInOutRoutes.Post("/checkout/:uuid", controllers.Checkout) // ดึงผู้ใช้ทั้งหมด
	checkInOutRoutes.Get("/status", controllers.GetCheckinStatus)

	// --- QR Check-in System ---
	checkInOutRoutes.Post("/admin/qr-token", controllers.AdminCreateQRToken)
	checkInOutRoutes.Get("/student/qr/:token" /*middleware.AuthJWT,*/, controllers.StudentClaimQRToken) // add JWT middleware in main router
	checkInOutRoutes.Get("/student/validate/:token", controllers.StudentValidateQRToken)

	checkInOutRoutes.Post("/student/checkin" /*middleware.AuthJWT,*/, controllers.StudentCheckin)
	checkInOutRoutes.Post("/student/checkout" /*middleware.AuthJWT,*/, controllers.StudentCheckout)
	checkInOutRoutes.Get("/student/activity/:activityId/form", controllers.GetActivityForm)
}
