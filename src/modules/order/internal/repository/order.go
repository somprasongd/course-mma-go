package repository

import (
	"context"
	"database/sql"
	"fmt"
	"go-mma/modules/order/internal/model"
	"go-mma/shared/common/errs"
	"go-mma/shared/common/storage/sqldb/transactor"
	"time"
)

type OrderRepository interface {
	Create(ctx context.Context, order *model.Order) error
	FindByID(ctx context.Context, id int) (*model.Order, error)
	Cancel(ctx context.Context, id int) error
}

type orderRepository struct {
	dbCtx transactor.DBContext
}

func NewOrderRepository(dbCtx transactor.DBContext) OrderRepository {
	return &orderRepository{
		dbCtx: dbCtx,
	}
}

func (r *orderRepository) Create(ctx context.Context, m *model.Order) error {
	query := `
	INSERT INTO public.orders (
			customer_id, order_total
	)
	VALUES ($1, $2)
	RETURNING *
	`

	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	err := r.dbCtx(ctx).QueryRowxContext(ctx, query, m.CustomerID, m.OrderTotal).StructScan(m)
	if err != nil {
		return errs.HandleDBError(fmt.Errorf("failed to create order: %w", err))
	}
	return nil
}

func (r *orderRepository) FindByID(ctx context.Context, id int) (*model.Order, error) {
	query := `
	SELECT *
	FROM public.orders
	WHERE id = $1
	AND canceled_at IS NULL
`
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	var order model.Order
	err := r.dbCtx(ctx).QueryRowxContext(ctx, query, id).StructScan(&order)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, errs.HandleDBError(fmt.Errorf("failed to get order by ID: %w", err))
	}
	return &order, nil
}

func (r *orderRepository) Cancel(ctx context.Context, id int) error {
	query := `
	UPDATE public.orders
	SET canceled_at = current_timestamp
	WHERE id = $1
`
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	_, err := r.dbCtx(ctx).ExecContext(ctx, query, id)
	if err != nil {
		return errs.HandleDBError(fmt.Errorf("failed to cancel order: %w", err))
	}
	return nil
}
