package models

// ErrorResponse ใช้สำหรับตอบกลับเมื่อเกิดข้อผิดพลาด
type ErrorResponse struct {
	Message string `json:"message" example:"Error description"`
}
