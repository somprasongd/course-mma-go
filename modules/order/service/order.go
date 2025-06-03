package service

import (
	"context"
	"go-mma/modules/order/dto"
	"go-mma/modules/order/model"
	"go-mma/modules/order/repository"
	"go-mma/util/errs"
	"go-mma/util/logger"
	"go-mma/util/transactor"

	custService "go-mma/modules/customer/service"
	notiService "go-mma/modules/notification/service"
)

var (
	ErrNoOrderID = errs.ResourceNotFoundError("the order with given id was not found")
)

type OrderService interface {
	CreateOrder(ctx context.Context, req *dto.CreateOrderRequest) (*dto.CreateOrderResponse, error)
	CancelOrder(ctx context.Context, id int) error
}

type orderService struct {
	transactor transactor.Transactor
	custSvc    custService.CustomerService
	orderRepo  repository.OrderRepository
	notiSvc    notiService.NotificationService
}

func NewOrderService(
	transactor transactor.Transactor,
	custSvc custService.CustomerService,
	orderRepo repository.OrderRepository,
	notiSvc notiService.NotificationService) OrderService {
	return &orderService{
		transactor: transactor,
		custSvc:    custSvc,
		orderRepo:  orderRepo,
		notiSvc:    notiSvc,
	}
}

func (s *orderService) CreateOrder(ctx context.Context, req *dto.CreateOrderRequest) (*dto.CreateOrderResponse, error) {
	// Business Logic Rule: ตรวจสอบ customer id
	customer, err := s.custSvc.GetCustomerByID(ctx, req.CustomerID)
	if err != nil {
		return nil, err
	}

	// ย้ายส่วนที่ติดต่อฐานข้อมูล กับส่งอีเมลมาทำงานใน WithinTransaction
	var order *model.Order
	err = s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {

		// ตัดยอด credit ในตาราง customer
		if err := s.custSvc.ReserveCredit(ctx, customer.ID, req.OrderTotal); err != nil {
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
		return nil, err
	}

	// สร้าง DTO Response
	resp := dto.NewCreateOrderResponse(order.ID)
	return resp, nil
}

func (s *orderService) CancelOrder(ctx context.Context, id int) error {
	// Business Logic Rule: ตรวจสอบ order id
	order, err := s.orderRepo.FindByID(ctx, id)
	if err != nil {
		logger.Log.Error(err.Error())
		return err
	}

	if order == nil {
		return ErrNoOrderID
	}

	err = s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {

		// ยกเลิก order
		if err := s.orderRepo.Cancel(ctx, order.ID); err != nil {
			logger.Log.Error(err.Error())
			return err
		}

		// Business Logic: คืนยอด credit
		err := s.custSvc.ReleaseCredit(ctx, order.CustomerID, order.OrderTotal)
		if err != nil {
			return err
		}

		return nil
	})

	return err
}
