package email

import (
	"context"
	"log"
	"os"
	"strings"

	DB "Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"

	"github.com/hibiken/asynq"
)

func NotifyStudentsOnCompleted(
	programID, programName string,
	resolveProgram func(string) (*models.ProgramDto, error),
) {
	// ถ้ามี Redis → เข้าคิว
	if DB.AsynqClient != nil {
		task, _ := NewNotifyProgramCompletedTask(programID, programName)
		if _, err := DB.AsynqClient.Enqueue(task, asynq.TaskID("notify-completed-"+programID), asynq.MaxRetry(3)); err != nil {
			log.Println("❌ enqueue notify-completed task:", err)
		} else {
			log.Println("✅ Enqueued notify-completed task:", programID)
		}
		return
	}

	// ไม่มี Redis → ส่งทันที (sync)
	log.Println("⚠️ Redis not available → sending completed emails synchronously")
	sender, err := NewSMTPSenderFromEnv()
	if err != nil {
		log.Println("❌ init mail sender:", err)
		return
	}

	_ = strings.TrimRight(os.Getenv("FRONTEND_URL"), "/") // ใช้ใน handler แล้ว
	handler := HandleNotifyProgramCompleted(sender, resolveProgram)

	task, _ := NewNotifyProgramCompletedTask(programID, programName)
	if err := handler(context.Background(), task); err != nil {
		log.Printf("❌ send completed emails: %v", err)
	} else {
		log.Printf("✅ sent completed emails for program %s", programID)
	}
}
