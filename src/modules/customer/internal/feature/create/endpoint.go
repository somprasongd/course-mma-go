package create

import (
	"go-mma/shared/common/errs"
	"go-mma/shared/common/mediator"
	"strings"

	"github.com/gofiber/fiber/v3"
)

func NewEndpoint(router fiber.Router, path string) {
	router.Post(path, createCustomerHTTPHandler)
}

func createCustomerHTTPHandler(c fiber.Ctx) error {
	// 1. รับ request body มาเป็น DTO
	var req CreateCustomerRequest
	if err := c.Bind().Body(&req); err != nil {
		return errs.InputValidationError(err.Error())
	}

	// 2. ตรวจสอบความถูกต้อง (validate)
	if err := req.Validate(); err != nil {
		return errs.InputValidationError(strings.Join(strings.Split(err.Error(), "\n"), ", "))
	}

	// 3. ส่งไปที่ Command Handler
	resp, err := mediator.Send[*CreateCustomerCommand, *CreateCustomerCommandResult](
		c.Context(),
		&CreateCustomerCommand{CreateCustomerRequest: req},
	)

	// 4. จัดการ error จาก feature หากเกิดขึ้น
	if err != nil {
		return err
	}

	// 5. ตอบกลับ client
	return c.Status(fiber.StatusCreated).JSON(resp)
}
