package customercontract

import (
	"context"
	"go-mma/shared/common/registry"
)

const (
	CreditManagerKey registry.ServiceKey = "customer:contract:credit"
)

type CustomerInfo struct {
	ID     int    `json:"id"`
	Email  string `json:"email"`
	Credit int    `json:"credit"`
}

type CustomerReader interface {
	GetCustomerByID(ctx context.Context, id int) (*CustomerInfo, error)
}

type CreditManager interface {
	CustomerReader // embed เพื่อ reuse
	ReserveCredit(ctx context.Context, id int, amount int) error
	ReleaseCredit(ctx context.Context, id int, amount int) error
}
