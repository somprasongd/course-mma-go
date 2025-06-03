package module

import (
	"go-mma/util/transactor"

	"github.com/gofiber/fiber/v3"
)

type Module interface {
	APIVersion() string
	RegisterRoutes(r fiber.Router)
}

type ModuleContext struct {
	Transactor transactor.Transactor
	DBCtx      transactor.DBContext
}

func NewModuleContext(transactor transactor.Transactor, dbCtx transactor.DBContext) *ModuleContext {
	return &ModuleContext{
		Transactor: transactor,
		DBCtx:      dbCtx,
	}
}
