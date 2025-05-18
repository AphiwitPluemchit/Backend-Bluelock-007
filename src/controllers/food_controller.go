package controllers

import (
	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/src/services"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CreateFood godoc
// @Summary      เพิ่มข้อมูลอาหาร
// @Description  สร้างข้อมูลอาหาร 1 รายการ
// @Tags         foods
// @Accept       json
// @Produce      json
// @Param        body body models.Food true "ข้อมูลอาหาร"
// @Success      201  {object}  models.Food
// @Failure      400  {object}  models.ErrorResponse
// @Failure      500  {object}  models.ErrorResponse
// @Router       /foods [post]
func CreateFood(c *fiber.Ctx) error {
	var input models.CreateFoodInput
if err := c.BodyParser(&input); err != nil {
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"error": "Invalid input",
	})
}

// ตรวจสอบว่า name ถูกส่งมาหรือไม่
if input.Name == "" {
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"error": "Food name is required",
	})
}

// สร้าง food ใหม่โดย generate ID เอง
food := models.Food{
	ID:   primitive.NewObjectID(),
	Name: input.Name,
}

err := services.CreateFood(&food)
if err != nil {
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		"error": "Error creating food",
	})
}

return c.Status(fiber.StatusCreated).JSON(food)

}

// GetFoods godoc
// @Summary      ดึงรายการอาหารทั้งหมด
// @Description  ดึงข้อมูลอาหารที่มีอยู่ทั้งหมด
// @Tags         foods
// @Produce      json
// @Success      200  {array}  models.Food
// @Failure      500  {object}  models.ErrorResponse
// @Router       /foods [get]
func GetFoods(c *fiber.Ctx) error {
	foods, err := services.GetAllFoods()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error fetching foods",
		})
	}

	return c.JSON(foods)
}

// GetFoodByID godoc
// @Summary      ดึงข้อมูลอาหารตาม ID
// @Description  ค้นหาข้อมูลอาหารโดยใช้ ID
// @Tags         foods
// @Produce      json
// @Param        id path string true "Food ID"
// @Success      200  {object}  models.Food
// @Failure      404  {object}  models.ErrorResponse
// @Router       /foods/{id} [get]
func GetFoodByID(c *fiber.Ctx) error {
	id := c.Params("id")
	food, err := services.GetFoodByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Food not found",
		})
	}

	return c.JSON(food)
}

// UpdateFood godoc
// @Summary      อัปเดตข้อมูลอาหาร
// @Description  อัปเดตข้อมูลอาหารที่มีอยู่
// @Tags         foods
// @Accept       json
// @Produce      json
// @Param        id path string true "Food ID"
// @Param        body body models.Food true "ข้อมูลอาหารที่ต้องการอัปเดต"
// @Success      200  {object}  models.SuccessResponse
// @Failure      400  {object}  models.ErrorResponse
// @Failure      500  {object}  models.ErrorResponse
// @Router       /foods/{id} [put]
func UpdateFood(c *fiber.Ctx) error {
	id := c.Params("id")
	var food models.Food

	if err := c.BodyParser(&food); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	err := services.UpdateFood(id, &food)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error updating food",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Food updated successfully",
	})
}

// DeleteFood godoc
// @Summary      ลบข้อมูลอาหาร
// @Description  ลบข้อมูลอาหารออกจากระบบ
// @Tags         foods
// @Param        id path string true "Food ID"
// @Success      200  {object}  models.SuccessResponse
// @Failure      400  {object}  models.ErrorResponse
// @Failure      500  {object}  models.ErrorResponse
// @Router       /foods/{id} [delete]
func DeleteFood(c *fiber.Ctx) error {
	id := c.Params("id")
	err := services.DeleteFood(id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error deleting food",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Food deleted successfully",
	})
}
