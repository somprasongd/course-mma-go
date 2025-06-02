package handler

import (
	"go-mma/dto"
	"go-mma/service"
	"go-mma/util/errs"
	"strings"

	"github.com/gofiber/fiber/v3"
)

type CustomerHandler struct {
	custService service.CustomerService
}

func NewCustomerHandler(custService service.CustomerService) *CustomerHandler {
	return &CustomerHandler{
		custService: custService,
	}
}

func (h *CustomerHandler) CreateCustomer(c fiber.Ctx) error {
	// 1. รับ request body มาเป็น DTO
	var req dto.CreateCustomerRequest
	if err := c.Bind().Body(&req); err != nil {
		return errs.InputValidationError(err.Error())
	}

	// 2. ตรวจสอบความถูกต้อง (validate)
	if err := req.Validate(); err != nil {
		return errs.InputValidationError(strings.Join(strings.Split(err.Error(), "\n"), ", "))
	}

	// 3. ส่งไปที่ Service Layer
	resp, err := h.custService.CreateCustomer(c.Context(), &req)

	// 4. จัดการ error จาก Service Layer หากเกิดขึ้น
	if err != nil {
		return err
	}

	// 5. ตอบกลับ client
	return c.Status(fiber.StatusCreated).JSON(resp)
}
