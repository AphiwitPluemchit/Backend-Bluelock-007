package controllers

import (
	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/src/services/students"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// CreateStudent godoc
// @Summary Create students
// @Description Create one or more students
// @Tags students
// @Accept json
// @Produce json
// @Param students body []models.Student true "List of students to create"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Router /students [post]
// ✅ CreateStudent - เพิ่ม Student หลายคน
// ✅ CreateStudent - เพิ่ม Student หลายคน (แบบ validate ทั้งก้อนก่อน)
func CreateStudent(c *fiber.Ctx) error {
	var req []struct {
		Name      string `json:"name"`
		EngName   string `json:"engName"`
		Code      string `json:"code"`
		Major     string `json:"major"`
		SoftSkill int    `json:"softSkill"`
		HardSkill int    `json:"hardSkill"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input format"})
	}
	if len(req) == 0 {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Empty payload"})
	}

	// ---------- Normalize/Validate ----------
	clean := func(s string) string { return strings.TrimSpace(s) }
	var codes []string
	payloadDupMap := make(map[string]int)
	var invalid []string

	for i := range req {
		req[i].Code = clean(req[i].Code)
		req[i].Name = strings.TrimSpace(req[i].Name)
		req[i].EngName = strings.TrimSpace(req[i].EngName)

		// ตัวอย่าง validate code (เช่นต้องเป็นตัวเลขยาว >= 5)
		if req[i].Code == "" || len(req[i].Code) < 5 {
			invalid = append(invalid, req[i].Code)
			continue
		}
		codes = append(codes, req[i].Code)
		payloadDupMap[req[i].Code]++
	}

	// ---------- ตรวจซ้ำใน payload ----------
	var payloadDup []string
	for code, cnt := range payloadDupMap {
		if cnt > 1 {
			payloadDup = append(payloadDup, code)
		}
	}

	// ---------- ตรวจซ้ำใน DB ----------
	existCodes, err := students.FindExistingCodes(codes)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to check duplicates"})
	}

	// ถ้ามีซ้ำ (ใน payload หรือใน DB) หรือมี invalid → ไม่บันทึกใครเลย
	if len(payloadDup) > 0 || len(existCodes) > 0 || len(invalid) > 0 {
		return c.Status(http.StatusConflict).JSON(fiber.Map{
			"error":              "Duplicate or invalid codes found",
			"duplicateInPayload": payloadDup,
			"duplicateInDB":      existCodes,
			"invalid":            invalid,
		})
	}

	// ---------- เริ่มบันทึก (ผ่าน service ทีละคน หรือใช้ Transaction ก็ได้) ----------
	var failed []string
	for _, s := range req {
		stu := models.Student{
			Code:      s.Code,
			Name:      s.Name,
			EngName:   cleanName(s.EngName), // ✅ ใช้ EngName ที่มาจากฟอร์ม ไม่ใช่ Name
			Status:    calculateStatus(s.SoftSkill, s.HardSkill),
			SoftSkill: s.SoftSkill,
			HardSkill: s.HardSkill,
			Major:     mapMajor(s.Major),
		}
		usr := models.User{
			Email:    strings.ToLower(s.Code + "@go.buu.ac.th"),
			Password: s.Code + "ABC",
		}
		if err := students.CreateStudent(&usr, &stu); err != nil {
			log.Println("❌ Failed to create student:", s.Code, err)
			failed = append(failed, s.Code)
		}
	}

	if len(failed) > 0 {
		return c.Status(http.StatusConflict).JSON(fiber.Map{
			"error":  "Failed to create some students",
			"failed": failed,
		})
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"message": "Students created successfully",
	})
}

func cleanList(arr []string) []string {
	var result []string
	for _, v := range arr {
		v = strings.TrimSpace(v)
		if v != "" {
			result = append(result, v)
		}
	}
	return result
}

// GetStudents godoc
// @Summary Get students
// @Description Get all students with optional filters
// @Tags students
// @Accept json
// @Produce json
// @Param page query int false "Page number"
// @Param limit query int false "Page size"
// @Param search query string false "Search keyword"
// @Param sortBy query string false "Sort by field"
// @Param order query string false "Order (asc/desc)"
// @Param studentStatus query string false "Student status (comma separated)"
// @Param major query string false "Major (comma separated)"
// @Param studentYear query string false "Student year (comma separated)"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /students [get]
func GetStudents(c *fiber.Ctx) error {
	params := models.DefaultPagination()
	params.Page, _ = strconv.Atoi(c.Query("page", strconv.Itoa(params.Page)))
	params.Limit, _ = strconv.Atoi(c.Query("limit", strconv.Itoa(params.Limit)))
	params.Search = c.Query("search", params.Search)
	params.SortBy = c.Query("sortBy", params.SortBy)
	params.Order = c.Query("order", params.Order)

	studentStatus := cleanList(strings.Split(c.Query("studentStatus"), ","))
	majors := cleanList(strings.Split(c.Query("major"), ","))
	studentYears := cleanList(strings.Split(c.Query("studentYear"), ","))
	studentCode := cleanList(strings.Split(c.Query("studentCode"), ","))
	log.Println("studentStatus", studentStatus)
	log.Println("majors", majors)
	log.Println("studentYears", studentYears)
	log.Println("studentCode", studentCode)
	students, total, totalPages, err := students.GetStudentsWithFilter(params, majors, studentYears, studentStatus)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error fetching students"})
	}

	return c.JSON(fiber.Map{
		"data": students,
		"meta": fiber.Map{
			"page":       params.Page,
			"limit":      params.Limit,
			"total":      total,
			"totalPages": totalPages,
		},
	})
}

// GetStudentByCode godoc
// @Summary Get student by code
// @Description Get a student by their code
// @Tags students
// @Accept json
// @Produce json
// @Param code path string true "Student code"
// @Success 200 {object} models.Student
// @Failure 404 {object} map[string]interface{}
// @Router /students/{code} [get]
func GetStudentByCode(c *fiber.Ctx) error {
	code := c.Params("code")
	student, err := students.GetStudentByCode(code)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Student not found",
		})
	}

	return c.JSON(student)
}

// UpdateStudent godoc
// @Summary Update student
// @Description Update a student's information
// @Tags students
// @Accept json
// @Produce json
// @Param id path string true "Student ID"
// @Param student body models.Student true "Student data"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /students/{id} [put]
func UpdateStudent(c *fiber.Ctx) error {
	id := c.Params("id")

	// ✅ struct แยก สำหรับรับค่าจาก frontend
	var req struct {
		Name      string `json:"name"`
		EngName   string `json:"engName"`
		Code      string `json:"code"`
		Major     string `json:"major"`
		SoftSkill int    `json:"softSkill"`
		HardSkill int    `json:"hardSkill"`
		Email     string `json:"email"` // ✅ เพิ่ม email
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	// ✅ map เข้า model.Student
	student := &models.Student{
		Name:      req.Name,
		EngName:   req.EngName,
		Code:      req.Code,
		Major:     req.Major,
		SoftSkill: req.SoftSkill,
		HardSkill: req.HardSkill,
	}

	// ✅ ส่งทั้ง student และ email แยกไป
	if err := students.UpdateStudent(id, student, req.Email); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error updating student",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Student updated successfully",
	})
}

// DeleteStudent godoc
// @Summary Delete student
// @Description Delete a student by ID
// @Tags students
// @Accept json
// @Produce json
// @Param id path string true "Student ID"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /students/{id} [delete]
func DeleteStudent(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := students.DeleteStudent(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error deleting student",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Student deleted successfully",
	})
}

// UpdateStudentStatusByIDs - อัปเดตสถานะนักเรียนหลายคนโดยใช้ ID
func UpdateStudentStatusByIDs(c *fiber.Ctx) error {
	type UpdateStatusRequest struct {
		StudentIDs []string `json:"studentIds"`
		Status     int      `json:"status"`
	}

	var req UpdateStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request format",
			"code":  "INVALID_REQUEST",
		})
	}

	if len(req.StudentIDs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Student IDs are required",
			"code":  "MISSING_IDS",
		})
	}

	err := students.UpdateStudentStatusByIDs(req.StudentIDs, req.Status)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update student status",
			"code":  "UPDATE_FAILED",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Student status updated successfully",
		"updated": len(req.StudentIDs),
		"success": true,
	})
}

func GetSammaryByCode(c *fiber.Ctx) error {
	code := c.Params("code")
	student, err := students.GetSammaryByCode(code)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Student not found"})
	}
	return c.JSON(student)
}

func GetSammaryByCodeWithHourHistory(c *fiber.Ctx) error {
	code := c.Params("code")
	student, err := students.GetSammaryByCodeWithHourHistory(code)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Student not found"})
	}
	return c.JSON(student)
}
func GetSammaryAll(c *fiber.Ctx) error {
	majors := cleanList(strings.Split(c.Query("major"), ","))
	studentYears := cleanList(strings.Split(c.Query("studentYear"), ","))
	summary, err := students.GetStudentSummary(majors, studentYears)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error generating summary"})
	}
	return c.JSON(summary)
}

//	func UpdateStudentStatus(c *fiber.Ctx) error {
//		id := c.Params("id")
//		err := students.UpdateStudentStatus(id)
//		if err != nil {
//			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Error updating student status"})
//		}
//		return c.JSON(fiber.Map{"message": "Student status updated successfully"})
//	}
func calculateStatus(softSkill, hardSkill int) int {
	total := softSkill + hardSkill

	switch {
	case softSkill >= 30 && hardSkill >= 12:
		return 3 // ครบ
	case total >= 20:
		return 2 // น้อย
	default:
		return 1 // น้อยมาก
	}
}
func mapMajor(fullName string) string {
	switch fullName {
	case "ปัญญาประดิษฐ์ประยุกต์และเทคโนโลยีอัจฉริยะ":
		return "AAI"
	case "วิศวกรรมซอฟต์แวร์":
		return "SE"
	case "วิทยาการคอมพิวเตอร์":
		return "CS"
	case "เทคโนโลยีสารสนเทศเพื่ออุตสาหกรรมดิจิทัล":
		return "ITDI"
	default:
		return "ไม่พบสาขา" // ถ้าไม่ตรง จะเก็บค่าที่ส่งมาเดิม
	}
}
func cleanName(name string) string {
	upper := strings.ToUpper(strings.TrimSpace(name))

	// ตัด MR. / MISS ออก
	upper = strings.TrimPrefix(upper, "MR. ")
	upper = strings.TrimPrefix(upper, "MISS ")

	return strings.TrimSpace(upper)
}
func generatePassword(code, engName string) string {
	engName = strings.ToUpper(strings.TrimSpace(engName))
	// ดึงมา 3 ตัวแรก
	prefix := ""
	if len(engName) >= 3 {
		prefix = engName[:3]
	} else {
		prefix = engName // ถ้าชื่อสั้นกว่า 3 ตัว
	}

	return code + prefix
}

// graph
// "total":{
// 	"total": 195,
// 	"completed": 114,
// 	"notCompleted": 81,
// 	"completionRate": 58,
// 	"softSkill":
// 	{
//     "completed": 143,
//     "notCompleted": 52,
//     "progress": 73
//     },
// 	"hardSkill":

//     "completed": 141,
//     "notCompleted": 54,
//     "progress": 72
// 	}}
// "CS":{
// 	"total": 195,
// 	"completed": 114,
// 	"notCompleted": 81,
// 	"completionRate": 58,
// 	"softSkill":
// 	{
//     "completed": 143,
//     "notCompleted": 52,
//     "progress": 73
// 	},
// 	"hardSkill":
// 	{
//     "completed": 141,
//     "notCompleted": 54,
//     "progress": 72
// 	}}
// "AAI":{}
// "SE":{}
// "ITDI":{
// 	"total": 195,
// 	"completed": 114,
// 	"notCompleted": 81,
// 	"completionRate": 58,
// 	"softSkill":
// 	{
//     "completed": 143,
//     "notCompleted": 52,
//     "progress": 73
// 	},
// 	"hardSkill": {
//     "completed": 141,
//     "notCompleted": 54,
//     "progress": 72
// }}
