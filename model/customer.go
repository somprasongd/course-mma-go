package model

import "time"

type Customer struct {
	ID        int       `db:"id"`
	Name      string    `db:"name"`
	Credit    int       `db:"credit"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func NewCustomer(name string, credit int) *Customer {
	return &Customer{
		Name:   name,
		Credit: credit,
	}
}
