package application

import (
	"context"
	"fmt"
	"go-mma/application/middleware"
	"go-mma/config"
	"go-mma/shared/common/logger"
	"net/http"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
)

type HTTPServer interface {
	Start()
	Shutdown() error
	Group(prefix string) fiber.Router
}

type httpServer struct {
	config config.Config
	app    *fiber.App
}

func newHTTPServer(config config.Config) HTTPServer {
	return &httpServer{
		config: config,
		app:    newFiber(),
	}
}

func newFiber() *fiber.App {
	app := fiber.New(fiber.Config{
		AppName: "Go MMA v0.0.1",
	})

	// global middleware
	app.Use(cors.New())                 // CORS ลำดับแรก เพื่อให้ OPTIONS request ผ่านได้เสมอ
	app.Use(requestid.New())            // สร้าง request id ใน request header สำหรับการ debug
	app.Use(recover.New())              // auto-recovers from panic (internal only)
	app.Use(middleware.RequestLogger()) // logs HTTP request
	app.Use(middleware.ResponseError()) // จัดการ error จาก Handler Layer หากเกิดขึ้น

	return app
}

func (s *httpServer) Start() {
	go func() {
		logger.Log.Info(fmt.Sprintf("Starting server on port %d", s.config.HTTPPort))
		if err := s.app.Listen(fmt.Sprintf(":%d", s.config.HTTPPort)); err != nil && err != http.ErrServerClosed {
			logger.Log.Fatal(fmt.Sprintf("Error starting server: %v", err))
		}
	}()
}

func (s *httpServer) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), s.config.GracefulTimeout)
	defer cancel()
	return s.app.ShutdownWithContext(ctx)
}

func (s *httpServer) Group(prefix string) fiber.Router {
	return s.app.Group(prefix)
}
