package controllers

import (
	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/src/services"

	"github.com/gofiber/fiber/v2"
)

// HandleError ฟังก์ชันช่วยเหลือในการส่ง Error Response
func HandleError(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(models.ErrorResponse{
		Status:  status,
		Message: message,
	})
}

// CreateAdmin godoc
// @Summary      Create a new admin
// @Description  Create a new admin
// @Tags         admins
// @Accept       json
// @Produce      json
// @Param        admin  body  models.Admin  true  "Admin object"
// @Success      201  {object}  models.Admin
// @Failure      400  {object}  models.ErrorResponse
// @Failure      500  {object}  models.ErrorResponse
// @Router       /admins [post]
func CreateAdmin(c *fiber.Ctx) error {
	var admin models.Admin
	if err := c.BodyParser(&admin); err != nil {
		return HandleError(c, fiber.StatusBadRequest, "Invalid input: "+err.Error())
	}

	err := services.CreateAdmin(&admin)
	if err != nil {
		return HandleError(c, fiber.StatusInternalServerError, "Error creating admin: "+err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Admin created successfully 12345",
		"admin":   admin,
	})
}


// GetAdmins godoc
// @Summary      Get all admins
// @Description  Get all admins
// @Tags         admins
// @Produce      json
// @Success      200  {array}  models.Admin
// @Failure      500  {object}  models.ErrorResponse
// @Router       /admins [get]
func GetAdmins(c *fiber.Ctx) error {
	admins, err := services.GetAllAdmins()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error fetching admins",
		})
	}

	return c.JSON(admins)
}

// GetAdminByID godoc
// @Summary      Get an admin by ID
// @Description  Get an admin by ID
// @Tags         admins
// @Produce      json
// @Param        id   path  string  true  "Admin ID"
// @Success      200  {object}  models.Admin
// @Failure      404  {object}  models.ErrorResponse
// @Failure      500  {object}  models.ErrorResponse
// @Router       /admins/{id} [get]
func GetAdminByID(c *fiber.Ctx) error {
	id := c.Params("id")
	admin, err := services.GetAdminByID(id)
	if err != nil {
		return HandleError(c, fiber.StatusNotFound, "Admin not found")
	}

	return c.JSON(admin)
}


// UpdateAdmin godoc
// @Summary      Update an admin
// @Description  Update an admin
// @Tags         admins
// @Accept       json
// @Produce      json
// @Param        id   path  string  true  "Admin ID"
// @Param        admin  body  models.Admin  true  "Admin object"
// @Success      200  {object}  models.Admin
// @Failure      400  {object}  models.ErrorResponse
// @Failure      500  {object}  models.ErrorResponse
// @Router       /admins/{id} [put]
func UpdateAdmin(c *fiber.Ctx) error {
	id := c.Params("id")
	var admin models.Admin

	if err := c.BodyParser(&admin); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	err := services.UpdateAdmin(id, &admin)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error updating admin",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Admin updated successfully",
	})
}

// DeleteAdmin godoc
// @Summary      Delete an admin
// @Description  Delete an admin
// @Tags         admins
// @Produce      json
// @Param        id   path  string  true  "Admin ID"
// @Success      200  {object}  models.Admin
// @Failure      500  {object}  models.ErrorResponse
// @Router       /admins/{id} [delete]
// DeleteAdmin - ลบผู้ใช้
func DeleteAdmin(c *fiber.Ctx) error {
	id := c.Params("id")
	err := services.DeleteAdmin(id)
	if err != nil {
		return HandleError(c, fiber.StatusInternalServerError, "Error deleting admin: "+err.Error())
	}

	return c.JSON(fiber.Map{
		"message": "Admin deleted successfully",
	})
}