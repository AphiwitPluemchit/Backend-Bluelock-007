package controllers

import (
	"Backend-Bluelock-007/src/models"
	"Backend-Bluelock-007/src/services"

	"github.com/gofiber/fiber/v2"
)

func GetOrderByIDHandler(c *fiber.Ctx) error {
	id := c.Params("id")
	order, err := services.GetOrderByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Order not found",
		})
	}

	return c.JSON(order)
}

// CreateOrder สร้างคำสั่งซื้อใหม่
func CreateOrder(c *fiber.Ctx) error {
	var order models.Order
	if err := c.BodyParser(&order); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	err := services.CreateOrder(&order)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error creating order",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Order created successfully",
	})
}

// GetOrders ดึงรายการคำสั่งซื้อทั้งหมด
func GetOrders(c *fiber.Ctx) error {
	orders, err := services.GetAllOrders() // เรียกข้อมูลจาก service ที่เกี่ยวข้อง
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error fetching orders",
		})
	}

	return c.JSON(orders)
}
