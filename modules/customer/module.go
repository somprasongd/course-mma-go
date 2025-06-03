package customer

import (
	"go-mma/modules/customer/handler"
	"go-mma/modules/customer/repository"
	"go-mma/modules/customer/service"
	"go-mma/util/module"
	"go-mma/util/registry"

	notiModule "go-mma/modules/notification"
	notiService "go-mma/modules/notification/service"

	"github.com/gofiber/fiber/v3"
)

const (
	CustomerServiceKey registry.ServiceKey = "CustomerService"
)

func NewModule(mCtx *module.ModuleContext) module.Module {
	return &moduleImp{mCtx: mCtx}
}

type moduleImp struct {
	mCtx    *module.ModuleContext
	custSvc service.CustomerService
}

func (m *moduleImp) APIVersion() string {
	return "v1"
}

func (m *moduleImp) Init(reg registry.ServiceRegistry) error {
	// Resolve NotificationService from the registry
	notiSvc, err := registry.ResolveAs[notiService.NotificationService](reg, notiModule.NotificationServiceKey)
	if err != nil {
		return err
	}

	repo := repository.NewCustomerRepository(m.mCtx.DBCtx)
	m.custSvc = service.NewCustomerService(m.mCtx.Transactor, repo, notiSvc)

	reg.Register(CustomerServiceKey, m.custSvc)

	return nil
}

func (m *moduleImp) RegisterRoutes(router fiber.Router) {
	// wiring dependencies
	hdl := handler.NewCustomerHandler(m.custSvc)

	customers := router.Group("/customers")
	customers.Post("", hdl.CreateCustomer)
}
