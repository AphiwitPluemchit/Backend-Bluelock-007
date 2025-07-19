package controllers

import (
	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/src/services/courses"
	"Backend-Bluelock-007/src/utils"

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
// @Summary      Get all courses
// @Description  Get all courses
// @Tags         courses
// @Produce      json
// @Success      200  {array}  models.Course
// @Failure      500  {object}  models.ErrorResponse
// @Router       /courses [get]
func GetAllCourses(c *fiber.Ctx) error {
	coursesList, err := courses.GetAllCourses()
	if err != nil {
		return utils.HandleError(c, fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(coursesList)
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
