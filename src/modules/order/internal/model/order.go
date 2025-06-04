package model

import "time"

type Order struct {
	ID         int        `db:"id"`
	CustomerID int        `db:"customer_id"`
	OrderTotal int        `db:"order_total"`
	CreatedAt  time.Time  `db:"created_at"`
	CanceledAt *time.Time `db:"canceled_at"`
}

func NewOrder(customerID int, orderTotal int) *Order {
	return &Order{
		CustomerID: customerID,
		OrderTotal: orderTotal,
	}
}
