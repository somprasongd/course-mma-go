package repository

import (
	"context"
	"database/sql"
	"fmt"
	"go-mma/modules/customer/internal/model"
	"go-mma/shared/common/errs"
	"go-mma/shared/common/storage/sqldb/transactor"
	"time"
)

type CustomerRepository interface {
	Create(ctx context.Context, customer *model.Customer) error
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	FindByID(ctx context.Context, id int64) (*model.Customer, error)
	UpdateCredit(ctx context.Context, customer *model.Customer) error
}

type customerRepository struct {
	dbCtx transactor.DBContext
}

func NewCustomerRepository(dbCtx transactor.DBContext) CustomerRepository {
	return &customerRepository{
		dbCtx: dbCtx,
	}
}

func (r *customerRepository) Create(ctx context.Context, customer *model.Customer) error {
	query := `
INSERT INTO public.customers (id, email, credit)
VALUES ($1, $2, $3)
RETURNING *
`

	// Use context.WithTimeout to set a timeout for the database query execution
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	err := r.dbCtx(ctx).
		QueryRowxContext(ctx, query, customer.ID, customer.Email, customer.Credit).
		StructScan(customer)
	if err != nil {
		return errs.HandleDBError(fmt.Errorf("failed to create customer: %w", err))
	}
	return nil // Return nil if the operation is successful
}

func (r *customerRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	query := `
	SELECT 1
	FROM public.customers
	WHERE email = $1
	LIMIT 1
	`

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var exists int
	err := r.dbCtx(ctx).
		QueryRowxContext(ctx, query, email).
		Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, errs.HandleDBError(fmt.Errorf("failed to select customer: %w", err))
	}
	return true, nil
}

func (r *customerRepository) FindByID(ctx context.Context, id int64) (*model.Customer, error) {
	query := `
	SELECT *
	FROM public.customers
	WHERE id = $1
`
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	var customer model.Customer
	err := r.dbCtx(ctx).QueryRowxContext(ctx, query, id).StructScan(&customer)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, errs.HandleDBError(fmt.Errorf("failed to get customer by ID: %w", err))
	}

	return &customer, nil
}

func (r *customerRepository) UpdateCredit(ctx context.Context, m *model.Customer) error {
	query := `
	UPDATE public.customers
	SET credit = $2
	WHERE id = $1
	RETURNING *
`
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	err := r.dbCtx(ctx).QueryRowxContext(ctx, query, m.ID, m.Credit).StructScan(m)
	if err != nil {
		return errs.HandleDBError(fmt.Errorf("failed to update customer credit: %w", err))
	}
	return nil
}
