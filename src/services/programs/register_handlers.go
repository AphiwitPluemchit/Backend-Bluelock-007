// file: src/services/programs/register_handlers.go
package programs

import (
	"os"
	"strings"

	emailpkg "Backend-Bluelock-007/src/services/programs/email" // ✅ ใช้แพ็กเกจ email ที่ย้ายแล้ว
	"github.com/hibiken/asynq"
)

func RegisterProgramHandlers(mux *asynq.ServeMux) error {
	// ✅ เตรียม sender จาก .env
	sender, err := emailpkg.NewSMTPSenderFromEnv()
	if err != nil {
		return err
	}

	// ✅ สร้างฟังก์ชันทำลิงก์ (มี default ถ้า .env ไม่มี)
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
			GetProgramByID,           // resolver จากแพ็กเกจปัจจุบัน (programs)
			GenerateStudentCodeFilter, // สร้าง code prefix
		),
	)

	// ✅ แจ้งเตือนก่อนเริ่ม 3 วัน (reminder)
	mux.HandleFunc(
		emailpkg.TypeNotifyProgramReminder, // ← ต้องอ้างผ่าน emailpkg
		emailpkg.HandleProgramReminder(     // ← ต้องอ้างผ่าน emailpkg
			sender,
			GetProgramByID,
		),
	)

	return nil
}
