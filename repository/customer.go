package repository

import (
	"context"
	"fmt"
	"go-mma/data/sqldb"
	"go-mma/model"
	"time"
)

type CustomerRepository struct {
	dbCtx sqldb.DBContext // dbCtx is an instance of DBContext interface for interacting with the database
}

func NewCustomerRepository(dbCtx sqldb.DBContext) *CustomerRepository {
	return &CustomerRepository{
		dbCtx: dbCtx, // Initialize the dbCtx field with the passed DBContext parameter
	}
}

func (r *CustomerRepository) Create(ctx context.Context, customer *model.Customer) error {
	query := `
INSERT INTO public.customers (name, credit)
VALUES ($1, $2)
RETURNING *
`

	// Use context.WithTimeout to set a timeout for the database query execution
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	err := r.dbCtx.DB().
		QueryRowxContext(ctx, query, customer.Name, customer.Credit).
		StructScan(customer)
	if err != nil {
		return fmt.Errorf("failed to create customer: %w", err) // Return an error if the query execution fails
	}
	return nil // Return nil if the operation is successful
}
