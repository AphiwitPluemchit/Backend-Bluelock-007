package qrcode

import (
	"fmt"

	"github.com/skip2/go-qrcode"
)

// GenerateQRCode สร้าง QR Code จากข้อมูลที่กำหนด และบันทึกเป็นไฟล์ PNG
func GenerateQRCode(data string, filename string) error {
	filePath := fmt.Sprintf("public/qrcodes/%s.png", filename) // เก็บไฟล์ในโฟลเดอร์ public
	err := qrcode.WriteFile(data, qrcode.Medium, 256, filePath)
	if err != nil {
		return err
	}
	return nil
}
