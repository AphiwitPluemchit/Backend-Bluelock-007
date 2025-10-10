package main

import (
	_ "Backend-Bluelock-007/docs"
	"Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/jobs"
	"Backend-Bluelock-007/src/routes"
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/swagger"
	"github.com/hibiken/asynq"
)

// @title Bluelock API
// @version 1.0
// @description This is the API documentation for Bluelock project.
// @host
// @BasePath

func main() {

	// get url from .env
	appURI := os.Getenv("APP_URI")
	if appURI == "" {
		appURI = "8888" // ใช้ 8888 เป็นค่าเริ่มต้น
	}

	origins := os.Getenv("ALLOWED_ORIGINS") // ✅ เปิดใช้งาน CORS Middleware
	if origins == "" {
		origins = "*"
	}

	// เชื่อมต่อกับ MongoDB
	err := database.ConnectMongoDB()
	if err != nil {
		log.Fatalf("Error connecting to the database: %v", err)
	}

	fmt.Println("🚀 Server is starting...", origins)

	// ✅ สร้าง Redis Client สําหรับการเชื่อมต่อ ทำ Redis Cache
	database.InitRedis()
	// ✅ สร้าง Asynq Client และเริ่มรัน Asynq Worker
	if database.RedisURI != "" {
		database.AsynqClient = asynq.NewClient(asynq.RedisClientOpt{Addr: database.RedisURI})

		go func() {
			srv := asynq.NewServer(
				asynq.RedisClientOpt{Addr: database.RedisURI},
				asynq.Config{
					Concurrency: 10, // รันพร้อมกันได้ 10 task
				},
			)
			mux := asynq.NewServeMux()
			mux.HandleFunc(jobs.TypeCompleteProgram, jobs.HandleCompleteProgramTask)
			mux.HandleFunc(jobs.TypeCloseEnroll, jobs.HandleCloseEnrollTask)

			if err := srv.Run(mux); err != nil {
				log.Fatal("❌ Failed to start Asynq worker:", err)
			} else {
				log.Println("🚀 Asynq Worker is starting...")
			}
		}()

	}

	// สร้าง app instance
	app := fiber.New()

	app.Use(cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Requested-With, X-Skip-Loading, X-Skip-Auth-Redirect",
		AllowCredentials: false, // ❌ ต้องเป็น false ถ้าใช้ "*"
		ExposeHeaders:    "Content-Length, Content-Type",
		MaxAge:           300, // Cache preflight requests for 5 minutes
	}))

	// Group API routes under /api
	routes.InitRoutes(app)

	// เปิดใช้งาน Swagger ที่ URL /swagger
	app.Get("/swagger/*", swagger.HandlerDefault)

	// ✅ ให้บริการไฟล์ใน uploads/program/images/
	app.Static("/uploads/program/images", "./uploads/program/images")

	// เริ่มเซิร์ฟเวอร์
	log.Println("Server is running on port " + appURI)
	err = app.Listen(fmt.Sprintf(":%s", url.PathEscape(appURI)))
	if err != nil {
		log.Fatal(err)
	}

}
