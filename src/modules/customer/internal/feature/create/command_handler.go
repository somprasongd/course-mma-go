package create

import (
	"context"
	"go-mma/modules/customer/domainerrors"
	"go-mma/modules/customer/internal/model"
	"go-mma/modules/customer/internal/repository"
	"go-mma/shared/common/domain"
	"go-mma/shared/common/logger"
	"go-mma/shared/common/storage/sqldb/transactor"
)

type createCustomerCommandHandler struct {
	transactor transactor.Transactor
	custRepo   repository.CustomerRepository
	dispatcher domain.DomainEventDispatcher
}

func NewCreateCustomerCommandHandler(
	transactor transactor.Transactor,
	custRepo repository.CustomerRepository,
	dispatcher domain.DomainEventDispatcher,
) *createCustomerCommandHandler {
	return &createCustomerCommandHandler{
		transactor: transactor,
		custRepo:   custRepo,
		dispatcher: dispatcher,
	}
}

func (h *createCustomerCommandHandler) Handle(ctx context.Context, cmd *CreateCustomerCommand) (*CreateCustomerCommandResult, error) {
	// ตรวจสอบ business rule/invariant
	if err := h.validateBusinessInvariant(ctx, cmd); err != nil {
		return nil, err
	}

	// แปลง Command → Model
	customer := model.NewCustomer(cmd.Email, cmd.Credit)

	// ย้ายส่วนที่ติดต่อฐานข้อมูล กับส่งอีเมลมาทำงานใน WithinTransaction
	err := h.transactor.WithinTransaction(ctx, func(ctx context.Context, registerPostCommitHook func(transactor.PostCommitHook)) error {

		// ส่งไปที่ Repository Layer เพื่อบันทึกข้อมูลลงฐานข้อมูล
		if err := h.custRepo.Create(ctx, customer); err != nil {
			// error logging
			logger.Log.Error(err.Error())
			return err
		}

		// ดึง domain events จาก customer model
		events := customer.PullDomainEvents()

		// ให้ dispatch หลัง commit แล้ว
		registerPostCommitHook(func(ctx context.Context) error {
			return h.dispatcher.Dispatch(ctx, events)
		})

		return nil
	})

	if err != nil {
		return nil, err
	}

	return NewCreateCustomerCommandResult(customer.ID), nil
}

func (h *createCustomerCommandHandler) validateBusinessInvariant(ctx context.Context, cmd *CreateCustomerCommand) error {
	// ตรวจสอบ Credit ต้องมากกว่า 0
	if cmd.Credit <= 0 {
		return domainerrors.ErrCreditValue
	}

	// ตรวจสอบ email ซ้ำ
	exists, err := h.custRepo.ExistsByEmail(ctx, cmd.Email)
	if err != nil {
		// error logging
		logger.Log.Error(err.Error())
		return err
	}

	if exists {
		return domainerrors.ErrEmailExists
	}
	return nil
}
