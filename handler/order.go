package handler

import (
	"go-mma/dto"
	"go-mma/service"
	"go-mma/util/errs"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
)

type OrderHandler struct {
	orderSvc *service.OrderService
}

func NewOrderHandler(orderSvc *service.OrderService) *OrderHandler {
	return &OrderHandler{orderSvc: orderSvc}
}

func (h *OrderHandler) CreateOrder(c fiber.Ctx) error {
	// 1. รับ request body มาเป็น DTO
	var req dto.CreateOrderRequest
	if err := c.Bind().Body(&req); err != nil {
		return errs.InputValidationError(err.Error())
	}

	// 2. ตรวจสอบความถูกต้อง (validate)
	if err := req.Validate(); err != nil {
		return errs.InputValidationError(strings.Join(strings.Split(err.Error(), "\n"), ", "))
	}

	// 3. ส่งไปที่ Service Layer
	resp, err := h.orderSvc.CreateOrder(c.Context(), &req)

	// 4. จัดการ error จาก Service Layer หากเกิดขึ้น
	if err != nil {
		return err
	}

	// 5. ตอบกลับ client
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *OrderHandler) CancelOrder(c fiber.Ctx) error {
	// 1. อ่านค่า id จาก path param
	id := c.Params("orderID")

	// 2. ตรวจสอบรูปแบบ order id
	orderID, err := strconv.Atoi(id)
	if err != nil {
		return errs.InputValidationError("invalid order id")
	}

	// 3. ส่งไปที่ Service Layer
	err = h.orderSvc.CancelOrder(c.Context(), orderID)

	// 4. จัดการ error จาก Service Layer หากเกิดขึ้น
	if err != nil {
		return err
	}

	// 5. ตอบกลับ client
	return c.SendStatus(fiber.StatusNoContent)
}
