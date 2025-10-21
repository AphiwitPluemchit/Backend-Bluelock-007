package programs

import (
	"os"
	"strings"

	"github.com/hibiken/asynq"
)

// RegisterProgramHandlers ลงทะเบียน Handler ทั้งหมดของ package programs
func RegisterProgramHandlers(mux *asynq.ServeMux) error {
	// ✅ 1) สร้าง sender จาก ENV
	sender, err := NewSMTPSenderFromEnv()
	if err != nil {
		return err // ถ้า SMTP env ยังไม่ครบ จะ fail ตอน start worker
	}

	// ✅ 2) อ่าน base URL จาก ENV และ sanitize
	base := os.Getenv("APP_BASE_URL")
	if base == "" {
		// fallback เป็น localhost เพื่อป้องกัน panic
		base = "http://localhost:9000"
	}
	base = strings.TrimRight(base, "/") // ป้องกัน // ซ้อนกัน

	// ✅ 3) สร้างฟังก์ชัน register-link builder (Frontend path)
	registerURL := func(programID string) string {
		// เช่น https://app.example.com/Student/Programs/64abc123...
		return base + "/Student/Programs/" + programID
	}

	// ✅ 4) ผูก handler กับ type ที่ใช้ใน task
	mux.HandleFunc(TypeNotifyOpenProgram, HandleNotifyOpenProgram(sender, registerURL))

	return nil
}
