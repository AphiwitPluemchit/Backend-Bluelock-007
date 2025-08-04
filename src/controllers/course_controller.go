package controllers

import (
	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/src/services/courses"
	"Backend-Bluelock-007/src/utils"
	"math"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

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
// @Success      200  {object}  models.PaginatedResponse
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

	response := models.PaginatedResponse[models.Course]{
		Data: result,
		Meta: models.PaginationMeta{
			Page:        params.Page,
			Limit:       params.Limit,
			Total:       total,
			TotalPages:  totalPages,
			HasNext:     params.Page < totalPages,
			HasPrevious: params.Page > 1,
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
