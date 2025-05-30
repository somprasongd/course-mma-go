package main

import (
	"fmt"
	"go-mma/application"
	"go-mma/config"
	"go-mma/data/sqldb"
	"go-mma/util/logger"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	logger.Init()
	defer logger.Log.Sync() // Ensure logs are written before exit

	config, err := config.Load()
	if err != nil {
		logger.Log.Panic(err.Error())
	}
	db, closeDB, err := sqldb.New(config.DSN)
	if err != nil {
		logger.Log.Panic(err.Error())
	}
	defer func() {
		if err := closeDB(); err != nil {
			logger.Log.Error(fmt.Sprintf("Error closing database: %v", err))
		}
	}()

	app := application.New(*config, db)
	app.RegisterRoutes()
	app.Run()

	// Wait for shutdown signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	app.Shutdown()

	// Optionally: close DB, cleanup, etc.

	logger.Log.Info("Shutdown complete.")
}
