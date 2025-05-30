package middleware

import (
	"go-mma/util/logger"
	"time"

	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

func RequestLogger() fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		duration := time.Since(start)

		logger.Log.Info("http request",
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
			zap.Int("status", c.Response().StatusCode()),
			zap.Duration("duration", duration),
		)

		return err
	}
}
