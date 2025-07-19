package routes

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"

	"github.com/gofiber/fiber/v2"
)

func ocrRoutes(router fiber.Router) {
	// Upload route
	router.Post("/ocr/upload", uploadHandler)

	// Approve route
	router.Post("/ocr/approve", approveHandler)
}

// uploadHandler จัดการการอัปโหลดไฟล์

func uploadHandler(c *fiber.Ctx) error {
	fmt.Println("📥 [Fiber] ได้รับการอัปโหลดไฟล์")
	// รับไฟล์จาก FormData field name: "file"
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No file uploaded",
		})
	}

	// Debug log
	log.Printf("📥 [Fiber] ได้รับไฟล์: %s\n", fileHeader.Filename)

	// Create directory if it does not exist
	if err := os.MkdirAll("./uploads/certificates", 0755); err != nil {
		log.Println("Failed to create directory:", err)
		// You may want to return an error here instead of continuing
	}

	// Save temp file
	filePath := fmt.Sprintf("./uploads/certificates/%s", fileHeader.Filename)
	if err := saveFile(fileHeader, filePath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save file",
		})
	}
	log.Printf("🛠 เซฟไฟล์ไปที่: %s\n", filePath)

	// Prepare to send to FastAPI OCR
	fastApiURL := "http://fastapi-ocr:8000/ocr"
	responseData, err := sendFileToFastAPI(filePath, fastApiURL)
	if err != nil {
		log.Printf("❌ OCR proxy error: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "OCR failed",
		})
	}

	// Success
	return c.JSON(responseData)
}

func approveHandler(c *fiber.Ctx) error {
	type ApprovePayload struct {
		StudentName string `json:"student_name"`
		CourseName  string `json:"course_name"`
		Date        string `json:"date"`
	}

	var payload ApprovePayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid payload",
		})
	}

	// Debug log
	log.Println("📥 รับข้อมูลอนุมัติจาก frontend:")
	log.Printf("%+v\n", payload)

	// ตอบกลับ
	return c.JSON(fiber.Map{
		"status": "approved",
	})
}

// ---------- Helper Functions ----------

// Save uploaded file to local storage
func saveFile(fileHeader *multipart.FileHeader, savePath string) error {
	src, err := fileHeader.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(savePath)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}

// Send file to FastAPI OCR
func sendFileToFastAPI(filePath, url string) (map[string]interface{}, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// ใช้ bytes.Buffer แทน fiber.Buffer
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filePath)
	if err != nil {
		return nil, err
	}

	if _, err := io.Copy(part, file); err != nil {
		return nil, err
	}

	writer.Close()

	// สร้าง request
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// อ่าน response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Return
	result := make(map[string]interface{})
	result["status_code"] = resp.StatusCode
	result["data"] = string(respBody)

	return result, nil
}
