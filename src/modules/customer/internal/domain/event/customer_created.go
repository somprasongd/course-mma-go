package event

import (
	"go-mma/shared/common/domain"
	"time"
)

const (
	CustomerCreatedDomainEventType domain.EventName = "CustomerCreated"
)

type CustomerCreatedDomainEvent struct {
	domain.BaseDomainEvent
	CustomerID int64
	Email      string
}

func NewCustomerCreatedDomainEvent(custID int64, email string) *CustomerCreatedDomainEvent {
	return &CustomerCreatedDomainEvent{
		BaseDomainEvent: domain.BaseDomainEvent{
			Name: CustomerCreatedDomainEventType,
			At:   time.Now(),
		},
		CustomerID: custID,
		Email:      email,
	}
}
