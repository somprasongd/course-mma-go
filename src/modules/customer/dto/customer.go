package dto

type Customer struct {
	ID     int    `json:"id"`
	Email  string `json:"email"`
	Credit int    `json:"credit"`
}
