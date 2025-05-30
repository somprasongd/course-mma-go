package handler

import (
	"go-mma/dto"
	"go-mma/service"
	"strings"

	"github.com/gofiber/fiber/v3"
)

type CustomerHandler struct {
	custService *service.CustomerService
}

func NewCustomerHandler(custService *service.CustomerService) *CustomerHandler {
	return &CustomerHandler{
		custService: custService,
	}
}

func (h *CustomerHandler) CreateCustomer(c fiber.Ctx) error {
	// 1. รับ request body มาเป็น DTO
	var req dto.CreateCustomerRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	// 2. ตรวจสอบความถูกต้อง (validate)
	if err := req.Validate(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": strings.Join(strings.Split(err.Error(), "\n"), ", ")})
	}

	// 3. ส่งไปที่ Service Layer
	resp, err := h.custService.CreateCustomer(c.Context(), &req)

	// 4. จัดการ error จาก Service Layer หากเกิดขึ้น
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// 5. ตอบกลับ client
	return c.Status(fiber.StatusCreated).JSON(resp)
}
