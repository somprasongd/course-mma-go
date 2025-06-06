package create

type CreateOrderCommand struct {
	CreateOrderRequest
}

type CreateOrderCommandResult struct {
	CreateOrderResponse
}

func NewCreateOrderCommandResult(id int) *CreateOrderCommandResult {
	return &CreateOrderCommandResult{
		CreateOrderResponse{ID: id},
	}
}
