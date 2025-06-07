package customer

import (
	"go-mma/modules/customer/internal/domain/event"
	"go-mma/modules/customer/internal/domain/eventhandler"
	"go-mma/modules/customer/internal/feature/create"
	getbyid "go-mma/modules/customer/internal/feature/get-by-id"
	releasecredit "go-mma/modules/customer/internal/feature/release-credit"
	reservecredit "go-mma/modules/customer/internal/feature/reserve-credit"
	"go-mma/modules/customer/internal/repository"
	"go-mma/shared/common/domain"
	"go-mma/shared/common/mediator"
	"go-mma/shared/common/module"
	"go-mma/shared/common/registry"

	notiModule "go-mma/modules/notification"
	notiService "go-mma/modules/notification/service"

	"github.com/gofiber/fiber/v3"
)

func NewModule(mCtx *module.ModuleContext) module.Module {
	return &moduleImp{mCtx: mCtx}
}

type moduleImp struct {
	mCtx *module.ModuleContext
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

	// Register domain event handlerAdd commentMore actions
	dispatcher := domain.NewSimpleDomainEventDispatcher()
	dispatcher.Register(event.CustomerCreatedDomainEventType, eventhandler.NewCustomerCreatedDomainEventHandler(notiSvc))

	repo := repository.NewCustomerRepository(m.mCtx.DBCtx)

	mediator.Register(create.NewCreateCustomerCommandHandler(m.mCtx.Transactor, repo, dispatcher))
	mediator.Register(getbyid.NewGetCustomerByIDQueryHandler(repo))
	mediator.Register(reservecredit.NewReserveCreditCommandHandler(m.mCtx.Transactor, repo))
	mediator.Register(releasecredit.NewReleaseCreditCommandHandler(m.mCtx.Transactor, repo))

	return nil
}

func (m *moduleImp) RegisterRoutes(router fiber.Router) {
	customers := router.Group("/customers")
	create.NewEndpoint(customers, "")
}
