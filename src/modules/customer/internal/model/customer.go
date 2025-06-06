package model

import (
	"go-mma/modules/customer/domainerrors"
	"time"
)

type Customer struct {
	ID        int       `db:"id"`
	Email     string    `db:"email"`
	Credit    int       `db:"credit"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func NewCustomer(email string, credit int) *Customer {
	return &Customer{
		Email:  email,
		Credit: credit,
	}
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
