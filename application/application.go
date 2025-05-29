package application

import (
	"go-mma/config"
	"go-mma/data/sqldb"
	"log"
)

type Application struct {
	config     config.Config
	httpServer HTTPServer
	db         sqldb.DBContext
}

func New(config config.Config, db sqldb.DBContext) *Application {
	return &Application{
		config:     config,
		httpServer: newHTTPServer(config),
		db:         db,
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
