package main

import (
	"go-mma/application"
	"go-mma/config"
	"go-mma/data/sqldb"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	config, err := config.Load()
	if err != nil {
		log.Panic(err)
	}
	db, closeDB, err := sqldb.New(config.DSN)
	if err != nil {
		log.Panic(err)
	}
	defer func() {
		if err := closeDB(); err != nil {
			log.Println("Error closing database:", err)
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

	log.Println("Shutdown complete.")
}
