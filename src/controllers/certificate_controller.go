package controllers

import (
	models "Backend-Bluelock-007/src/models"
	services "Backend-Bluelock-007/src/services/certificates"
	"fmt"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// @Summary      Verify a URL
// @Description  Verify a URL
// @Tags         certificates
// @Accept       json
// @Produce      json
// @Param        url        query     string  true  "URL to verify example: https://learner.thaimooc.ac.th/credential-wallet/10793bb5-6e4f-4873-9309-f25f216a46c7/sahaphap.rit/public"
// @Param        studentId  query     string  true  "Student ID example: 685abc586c4acf57c7e2f104 (สหภาพ)"
// @Param        courseId   query     string  true  "Course ID example: ThaiMooc: 6890a889ebc423e6aeb5605a or BuuMooc: 68b5c6b7e30cd42f34959a5e (การออกแบบและนำเสนอ)"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]interface{}
// @Failure      500   {object}  map[string]interface{}
// @Router       /certificates/url-verify [get]
func VerifyURL(c *fiber.Ctx) error {
	url := c.Query("url")
	studentId := c.Query("studentId")
	courseId := c.Query("courseId")

	fmt.Println("url", url)

	isVerified, isDuplicate, err := services.VerifyURL(url, studentId, courseId)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"isVerified":  isVerified,
		"isDuplicate": isDuplicate,
	})

}

// @Summary      Get Certificates
// @Description  Get Certificates
// @Tags         certificates
// @Accept       json
// @Produce      json
// @Param        page     query     int     false  "Page number"
// @Param        limit    query     int     false  "Limit per page"
// @Param        search   query     string  false  "Search query"
// @Param        sortBy   query     string  false  "Sort by field"
// @Param        order    query     string  false  "Sort order"
// @Param        studentId query     string  false  "Student ID"
// @Param        courseId  query     string  false  "Course ID"
// @Param        status   query     string  false  "Status"
// @Param        major    query     string  false  "Major"
// @Param        year     query     string  false  "Year"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]interface{}
// @Failure      500   {object}  map[string]interface{}
// @Router       /certificates [get]
func GetCertificates(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)
	search := c.Query("search", "")
	sortBy := c.Query("sortBy", "_id")
	order := c.Query("order", "desc")
	studentId := c.Query("studentId", "")
	courseId := c.Query("courseId", "")
	status := c.Query("status", "")
	// Support both ?status=pending,approved and repeated ?status[]=pending&status[]=approved
	// Support both ?major=AAI and repeated ?major[]=AAI&major[]=ITDI
	major := c.Query("major", "")
	year := c.Query("year", "")
	// Parse raw query string to get repeated params
	qs := string(c.Request().URI().QueryString())
	vals, _ := url.ParseQuery(qs)
	majorsArr := vals["major"]
	if len(majorsArr) == 0 {
		majorsArr = vals["major[]"]
	}
	if len(majorsArr) > 0 {
		major = strings.Join(majorsArr, ",")
	}

	// parse repeated status[] if present
	statusesArr := vals["status"]
	if len(statusesArr) == 0 {
		statusesArr = vals["status[]"]
	}
	if len(statusesArr) > 0 {
		status = strings.Join(statusesArr, ",")
	}

	// parse repeated year[] if present
	yearsArr := vals["year"]
	if len(yearsArr) == 0 {
		yearsArr = vals["year[]"]
	}
	if len(yearsArr) > 0 {
		year = strings.Join(yearsArr, ",")
	}

	fmt.Println("major", major)
	fmt.Println("year", year)
	fmt.Println("studentId", studentId)
	fmt.Println("courseId", courseId)
	fmt.Println("status", status)

	pagination := models.PaginationParams{
		Page:   page,
		Limit:  limit,
		Search: search,
		SortBy: sortBy,
		Order:  order,
	}
	uploadCertificateQuery := models.UploadCertificateQuery{
		StudentID: studentId,
		CourseID:  courseId,
		Status:    status,
		Major:     major,
		Year:      year,
	}

	certificates, paginationMeta, err := services.GetUploadCertificates(uploadCertificateQuery, pagination)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": certificates,
		"meta": paginationMeta,
	})
}

// UpdateCertificateStatusRequest ใช้สำหรับ request body ในการอัพเดทสถานะ certificate
type UpdateCertificateStatusRequest struct {
	Status models.StatusType `json:"status" example:"approved" enums:"pending,approved,rejected"`
	Remark string            `json:"remark" example:"Certificate verified by admin"`
}

// @Summary      Update Certificate Status
// @Description  Update the status of a certificate (Admin only). This will automatically handle hours calculation.
// @Tags         certificates
// @Accept       json
// @Produce      json
// @Param        id    path      string  true  "Certificate ID"
// @Param        body  body      UpdateCertificateStatusRequest  true  "Status update request"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]interface{}
// @Failure      404   {object}  map[string]interface{}
// @Failure      500   {object}  map[string]interface{}
// @Router       /certificates/{id}/status [put]
func UpdateCertificateStatus(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Certificate ID is required",
		})
	}

	var req UpdateCertificateStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate status
	if req.Status != models.StatusPending && req.Status != models.StatusApproved && req.Status != models.StatusRejected {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid status. Must be one of: pending, approved, rejected",
		})
	}

	updatedCert, err := services.UpdateUploadCertificateStatus(id, req.Status, req.Remark)
	if err != nil {
		if err.Error() == "upload certificate not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Certificate status updated successfully",
		"data":    updatedCert,
	})
}

// @Summary      Get Certificate by ID
// @Description  Get a single certificate by ID
// @Tags         certificates
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Certificate ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /certificates/{id} [get]
func GetCertificate(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Certificate ID is required",
		})
	}

	certificate, err := services.GetUploadCertificate(id)
	if err != nil {
		if err.Error() == "invalid upload certificate ID" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Certificate not found",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": certificate,
	})
}
