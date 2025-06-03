package main

import (
	"fmt"
	"go-mma/application"
	"go-mma/config"
	"go-mma/data/sqldb"
	"go-mma/modules/customer"
	"go-mma/modules/notification"
	"go-mma/modules/order"
	"go-mma/util/logger"
	"go-mma/util/module"
	"go-mma/util/transactor"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	closeLog, err := logger.Init()
	if err != nil {
		panic(err.Error())
	}
	defer closeLog()

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

	transactor, dbCtx := transactor.New(db.DB())

	app := application.New(*config, db)

	mCtx := module.NewModuleContext(transactor, dbCtx)
	app.RegisterModules([]module.Module{
		notification.NewModule(mCtx),
		customer.NewModule(mCtx),
		order.NewModule(mCtx),
	})

	app.Run()

	// Wait for shutdown signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	app.Shutdown()

	// Optionally: close DB, cleanup, etc.

	logger.Log.Info("Shutdown complete.")
}
