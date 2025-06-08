package eventhandler

import (
	"context"
	"go-mma/modules/customer/internal/domain/event"
	"go-mma/shared/common/domain"
	"go-mma/shared/common/eventbus"
	"go-mma/shared/messaging"
)

type customerCreatedDomainEventHandler struct {
	eventBus eventbus.EventBus
}

func NewCustomerCreatedDomainEventHandler(eventBus eventbus.EventBus) domain.DomainEventHandler {
	return &customerCreatedDomainEventHandler{
		eventBus: eventBus,
	}
}

func (h *customerCreatedDomainEventHandler) Handle(ctx context.Context, evt domain.DomainEvent) error {
	e, ok := evt.(*event.CustomerCreatedDomainEvent) // ใช้ pointer

	if !ok {
		return domain.ErrInvalidEvent
	}

	// สร้าง IntegrationEvent จาก Domain Event
	integrationEvent := messaging.NewCustomerCreatedIntegrationEvent(
		e.CustomerID,
		e.Email,
	)

	return h.eventBus.Publish(ctx, integrationEvent)
}
