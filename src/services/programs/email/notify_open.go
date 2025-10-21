package email

import (
	"context"
	"log"
	"os"
	"strings"

	"github.com/hibiken/asynq"
	DB "Backend-Bluelock-007/src/database"
	"Backend-Bluelock-007/src/models"
)

func NotifyStudentsOnOpen(
	programID, programName string,
	resolveProgram func(string) (*models.ProgramDto, error),
	codePrefixFn func([]int) []string,
) {
	// มี Redis → เข้าคิว
	if DB.AsynqClient != nil {
		task, _ := NewNotifyOpenProgramTask(programID, programName)
		if _, err := DB.AsynqClient.Enqueue(task, asynq.TaskID("notify-open-"+programID), asynq.MaxRetry(3)); err != nil {
			log.Println("❌ enqueue notify-open task:", err)
		} else {
			log.Println("✅ Enqueued notify-open task:", programID)
		}
		return
	}

	// ไม่มี Redis → ส่งทันที
	log.Println("⚠️ Redis not available → sending open-notify emails synchronously")
	sender, err := NewSMTPSenderFromEnv()
	if err != nil {
		log.Println("❌ init mail sender:", err)
		return
	}

	base := strings.TrimRight(os.Getenv("APP_BASE_URL"), "/")
	if base == "" {
		base = "http://localhost:9000"
	}

	handler := HandleNotifyOpenProgram(
		sender,
		func(pid string) string { return base + "/Student/Programs/" + pid },
		resolveProgram,
		codePrefixFn,
	)

	task, _ := NewNotifyOpenProgramTask(programID, programName)
	if err := handler(context.Background(), task); err != nil {
		log.Printf("❌ send emails: %v", err)
	} else {
		log.Printf("✅ sent open-notify emails for program %s", programID)
	}
}
