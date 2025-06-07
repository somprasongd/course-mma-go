package model

import (
	"go-mma/modules/customer/domainerrors"
	"go-mma/modules/customer/internal/domain/event"
	"go-mma/shared/common/domain"
	"go-mma/shared/common/idgen"
	"time"
)

type Customer struct {
	ID               int64     `db:"id"`
	Email            string    `db:"email"`
	Credit           int       `db:"credit"`
	CreatedAt        time.Time `db:"created_at"`
	UpdatedAt        time.Time `db:"updated_at"`
	domain.Aggregate           // ทำให้ model เป็น aggregate root มี domain events
}

func NewCustomer(email string, credit int) *Customer {
	customer := &Customer{
		ID:     idgen.GenerateTimeRandomID(),
		Email:  email,
		Credit: credit,
	}

	customer.AddDomainEvent(event.NewCustomerCreatedDomainEvent(customer.ID, customer.Email))

	return customer
}

func (c *Customer) ReserveCredit(v int) error {
	newCredit := c.Credit - v
	if newCredit < 0 {
		return domainerrors.ErrInsufficientCredit
	}
	c.Credit = newCredit
	return nil
}

func (c *Customer) ReleaseCredit(v int) {
	if c.Credit <= 0 {
		c.Credit = 0
	}
	c.Credit = c.Credit + v
	return
}
