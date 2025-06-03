package notification

import (
	"go-mma/modules/notification/service"
	"go-mma/util/module"
	"go-mma/util/registry"

	"github.com/gofiber/fiber/v3"
)

const (
	NotificationServiceKey registry.ServiceKey = "NotificationService"
)

func NewModule(mCtx *module.ModuleContext) module.Module {
	return &moduleImp{mCtx: mCtx}
}

type moduleImp struct {
	mCtx    *module.ModuleContext
	notiSvc service.NotificationService
}

func (m *moduleImp) APIVersion() string {
	return "v1"
}

func (m *moduleImp) Init(reg registry.ServiceRegistry) error {
	m.notiSvc = service.NewNotificationService()

	reg.Register(NotificationServiceKey, m.notiSvc)

	return nil
}

func (m *moduleImp) RegisterRoutes(router fiber.Router) {

}
