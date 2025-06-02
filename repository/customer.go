package repository

import (
	"context"
	"database/sql"
	"fmt"
	"go-mma/data/sqldb"
	"go-mma/model"
	"go-mma/util/errs"
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
INSERT INTO public.customers (email, credit)
VALUES ($1, $2)
RETURNING *
`

	// Use context.WithTimeout to set a timeout for the database query execution
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	err := r.dbCtx.DB().
		QueryRowxContext(ctx, query, customer.Email, customer.Credit).
		StructScan(customer)
	if err != nil {
		return errs.HandleDBError(fmt.Errorf("failed to create customer: %w", err))
	}
	return nil // Return nil if the operation is successful
}

func (r *CustomerRepository) FindByEmail(ctx context.Context, email string) (*model.Customer, error) {
	query := `
SELECT id, email, credit, created_at, updated_at
FROM public.customers
WHERE email = $1
`

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	customer := &model.Customer{}
	err := r.dbCtx.DB().
		QueryRowxContext(ctx, query, email).
		StructScan(customer)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, errs.HandleDBError(fmt.Errorf("failed to select customer: %w", err))
	}
	return customer, nil
}
