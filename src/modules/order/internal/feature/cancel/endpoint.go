package cancel

import (
	"go-mma/shared/common/errs"
	"go-mma/shared/common/mediator"
	"strconv"

	"github.com/gofiber/fiber/v3"
)

func NewEndpoint(router fiber.Router, path string) {
	router.Delete(path, cancelOrderHTTPHandler)
}

func cancelOrderHTTPHandler(c fiber.Ctx) error {
	// 1. อ่านค่า id จาก path param
	id := c.Params("orderID")

	// 2. ตรวจสอบรูปแบบ order id
	orderID, err := strconv.Atoi(id)
	if err != nil {
		return errs.InputValidationError("invalid order id")
	}

	// 3. ส่งไปที่ Command Handler
	_, err = mediator.Send[*CancelOrderCommand, *mediator.NoResponse](
		c.Context(),
		&CancelOrderCommand{ID: int64(orderID)},
	)

	// 4. จัดการ error จาก feature หากเกิดขึ้น
	if err != nil {
		return err
	}

	// 5. ตอบกลับ client
	return c.SendStatus(fiber.StatusNoContent)
}
