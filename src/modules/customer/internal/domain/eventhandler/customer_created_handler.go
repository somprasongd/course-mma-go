package eventhandler

import (
	"context"
	"go-mma/modules/customer/internal/domain/event"
	notiService "go-mma/modules/notification/service"
	"go-mma/shared/common/domain"
)

type customerCreatedDomainEventHandler struct {
	notiSvc notiService.NotificationService
}

func NewCustomerCreatedDomainEventHandler(notiSvc notiService.NotificationService) domain.DomainEventHandler {
	return &customerCreatedDomainEventHandler{
		notiSvc: notiSvc,
	}
}

func (h *customerCreatedDomainEventHandler) Handle(ctx context.Context, evt domain.DomainEvent) error {
	e, ok := evt.(*event.CustomerCreatedDomainEvent) // ใช้ pointer

	if !ok {
		return domain.ErrInvalidEvent
	}
	// ส่งอีเมลต้อนรับ
	if err := h.notiSvc.SendEmail(e.Email, "Welcome to our service!", map[string]any{
		"message": "Thank you for joining us! We are excited to have you as a member.",
	}); err != nil {
		return err
	}

	return nil
}
