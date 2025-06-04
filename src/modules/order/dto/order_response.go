package dto

type CreateOrderResponse struct {
	ID int `json:"id"`
}

func NewCreateOrderResponse(id int) *CreateOrderResponse {
	return &CreateOrderResponse{ID: id}
}
