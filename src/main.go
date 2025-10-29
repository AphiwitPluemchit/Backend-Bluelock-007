package main

import (
	_ "Backend-Bluelock-007/docs"
	"Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/jobs"
	"Backend-Bluelock-007/src/routes"
	"Backend-Bluelock-007/src/services/programs" // 👈 ผูก email handlers ที่นี่
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/swagger"
	"github.com/hibiken/asynq"
)

func main() {
	// ใช้ Asia/Bangkok เป็น default timezone ของโปรเซส
	if loc, err := time.LoadLocation("Asia/Bangkok"); err == nil {
		time.Local = loc
		log.Println("✅ Set process timezone to Asia/Bangkok (time.Local)")
	} else {
		log.Println("⚠️ Failed to load Asia/Bangkok location, using system default")
	}

	// ---- App / CORS ----
	appURI := os.Getenv("APP_URI")
	if appURI == "" {
		appURI = "8888"
	}
	origins := os.Getenv("ALLOWED_ORIGINS")
	if origins == "" {
		origins = "*"
	}

	// ---- MongoDB ----
	if err := database.ConnectMongoDB(); err != nil {
		log.Fatalf("Error connecting to MongoDB: %v", err)
	}
	log.Println("✅ MongoDB connected")

	// ---- Redis & Asynq ----
	database.InitRedis() // sets database.RedisURI and database.RedisClient (if ok)
	database.InitAsynq() // sets database.AsynqClient (if Redis ok)

	// ---- Start Asynq Worker (background goroutine) ----
	if database.AsynqClient != nil && database.RedisURI != "" {
		go func() {
			srv := asynq.NewServer(
				asynq.RedisClientOpt{Addr: database.RedisURI},
				asynq.Config{
					// ปรับตามโหลดงานจริง
					Concurrency: 10,
					Queues: map[string]int{
						// ใช้คิว default ทั้งหมด
						"default": 1,
					},
				},
			)

			mux := asynq.NewServeMux()

			// ✅ งานเปลี่ยนสถานะโปรแกรม (ของเดิม)
			mux.HandleFunc(jobs.TypeCompleteProgram, jobs.HandleCompleteProgramTask)
			mux.HandleFunc(jobs.TypeCloseEnroll, jobs.HandleCloseEnrollTask)

			// ✅ งานอีเมล: เปิดลงทะเบียน / แจ้งเตือนก่อนเริ่ม 3 วัน
			// (ภายในจะผูก handler: programs/email.TypeNotifyOpenProgram, programs/email.TypeNotifyProgramReminder)
			
			if err := programs.RegisterProgramHandlers(mux); err != nil {
				log.Println("⚠️ RegisterProgramHandlers error:", err)
			}
			log.Println("🚀 Asynq Worker is starting on Redis:", database.RedisURI)
			if err := srv.Run(mux); err != nil {
				log.Println("⚠️ Asynq worker stopped:", err)
			}
		}()
	} else {
		log.Println("⚠️ Asynq worker will not start. Background jobs disabled (no Redis).")
	}

	// ---- Fiber App ----
	app := fiber.New()

	app.Use(cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Requested-With, X-Skip-Loading, X-Skip-Auth-Redirect",
		AllowCredentials: false,
		ExposeHeaders:    "Content-Length, Content-Type",
		MaxAge:           300,
	}))

	// Routes
	routes.InitRoutes(app)

	// Swagger
	app.Get("/swagger/*", swagger.HandlerDefault)

	// Static uploads
	app.Static("/uploads", "./uploads")

	// ---- Start HTTP Server ----
	log.Println("🚀 Server is running on port", appURI)
	if err := app.Listen(fmt.Sprintf(":%s", url.PathEscape(appURI))); err != nil {
		log.Fatal(err)
	}
}
