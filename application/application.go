package application

import (
	"go-mma/config"
	"log"
)

type Application struct {
	config     config.Config
	httpServer HTTPServer
}

func New(config config.Config) *Application {
	return &Application{
		config:     config,
		httpServer: newHTTPServer(config),
	}
}

func (app *Application) Run() error {
	app.httpServer.Start()

	return nil
}

func (app *Application) Shutdown() error {
	// Gracefully close fiber server
	log.Println("Shutting down server")
	if err := app.httpServer.Shutdown(); err != nil {
		log.Fatalf("Error shutting down server: %v", err)
	}
	log.Println("Server stopped")

	return nil
}

func (app *Application) RegisterRoutes() {
	app.httpServer.RegisterRoutes()
}
