package main

import (
	_ "Backend-Bluelock-007/docs"
	"Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/jobs"
	"Backend-Bluelock-007/src/routes"
	"Backend-Bluelock-007/src/services"
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/swagger"
	"github.com/hibiken/asynq"
)

func main() {

	// get url from .env
	appURI := os.Getenv("APP_URI")
	if appURI == "" {
		appURI = "8888" // ใช้ 8888 เป็นค่าเริ่มต้น
	}
	redisURI := os.Getenv("REDIS_URI")
	// if redisURI == "" {
	// 	redisURI = "localhost:6379" // ใช้ localhost:6379 เป็นค่าเริ่มต้น
	// }
	origins := os.Getenv("ALLOWED_ORIGINS") // ✅ เปิดใช้งาน CORS Middleware
	if origins == "" {
		origins = "*"
	}

	// เชื่อมต่อกับ MongoDB
	err := database.ConnectMongoDB()
	if err != nil {
		log.Fatalf("Error connecting to the database: %v", err)
	}

	// สร้าง app instance
	app := fiber.New()

	app.Use(cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowCredentials: false, // ❌ ต้องเป็น false ถ้าใช้ "*"
	}))

	// ✅ สร้าง Asynq Client และเริ่มรัน Asynq Worker
	if redisURI != "" {
		services.AsynqClient = asynq.NewClient(asynq.RedisClientOpt{Addr: redisURI})

		go func() {
			srv := asynq.NewServer(
				asynq.RedisClientOpt{Addr: redisURI},
				asynq.Config{
					Concurrency: 10, // รันพร้อมกันได้ 10 task
				},
			)
			mux := asynq.NewServeMux()
			mux.HandleFunc(jobs.TypeCloseActivity, jobs.HandleCloseActivityTask)

			if err := srv.Run(mux); err != nil {
				log.Fatal("❌ Failed to start Asynq worker:", err)
			} else {
				log.Println("🚀 Asynq Worker is starting...")
			}
		}()
	}

	// เปิดใช้งาน Swagger ที่ URL /swagger
	app.Get("/swagger/*", swagger.HandlerDefault)

	// รวม routes จากแต่ละ module
	routes.InitRoutes(app)

	// ✅ ให้บริการไฟล์ใน uploads/activity/images/
	app.Static("/uploads/activity/images", "./uploads/activity/images")

	// เริ่มเซิร์ฟเวอร์
	log.Println("Server is running on port " + appURI)
	err = app.Listen(fmt.Sprintf(":%s", url.PathEscape(appURI)))
	if err != nil {
		log.Fatal(err)
	}

}
