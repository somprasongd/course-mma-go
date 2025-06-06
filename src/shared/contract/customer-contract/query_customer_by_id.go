package customercontract

type GetCustomerByIDQuery struct {
	ID int `json:"id"`
}

type GetCustomerByIDQueryResult struct {
	ID     int    `json:"id"`
	Email  string `json:"email"`
	Credit int    `json:"credit"`
}
