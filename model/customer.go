package model

import "time"

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
