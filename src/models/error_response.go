package models

// ErrorResponse โครงสร้างมาตรฐานสำหรับการส่ง Error
type ErrorResponse struct {
	Status  int    `json:"status"`  // HTTP Status Code
	Message string `json:"message"` // รายละเอียดของ Error
}

// ErrorResponse โครงสร้างมาตรฐานสำหรับการส่ง Error 