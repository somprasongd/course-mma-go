package response

import (
	"go-mma/util/errs"

	"github.com/gofiber/fiber/v3"
)

func JSONError(c fiber.Ctx, err error) error {
	// Convert non-AppError to AppError with type ErrOperationFailed
	appErr, ok := err.(*errs.AppError)
	if !ok {
		appErr = errs.New(
			errs.ErrOperationFailed,
			err.Error(),
			err,
		)
	}

	// Get the appropriate HTTP status code
	statusCode := errs.GetHTTPStatus(err)

	// Return structured response with error type and message
	return c.Status(statusCode).JSON(appErr)
}
