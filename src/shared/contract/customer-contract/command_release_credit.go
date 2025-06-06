package customercontract

type ReleaseCreditCommand struct {
	CustomerID   int `json:"customer_id"`
	CreditAmount int `json:"credit_amount"`
}
