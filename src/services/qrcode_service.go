package services

import (
	"Backend-Bluelock-007/src/qrcode"
	"fmt"
	"time"
)

const BaseURL = "http://dekdee3.informatics.buu.ac.th:8765/#/Student/Activity/MyActivityDetail"

// CreateCheckInQRCode - สร้าง QR Code สำหรับเช็คชื่อ
func CreateCheckInQRCode(activityID string) (string, error) {
	qrData := fmt.Sprintf("%s/%s", BaseURL, activityID) // ✅ ใส่ URL ที่ต้องการ
	fileName := fmt.Sprintf("checkin_%s_%d", activityID, time.Now().Unix())

	err := qrcode.GenerateQRCode(qrData, fileName)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("/public/qrcodes/%s.png", fileName), nil
}

// CreateCheckOutQRCode - สร้าง QR Code สำหรับเช็คชื่อออก
func CreateCheckOutQRCode(activityID string) (string, error) {
	qrData := fmt.Sprintf("%s/%s", BaseURL, activityID) // ✅ ใส่ URL ที่ต้องการ
	fileName := fmt.Sprintf("checkout_%s_%d", activityID, time.Now().Unix())

	err := qrcode.GenerateQRCode(qrData, fileName)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("/public/qrcodes/%s.png", fileName), nil
}
