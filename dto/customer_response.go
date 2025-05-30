package dto

type CreateCustomerResponse struct {
	ID int `json:"id"`
}

func NewCreateCustomerResponse(id int) *CreateCustomerResponse {
	return &CreateCustomerResponse{ID: id}
}
