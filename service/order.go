package service

import (
	"context"
	"go-mma/dto"
	"go-mma/model"
	"go-mma/repository"
	"go-mma/util/errs"
	"go-mma/util/logger"
	"go-mma/util/transactor"
)

var (
	ErrNoCustomerID = errs.ResourceNotFoundError("the customer with given id was not found")
	ErrNoOrderID    = errs.ResourceNotFoundError("the order with given id was not found")
)

type OrderService struct {
	transactor transactor.Transactor
	custRepo   *repository.CustomerRepository
	orderRepo  *repository.OrderRepository
	notiSvc    *NotificationService
}

func NewOrderService(
	transactor transactor.Transactor,
	custRepo *repository.CustomerRepository,
	orderRepo *repository.OrderRepository,
	notiSvc *NotificationService) *OrderService {
	return &OrderService{
		transactor: transactor,
		custRepo:   custRepo,
		orderRepo:  orderRepo,
		notiSvc:    notiSvc,
	}
}

func (s *OrderService) CreateOrder(ctx context.Context, req *dto.CreateOrderRequest) (int, error) {
	// Business Logic Rule: ตรวจสอบ customer id
	customer, err := s.custRepo.FindByID(ctx, req.CustomerID)
	if err != nil {
		logger.Log.Error(err.Error())
		return 0, err
	}

	if customer == nil {
		return 0, ErrNoCustomerID
	}

	// Business Logic Rule: ตัดยอด credit ถ้าไม่พอให้ error
	if err := customer.ReserveCredit(req.OrderTotal); err != nil {
		return 0, err
	}

	// ย้ายส่วนที่ติดต่อฐานข้อมูล กับส่งอีเมลมาทำงานใน WithinTransaction
	var order *model.Order
	err = s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {

		// ตัดยอด credit ในตาราง customer
		if err := s.custRepo.UpdateCredit(ctx, customer); err != nil {
			logger.Log.Error(err.Error())
			return err
		}

		// สร้าง order ใหม่
		order = model.NewOrder(req.CustomerID, req.OrderTotal)
		err = s.orderRepo.Create(ctx, order)
		if err != nil {
			logger.Log.Error(err.Error())
			return err
		}

		err = s.notiSvc.SendEmail(customer.Email, "Order Created", map[string]any{
			"order_id": order.ID,
			"total":    order.OrderTotal,
		})
		if err != nil {
			logger.Log.Error(err.Error())
			return err
		}

		return nil
	})

	if err != nil {
		return 0, err
	}

	return order.ID, nil
}

func (s *OrderService) CancelOrder(ctx context.Context, id int) error {
	// Business Logic Rule: ตรวจสอบ order id
	order, err := s.orderRepo.FindByID(ctx, id)
	if err != nil {
		logger.Log.Error(err.Error())
		return err
	}

	if order == nil {
		return ErrNoOrderID
	}

	// ยกเลิก order
	if err := s.orderRepo.Cancel(ctx, order.ID); err != nil {
		logger.Log.Error(err.Error())
		return err
	}

	// Business Logic Rule: ตรวจสอบ customer id
	customer, err := s.custRepo.FindByID(ctx, order.CustomerID)
	if err != nil {
		logger.Log.Error(err.Error())
		return err
	}

	if customer == nil {
		return ErrNoCustomerID
	}

	// Business Logic: คืนยอด credit
	customer.ReleaseCredit(order.OrderTotal)

	// บันทึกการคืนยอด credit
	if err := s.custRepo.UpdateCredit(ctx, customer); err != nil {
		logger.Log.Error(err.Error())
		return err
	}

	return nil
}
