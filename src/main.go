package main

import (
	"Backend-Bluelock-007/src/database"
	_ "Backend-Bluelock-007/src/docs"
	"Backend-Bluelock-007/src/routes"
	"log"

	"github.com/gofiber/fiber/v2"
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

	// เปิดใช้งาน Swagger ที่ URL /swagger
	app.Get("/swagger/*", swagger.HandlerDefault)

	// รวม routes จากแต่ละ module
	routes.InitRoutes(app)

	// เริ่มเซิร์ฟเวอร์
	log.Println("Server is running on port 8080...")
	log.Fatal(app.Listen(":8080"))
}
