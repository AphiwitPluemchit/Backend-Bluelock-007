package controllers

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"

	"Backend-Bluelock-007/src/services/courses"
	"Backend-Bluelock-007/src/services/students"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// @Summary      Upload a file
// @Description  Upload a file
// @Tags         ocr
// @Accept       multipart/form-data
// @Produce      json
// @Param        file        formData  file    true   "File to upload"
// @Param        studentId   query     string  false  "Student ID"
// @Param        courseId    query     string  false  "Course ID"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]interface{}
// @Failure      500   {object}  map[string]interface{}
// @Router       /ocr/upload [post]
func UploadHandler(c *fiber.Ctx) error {
	fmt.Println("📥 [Fiber] ได้รับการอัปโหลดไฟล์")
	// รับไฟล์จาก FormData field name: "file"
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No file uploaded",
		})
	}
	studentId := c.Query("studentId")
	courseId := c.Query("courseId")

	stId, err := primitive.ObjectIDFromHex(studentId)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid student ID",
		})
	}
	crId, err := primitive.ObjectIDFromHex(courseId)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid course ID",
		})
	}

	// get student name and course name
	student, err := students.GetStudentById(stId)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error fetching student",
		})
	}
	course, err := courses.GetCourseByID(crId)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error fetching course",
		})
	}

	// Debug log
	log.Printf("📥 [Fiber] ได้รับไฟล์: %s\n", fileHeader.Filename)

	// Prepare to send to FastAPI OCR
	fastApiURL := os.Getenv("FASTAPI_URL")
	if fastApiURL == "" {
		fastApiURL = "http://fastapi-ocr:8000/ocr"
	}
	fmt.Println("FastAPI URL: " + fastApiURL)
	responseData, err := sendFileToFastAPI(fileHeader, student.Name, course.Name, course.Type, fastApiURL)
	if err != nil {
		log.Printf("❌ OCR proxy error: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "OCR failed",
		})
	}

	// Debug log
	log.Printf("📤 [Fiber] ส่งไฟล์ไปยัง FastAPI OCR: %s\n", fileHeader.Filename)
	log.Printf("� [Fiber] ได้รับผลลัพธ์จาก FastAPI OCR: %+v\n", responseData)

	// Success
	return c.JSON(responseData)
}

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
func sendFileToFastAPI(fileHeader *multipart.FileHeader, studentName string, courseName string, courseType string, url string) (map[string]interface{}, error) {
	// เปิดไฟล์จาก Multipart โดยตรง แทนการเปิดจากพาธบนดิสก์
	src, err := fileHeader.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()

	// ใช้ bytes.Buffer แทน fiber.Buffer
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", fileHeader.Filename)
	if err != nil {
		return nil, err
	}

	if _, err := io.Copy(part, src); err != nil {
		return nil, err
	}

	// Add other form fields
	_ = writer.WriteField("studentName", studentName)
	_ = writer.WriteField("courseName", courseName)
	_ = writer.WriteField("courseType", courseType)

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
