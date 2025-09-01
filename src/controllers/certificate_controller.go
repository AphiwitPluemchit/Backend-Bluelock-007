package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"time"

	"Backend-Bluelock-007/src/models"
	services "Backend-Bluelock-007/src/services/certificates"
	"Backend-Bluelock-007/src/services/courses"
	"Backend-Bluelock-007/src/services/students"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// @Summary      Upload a file
// @Description  Upload a file
// @Tags         certificate
// @Accept       multipart/form-data
// @Produce      json
// @Param        file        formData  file    true   "File to upload"
// @Param        studentId   query     string  false  "Student ID"
// @Param        courseId    query     string  false  "Course ID"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]interface{}
// @Failure      500   {object}  map[string]interface{}
// @Router       /certificate/upload [post]
func UploadHandler(c *fiber.Ctx) error {
	fmt.Println(" [Fiber] ได้รับการอัปโหลดไฟล์")
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

	// choose display name based on course format without mutating original struct
	displayName := student.Name
	if !course.IsThaiFormat {
		displayName = student.EngName
	}

	// Prepare to send to FastAPI OCR
	fastApiURL := fastAPIURL()

	responseData, err := sendFileToFastAPI(fileHeader, displayName, course.Name, course.Type, fastApiURL)
	if err != nil {
		log.Printf(" OCR proxy error: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "OCR failed",
		})
	}

	// extract data from response with type assertions
	isNameMatch, isCourseMatch, url := extractOCRFlags(responseData)

	uploadCertificate := models.UploadCertificate{
		StudentId:     stId,
		CourseId:      crId,
		Url:           url,
		Status:        models.Pending,
		IsNameMatch:   isNameMatch,
		IsCourseMatch: isCourseMatch,
		UploadAt:      time.Now(),
	}

	// check if url is duplicate
	isDuplicate, err := services.IsVerifiedDuplicate(c.Context(), url)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error checking duplicate",
		})
	}
	if isDuplicate {
		fmt.Println("Duplicate URL")
		uploadCertificate.IsDuplicate = true
		uploadCertificate.Remark = "Upload ไปแล้ว หรือ URL ซ้ำ"
	}

	// create upload certificate
	result, err := services.CreateUploadCertificate(&uploadCertificate)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error creating upload certificate",
		})
	}

	// if isNameMatch or isCourseMatch is false
	if isNameMatch && isCourseMatch {

	} else {

		// save file to local storage use id for filename
		err = saveFile(fileHeader, "./uploads/certificates/fail/"+result.InsertedID.(primitive.ObjectID).Hex()+".pdf")
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Error saving file",
			})
		}

	}

	// Upload Success
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Certificate Uploaded, Please wait for 2 days to check the result",
	})
}

// Resolve FastAPI OCR URL with env override
func fastAPIURL() string {
	if v := os.Getenv("FASTAPI_URL"); v != "" {
		return v
	}
	return "http://fastapi-ocr:8000/ocr"
}

// Extract OCR flags from FastAPI response map
func extractOCRFlags(responseData map[string]interface{}) (bool, bool, string) {
	var isNameMatch bool
	var isCourseMatch bool
	var url string

	if dataRaw, ok := responseData["data"]; ok {
		switch v := dataRaw.(type) {
		case map[string]interface{}:
			if b, ok := v["isNameMatch"].(bool); ok {
				isNameMatch = b
			}
			if b, ok := v["isCourseMatch"].(bool); ok {
				isCourseMatch = b
			}
			if s, ok := v["url"].(string); ok {
				url = s
			}
		case string:
			// Non-JSON response from FastAPI; keep defaults (false) and log for visibility
			log.Printf(" [Fiber] FastAPI returned non-JSON data: %s\n", v)
		default:
			// Unknown structure; keep defaults
			log.Printf(" [Fiber] Unexpected data type from FastAPI: %T\n", v)
		}
	} else {
		// Fallback: FastAPI may return fields at the top-level instead of under "data"
		if b, ok := responseData["isNameMatch"].(bool); ok {
			isNameMatch = b
		}
		if b, ok := responseData["isCourseMatch"].(bool); ok {
			isCourseMatch = b
		}
		if s, ok := responseData["url"].(string); ok {
			url = s
		}

		if url == "" && !isNameMatch && !isCourseMatch {
			log.Printf(" [Fiber] 'data' field missing and no top-level keys present in FastAPI response: %+v\n", responseData)
		}
	}

	fmt.Println("isNameMatch: ", isNameMatch)
	fmt.Println("isCourseMatch: ", isCourseMatch)
	fmt.Println("url: ", url)

	return isNameMatch, isCourseMatch, url
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

	// Determine and set accurate Content-Type for the file part
	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		ext := filepath.Ext(fileHeader.Filename)
		contentType = mime.TypeByExtension(ext)
		if contentType == "" {
			if strings.EqualFold(ext, ".pdf") {
				contentType = "application/pdf"
			} else {
				contentType = "application/octet-stream"
			}
		}
	}

	// Create a part with explicit headers so FastAPI sees the correct content type
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, "file", fileHeader.Filename))
	h.Set("Content-Type", contentType)
	part, err := writer.CreatePart(h)
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

	// Try to decode JSON response from FastAPI
	var parsed interface{}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		// If not JSON, return raw text under "data"
		result := make(map[string]interface{})
		result["status_code"] = resp.StatusCode
		result["data"] = string(respBody)
		return result, nil
	}

	// ✅ คลี่ "data" ออกมาตั้งแต่ตรงนี้
	result := make(map[string]interface{})
	result["status_code"] = resp.StatusCode

	if m, ok := parsed.(map[string]interface{}); ok {
		// FastAPI: { "status": "success", "data": {...} }
		if inner, ok := m["data"]; ok {
			result["data"] = inner // << ใส่เฉพาะ data ด้านใน
		} else {
			// กันเผื่อ API เปลี่ยนโครง: เอาทั้งก้อน
			result["data"] = m
		}
	} else {
		// กันพลาด: ไม่ใช่ object
		result["data"] = parsed
	}

	return result, nil
}

// @Summary      Verify a URL
// @Description  Verify a URL
// @Tags         certificate
// @Accept       json
// @Produce      json
// @Param        url        query     string  true  "URL to verify example: https://learner.thaimooc.ac.th/credential-wallet/10793bb5-6e4f-4873-9309-f25f216a46c7/sahaphap.rit/public"
// @Param        studentId  query     string  true  "Student ID example: 685abc586c4acf57c7e2f104 (สหภาพ)"
// @Param        courseId   query     string  true  "Course ID example: ThaiMooc: 6890a889ebc423e6aeb5605a or BuuMooc: 68b5c6b7e30cd42f34959a5e (การออกแบบและนำเสนอ)"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]interface{}
// @Failure      500   {object}  map[string]interface{}
// @Router       /certificate/url-verify [get]
func VerifyURL(c *fiber.Ctx) error {
	url := c.Query("url")
	studentId := c.Query("studentId")
	courseId := c.Query("courseId")

	isVerified, err := services.VerifyURL(url, studentId, courseId)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"isVerified": isVerified,
	})

}
