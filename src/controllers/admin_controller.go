package controllers

import (
	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/src/services/admins"
	"Backend-Bluelock-007/src/utils"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

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
	var req struct {
		Name     string `json:"name"`     // โปรไฟล์
		Email    string `json:"email"`    // auth
		Password string `json:"password"` // auth
	}

	// ✅ ดึงข้อมูลจาก Body
	if err := c.BodyParser(&req); err != nil {
		return utils.HandleError(c, fiber.StatusBadRequest, "Invalid input: "+err.Error())
	}

	// ✅ เตรียม Admin (Profile)
	admin := models.Admin{
		Name: req.Name,
	}

	// ✅ เตรียม User (Auth)
	user := models.User{
		Email:    strings.ToLower(req.Email),
		Password: req.Password,
	}
	user.Password = "123456"

	// ✅ เรียกใช้ service
	err := admins.CreateAdmin(&user, &admin)
	if err != nil {
		return utils.HandleError(c, fiber.StatusInternalServerError, "Error creating admin: "+err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Admin created successfully",
		"admin":   admin,
	})
}

// GetAdmins godoc
// @Summary      Get admins with pagination, search, and sorting
// @Description  Get admins with pagination, search, and sorting
// @Tags         admins
// @Produce      json
// @Param        page    query  int     false  "Page number"  default(1)
// @Param        limit   query  int     false  "Items per page"  default(10)
// @Param        search  query  string  false  "Search by name or email"
// @Param        sortBy  query  string  false  "Sort by field (default: name)"
// @Param        order   query  string  false  "Sort order (asc or desc)"  default(asc)
// @Success      200  {object}  map[string]interface{}
// @Failure      500  {object}  models.ErrorResponse
// @Router       /admins [get]
func GetAdmins(c *fiber.Ctx) error {
	// ใช้ DTO Default แล้วอัปเดตค่าจาก Query Parameter
	params := models.DefaultPagination()

	// อ่านค่า Query Parameter และแปลงเป็น int
	params.Page, _ = strconv.Atoi(c.Query("page", strconv.Itoa(params.Page)))
	params.Limit, _ = strconv.Atoi(c.Query("limit", strconv.Itoa(params.Limit)))
	params.Search = c.Query("search", params.Search)
	params.SortBy = c.Query("sortBy", params.SortBy)
	params.Order = c.Query("order", params.Order)

	// ดึงข้อมูลจาก Service
	admins, total, totalPages, err := admins.GetAllAdmins(params)
	if err != nil {
		return utils.HandleError(c, fiber.StatusInternalServerError, "Error getting admins: "+err.Error())
	}

	return c.JSON(fiber.Map{
		"data": admins,
		"meta": fiber.Map{
			"page":       params.Page,
			"limit":      params.Limit,
			"total":      total,
			"totalPages": totalPages,
		},
	})
	// ส่ง Response กลับไป

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
	admin, err := admins.GetAdminByID(id)
	if err != nil {
		return utils.HandleError(c, fiber.StatusNotFound, "Admin not found")
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

	err := admins.UpdateAdmin(id, &admin)
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
	err := admins.DeleteAdmin(id)
	if err != nil {
		return utils.HandleError(c, fiber.StatusInternalServerError, "Error deleting admin: "+err.Error())
	}

	return c.JSON(fiber.Map{
		"message": "Admin deleted successfully",
	})
}
