package cancel

import (
	"context"
	"go-mma/modules/order/domainerrors"
	"go-mma/modules/order/internal/repository"
	"go-mma/shared/common/logger"
	"go-mma/shared/common/mediator"
	"go-mma/shared/common/storage/sqldb/transactor"
	"go-mma/shared/contract/customercontract"
)

type cancelOrderCommandHandler struct {
	transactor transactor.Transactor
	orderRepo  repository.OrderRepository
}

func NewCancelOrderCommandHandler(
	transactor transactor.Transactor,
	orderRepo repository.OrderRepository) *cancelOrderCommandHandler {
	return &cancelOrderCommandHandler{
		transactor: transactor,
		orderRepo:  orderRepo,
	}
}

func (h *cancelOrderCommandHandler) Handle(ctx context.Context, cmd *CancelOrderCommand) (*mediator.NoResponse, error) {
	// ตรวจสอบ order id
	order, err := h.orderRepo.FindByID(ctx, cmd.ID)
	if err != nil {
		logger.Log.Error(err.Error())
		return nil, err
	}

	if order == nil {
		return nil, domainerrors.ErrNoOrderID
	}

	err = h.transactor.WithinTransaction(ctx, func(ctx context.Context) error {

		// ยกเลิก order
		if err := h.orderRepo.Cancel(ctx, order.ID); err != nil {
			logger.Log.Error(err.Error())
			return err
		}

		// Business Logic: คืนยอด credit
		if _, err = mediator.Send[*customercontract.ReleaseCreditCommand, *mediator.NoResponse](
			ctx,
			&customercontract.ReleaseCreditCommand{
				CustomerID:   order.CustomerID,
				CreditAmount: order.OrderTotal,
			},
		); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return nil, nil
}
