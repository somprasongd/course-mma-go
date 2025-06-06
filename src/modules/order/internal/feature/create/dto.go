package create

import "fmt"

type CreateOrderRequest struct {
	CustomerID int `json:"customer_id"`
	OrderTotal int `json:"order_total"`
}

func (r *CreateOrderRequest) Validate() error {
	if r.CustomerID <= 0 {
		return fmt.Errorf("customer_id is required")
	}
	if r.OrderTotal <= 0 {
		return fmt.Errorf("order_total must be greater than 0")
	}
	return nil
}

type CreateOrderResponse struct {
	ID int `json:"id"`
}
