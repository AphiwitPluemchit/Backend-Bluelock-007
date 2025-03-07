package main

import (
	_ "Backend-Bluelock-007/docs"
	"Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/routes"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/swagger"
)

func main() {

	// เชื่อมต่อกับ MongoDB
	err := database.ConnectMongoDB()
	if err != nil {
		log.Fatalf("Error connecting to the database: %v", err)
	}

	// สร้าง app instance
	app := fiber.New()

	// ✅ เปิดใช้งาน CORS Middleware
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "http://localhost:9000, https://your-frontend-domain.com", // ใส่โดเมนของ Frontend ที่อนุญาต
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowCredentials: true,
	}))

	// เปิดใช้งาน Swagger ที่ URL /swagger
	app.Get("/swagger/*", swagger.HandlerDefault)

	// รวม routes จากแต่ละ module
	routes.InitRoutes(app)

	// เริ่มเซิร์ฟเวอร์
	log.Println("Server is running on port 8080...")
	log.Fatal(app.Listen(":8080"))
}
