package main

import (
	"fmt"
	"go-mma/application"
	"go-mma/config"
	"go-mma/modules/customer"
	"go-mma/modules/notification"
	"go-mma/modules/order"
	"go-mma/shared/common/logger"
	"go-mma/shared/common/module"
	"go-mma/shared/common/storage/sqldb"
	"go-mma/shared/common/storage/sqldb/transactor"
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

	app := application.New(*config)

	transactor, dbCtx := transactor.New(
		db.DB(),
		transactor.WithNestedTransactionStrategy(transactor.NestedTransactionsSavepoints),
	)
	mCtx := module.NewModuleContext(transactor, dbCtx)
	err = app.RegisterModules(
		notification.NewModule(mCtx),
		customer.NewModule(mCtx),
		order.NewModule(mCtx),
	)
	if err != nil {
		logger.Log.Fatal(fmt.Sprintf("Error initializing module: %v", err))
	}

	app.Run()

	// Wait for shutdown signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	app.Shutdown()

	// Optionally: close DB, cleanup, etc.

	logger.Log.Info("Shutdown complete.")
}
