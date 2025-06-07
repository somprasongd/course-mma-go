package create

import (
	"context"
	"go-mma/modules/order/internal/model"
	"go-mma/modules/order/internal/repository"
	"go-mma/shared/common/logger"
	"go-mma/shared/common/mediator"
	"go-mma/shared/common/storage/sqldb/transactor"
	"go-mma/shared/contract/customercontract"

	notiService "go-mma/modules/notification/service"
)

type createOrderCommandHandler struct {
	transactor transactor.Transactor
	orderRepo  repository.OrderRepository
	notiSvc    notiService.NotificationService
}

func NewCreateOrderCommandHandler(
	transactor transactor.Transactor,
	orderRepo repository.OrderRepository,
	notiSvc notiService.NotificationService) *createOrderCommandHandler {
	return &createOrderCommandHandler{
		transactor: transactor,
		orderRepo:  orderRepo,
		notiSvc:    notiSvc,
	}
}

func (h *createOrderCommandHandler) Handle(ctx context.Context, cmd *CreateOrderCommand) (*CreateOrderCommandResult, error) {
	// ตรวจสอบ customer id
	customer, err := mediator.Send[*customercontract.GetCustomerByIDQuery, *customercontract.GetCustomerByIDQueryResult](
		ctx,
		&customercontract.GetCustomerByIDQuery{ID: cmd.CustomerID},
	)
	if err != nil {
		return nil, err
	}

	var order *model.Order
	err = h.transactor.WithinTransaction(ctx, func(ctx context.Context, registerPostCommitHook func(transactor.PostCommitHook)) error {

		// ตัดยอด credit ในตาราง customer
		if _, err := mediator.Send[*customercontract.ReserveCreditCommand, *mediator.NoResponse](
			ctx,
			&customercontract.ReserveCreditCommand{CustomerID: cmd.CustomerID, CreditAmount: cmd.OrderTotal},
		); err != nil {
			return err
		}

		// สร้าง order ใหม่
		order = model.NewOrder(cmd.CustomerID, cmd.OrderTotal)
		err := h.orderRepo.Create(ctx, order)
		if err != nil {
			logger.Log.Error(err.Error())
			return err
		}

		registerPostCommitHook(func(ctx context.Context) error {
			return h.notiSvc.SendEmail(customer.Email, "Order Created", map[string]any{
				"order_id": order.ID,
				"total":    order.OrderTotal,
			})
		})

		return nil
	})

	if err != nil {
		return nil, err
	}

	return NewCreateOrderCommandResult(order.ID), nil
}
