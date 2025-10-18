package controllers

import (
	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/src/services/courses"
	"Backend-Bluelock-007/src/utils"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var courseImagePath = "./uploads/course/images/"

// CreateCourse godoc
// @Summary      Create a new course
// @Description  Create a new course
// @Tags         courses
// @Accept       json
// @Produce      json
// @Param        body body models.Course true "Course object"
// @Success      201  {object}  models.Course
// @Failure      400  {object}  models.ErrorResponse
// @Failure      500  {object}  models.ErrorResponse
// @Router       /courses [post]
func CreateCourse(c *fiber.Ctx) error {
	var request models.Course
	if err := c.BodyParser(&request); err != nil {
		return utils.HandleError(c, fiber.StatusBadRequest, "Invalid input: "+err.Error())
	}
	course, err := courses.CreateCourse(&request)
	if err != nil {
		return utils.HandleError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.Status(fiber.StatusCreated).JSON(course)
}

// GetAllCourses godoc
// @Summary      Get all courses with pagination and filtering
// @Description  Get all courses with pagination and filtering options
// @Tags         courses
// @Produce      json
// @Param        query  query     models.PaginationParams true "Pagination and filtering parameters"
// @Param        filters query     models.CourseFilters true "Filtering parameters"
// @Success      200  {object}  models.CoursePaginatedResponse
// @Failure      500  {object}  models.ErrorResponse
// @Router       /courses [get]
func GetAllCourses(c *fiber.Ctx) error {
	params := models.DefaultPagination()
	if err := c.QueryParser(&params); err != nil {
		return utils.HandleError(c, fiber.StatusBadRequest, "Invalid query parameters")
	}

	var filters models.CourseFilters
	if err := c.QueryParser(&filters); err != nil {
		return utils.HandleError(c, fiber.StatusBadRequest, "Invalid filter parameters")
	}

	result, total, err := courses.GetAllCourses(params, filters)
	if err != nil {
		return utils.HandleError(c, fiber.StatusInternalServerError, err.Error())
	}

	totalPages := int(math.Ceil(float64(total) / float64(params.Limit)))

	response := models.CoursePaginatedResponse{
		Data: result,
		Meta: models.PaginationMeta{
			Page:       params.Page,
			Limit:      params.Limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}
	return c.Status(fiber.StatusOK).JSON(response)
}

// GetCourseByID godoc
// @Summary      Get a course by ID
// @Description  Get a course by ID
// @Tags         courses
// @Produce      json
// @Param        id   path  string  true  "Course ID"
// @Success      200  {object}  models.Course
// @Failure      404  {object}  models.ErrorResponse
// @Failure      500  {object}  models.ErrorResponse
// @Router       /courses/{id} [get]
func GetCourseByID(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return utils.HandleError(c, fiber.StatusBadRequest, "Invalid ID")
	}
	course, err := courses.GetCourseByID(id)
	if err != nil {
		return utils.HandleError(c, fiber.StatusNotFound, err.Error())
	}
	return c.JSON(course)
}

// UpdateCourse godoc
// @Summary      Update a course
// @Description  Update a course
// @Tags         courses
// @Accept       json
// @Produce      json
// @Param        id     path  string        true  "Course ID"
// @Param        body   body  models.Course true  "Course object"
// @Success      200  {object}  models.Course
// @Failure      400  {object}  models.ErrorResponse
// @Failure      500  {object}  models.ErrorResponse
// @Router       /courses/{id} [put]
func UpdateCourse(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return utils.HandleError(c, fiber.StatusBadRequest, "Invalid ID")
	}
	var request models.Course
	if err := c.BodyParser(&request); err != nil {
		return utils.HandleError(c, fiber.StatusBadRequest, "Invalid input: "+err.Error())
	}
	updated, err := courses.UpdateCourse(id, request)
	if err != nil {
		return utils.HandleError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(updated)
}

// DeleteCourse godoc
// @Summary      Delete a course
// @Description  Delete a course
// @Tags         courses
// @Produce      json
// @Param        id   path  string  true  "Course ID"
// @Success      200
// @Failure      400  {object}  models.ErrorResponse
// @Failure      500  {object}  models.ErrorResponse
// @Router       /courses/{id} [delete]
func DeleteCourse(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return utils.HandleError(c, fiber.StatusBadRequest, "Invalid ID")
	}
	err = courses.DeleteCourse(id)
	if err != nil {
		return utils.HandleError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.SendStatus(fiber.StatusOK)
}

// UploadCourseImage godoc
// @Summary      Upload an image for a course
// @Description  Upload an image for a course. If filename is provided, the old file will be deleted.
// @Tags         courses
// @Accept       multipart/form-data
// @Produce      json
// @Param        id path string true "Course ID"
// @Param        filename query string false "Old file name to be replaced"
// @Param        file formData file true "Image file"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  models.ErrorResponse
// @Failure      500  {object}  models.ErrorResponse
// @Router       /courses/{id}/image [post]
func UploadCourseImage(c *fiber.Ctx) error {
	id := c.Params("id")
	oldFileName := c.Query("filename")

	// Get uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		return utils.HandleError(c, fiber.StatusBadRequest, "Failed to upload file: "+err.Error())
	}

	// Validate file type (optional - add image type validation)
	ext := filepath.Ext(file.Filename)
	allowedExts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true}
	if !allowedExts[ext] {
		return utils.HandleError(c, fiber.StatusBadRequest, "Invalid file type. Only image files are allowed (jpg, jpeg, png, gif, webp)")
	}

	// Delete old file if exists
	if oldFileName != "" {
		oldFilePath := courseImagePath + oldFileName
		if err := os.Remove(oldFilePath); err != nil {
			log.Println("Warning: Failed to remove old file:", err)
			// Continue anyway - don't fail the upload if old file deletion fails
		}
	}

	// Generate new unique filename
	newFileName := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	filePath := filepath.Join(courseImagePath, newFileName)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(courseImagePath, 0755); err != nil {
		return utils.HandleError(c, fiber.StatusInternalServerError, "Failed to create directory: "+err.Error())
	}

	// Save file to disk
	if err := c.SaveFile(file, filePath); err != nil {
		return utils.HandleError(c, fiber.StatusInternalServerError, "Failed to save file: "+err.Error())
	}

	// Update MongoDB with new image path
	err = courses.UploadCourseImage(id, newFileName)
	if err != nil {
		// Rollback: Delete uploaded file if database update fails
		if removeErr := os.Remove(filePath); removeErr != nil {
			log.Println("Failed to remove uploaded file after DB error:", removeErr)
		}
		return utils.HandleError(c, fiber.StatusInternalServerError, "Failed to update database: "+err.Error())
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":  "Image uploaded successfully",
		"fileName": newFileName,
		"path":     filePath,
	})
}

// DeleteCourseImage godoc
// @Summary      Delete an image for a course
// @Description  Delete an image for a course and remove the file from disk
// @Tags         courses
// @Produce      json
// @Param        id path string true "Course ID"
// @Param        filename query string true "File name to delete"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  models.ErrorResponse
// @Failure      500  {object}  models.ErrorResponse
// @Router       /courses/{id}/image [delete]
func DeleteCourseImage(c *fiber.Ctx) error {
	id := c.Params("id")
	fileName := c.Query("filename")

	if fileName == "" {
		return utils.HandleError(c, fiber.StatusBadRequest, "Filename is required")
	}

	// Delete file from disk
	filePath := courseImagePath + fileName
	if err := os.Remove(filePath); err != nil {
		log.Println("Warning: Failed to remove file:", err)
		// Continue anyway - maybe file doesn't exist
	}

	// Update MongoDB to clear image path
	err := courses.UploadCourseImage(id, "")
	if err != nil {
		return utils.HandleError(c, fiber.StatusInternalServerError, "Failed to update database: "+err.Error())
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Image deleted successfully",
	})
}
