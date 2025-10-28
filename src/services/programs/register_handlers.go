// file: src/services/programs/register_handlers.go
package programs

import (
	"os"
	"strings"

	emailpkg "Backend-Bluelock-007/src/services/programs/email"
	"github.com/hibiken/asynq"
)

func RegisterProgramHandlers(mux *asynq.ServeMux) error {
	sender, err := emailpkg.NewSMTPSenderFromEnv()
	if err != nil {
		return err
	}

	base := strings.TrimRight(os.Getenv("FRONTEND_URL"), "/")
	if base == "" {
		base = "http://localhost:9000"
	}
	registerURL := func(programID string) string {
		return base + "/Student/Programs/" + programID
	}

	// ✅ แจ้งเปิดลงทะเบียน (open)
	mux.HandleFunc(
		emailpkg.TypeNotifyOpenProgram,
		emailpkg.HandleNotifyOpenProgram(
			sender,
			registerURL,
			GetProgramByID,
			GenerateStudentCodeFilter,
		),
	)

	// ✅ แจ้งเตือนก่อนเริ่ม 3 วัน (reminder)
	mux.HandleFunc(
		emailpkg.TypeNotifyProgramReminder,
		emailpkg.HandleProgramReminder(
			sender,
			GetProgramByID,
		),
	)
	// ✅ แจ้ง “กิจกรรมเสร็จสิ้น”
	mux.HandleFunc(
		emailpkg.TypeNotifyProgramCompleted,
		emailpkg.HandleNotifyProgramCompleted(
			sender,
			GetProgramByID,
		),
	)

	return nil
}
