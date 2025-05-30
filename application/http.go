package application

import (
	"context"
	"fmt"
	"go-mma/application/middleware"
	"go-mma/config"
	"go-mma/data/sqldb"
	"go-mma/handler"
	"go-mma/repository"
	"go-mma/service"
	"go-mma/util/logger"
	"net/http"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/recover"
)

type HTTPServer interface {
	Start()
	Shutdown() error
	RegisterRoutes(db sqldb.DBContext)
}

type httpServer struct {
	config config.Config
	app    *fiber.App
}

func newHTTPServer(config config.Config) HTTPServer {
	return &httpServer{
		config: config,
		app:    newFiber(config),
	}
}

func newFiber(config config.Config) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName: fmt.Sprintf("Go MMA v%s", config.AppVersion),
	})

	// global middleware
	app.Use(cors.New())                 // CORS ลำดับแรก เพื่อให้ OPTIONS request ผ่านได้เสมอ
	app.Use(recover.New())              // auto-recovers from panic (internal only)
	app.Use(middleware.RequestLogger()) // logs HTTP request

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

func (s *httpServer) Router() *fiber.App {
	return s.app
}

func (s *httpServer) RegisterRoutes(db sqldb.DBContext) {
	v1 := s.app.Group("/api/v1")

	customers := v1.Group("/customers")
	{
		repo := repository.NewCustomerRepository(db)
		svc := service.NewCustomerService(repo)
		hdlr := handler.NewCustomerHandler(svc)
		customers.Post("", hdlr.CreateCustomer)
	}

	orders := v1.Group("/orders")
	{
		hdlr := handler.NewOrderHandler()
		orders.Post("", hdlr.CreateOrder)
		orders.Delete("/:orderID", hdlr.CancelOrder)
	}
}
