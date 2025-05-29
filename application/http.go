package application

import (
	"context"
	"fmt"
	"go-mma/config"
	"go-mma/data/sqldb"
	"go-mma/handler"
	"log"
	"net/http"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
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
	app.Use(logger.New())  // logs HTTP request/response details
	app.Use(recover.New()) // recovers from any panics
	app.Use(cors.New())    // allows all origins

	return app
}

func (s *httpServer) Start() {
	go func() {
		log.Printf("Starting server on port %d", s.config.HTTPPort)
		if err := s.app.Listen(fmt.Sprintf(":%d", s.config.HTTPPort)); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error starting server: %v", err)
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
		hdlr := handlers.NewCustomerHandler(db)
		customers.Post("", hdlr.CreateCustomer)
	}

	orders := v1.Group("/orders")
	{
		hdlr := handlers.NewOrderHandler()
		orders.Post("", hdlr.CreateOrder)
		orders.Delete("/:orderID", hdlr.CancelOrder)
	}
}
