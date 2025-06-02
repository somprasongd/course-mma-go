// internal/middleware/error_handler.go
package middleware

import (
	"go-mma/util/response"

	"github.com/gofiber/fiber/v3"
)

func ResponseError() fiber.Handler {
	return func(c fiber.Ctx) error {
		err := c.Next()
		if err == nil {
			return nil
		}

		return response.JSONError(c, err)
	}
}
