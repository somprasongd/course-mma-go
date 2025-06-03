package application

import (
	"fmt"
	"go-mma/config"
	"go-mma/data/sqldb"
	"go-mma/util/logger"
	"go-mma/util/module"
	"go-mma/util/registry"
)

type Application struct {
	config          config.Config
	httpServer      HTTPServer
	serviceRegistry registry.ServiceRegistry
}

func New(config config.Config, db sqldb.DBContext) *Application {
	return &Application{
		config:          config,
		httpServer:      newHTTPServer(config),
		serviceRegistry: registry.NewServiceRegistry(),
	}
}

func (app *Application) Run() error {
	app.httpServer.Start()

	return nil
}

func (app *Application) Shutdown() error {
	// Gracefully close fiber server
	logger.Log.Info("Shutting down server")
	if err := app.httpServer.Shutdown(); err != nil {
		logger.Log.Fatal(fmt.Sprintf("Error shutting down server: %v", err))
	}
	logger.Log.Info("Server stopped")

	return nil
}

func (app *Application) RegisterModules(modules []module.Module) {
	for _, m := range modules {
		// Initialize each module
		if err := m.Init(app.serviceRegistry); err != nil {
			logger.Log.Fatal(fmt.Sprintf("module initialization error: %v", err))
		}

		// Register routes for each module
		groupPrefix := "/api"
		if len(m.APIVersion()) > 0 {
			groupPrefix = fmt.Sprintf("/api/%s", m.APIVersion())
		}
		group := app.httpServer.Group(groupPrefix)
		m.RegisterRoutes(group)
	}
}
