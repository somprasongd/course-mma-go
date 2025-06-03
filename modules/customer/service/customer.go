package service

import (
	"context"
	"go-mma/modules/customer/dto"
	"go-mma/modules/customer/model"
	"go-mma/modules/customer/repository"
	"go-mma/util/errs"
	"go-mma/util/logger"
	"go-mma/util/transactor"

	notiService "go-mma/modules/notification/service"
)

var (
	ErrEmailExists = errs.ConflictError("email already exists")
)

type CustomerService interface {
	CreateCustomer(ctx context.Context, req *dto.CreateCustomerRequest) (*dto.CreateCustomerResponse, error)
}

type customerService struct {
	transactor transactor.Transactor
	custRepo   repository.CustomerRepository
	notiSvc    notiService.NotificationService
}

func NewCustomerService(
	transactor transactor.Transactor,
	custRepo repository.CustomerRepository,
	notiSvc notiService.NotificationService,
) CustomerService {
	return &customerService{
		transactor: transactor,
		custRepo:   custRepo,
		notiSvc:    notiSvc,
	}
}

func (s *customerService) CreateCustomer(ctx context.Context, req *dto.CreateCustomerRequest) (*dto.CreateCustomerResponse, error) {
	// Business Logic Rule: ตรวจสอบ email ซ้ำ
	customer, err := s.custRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		// error logging
		logger.Log.Error(err.Error())
		return nil, err
	}

	if customer != nil {
		return nil, ErrEmailExists
	}

	// แปลง DTO → Model
	customer = model.NewCustomer(req.Email, req.Credit)

	// ย้ายส่วนที่ติดต่อฐานข้อมูล กับส่งอีเมลมาทำงานใน WithinTransaction
	err = s.transactor.WithinTransaction(ctx, func(ctx context.Context) error {

		// ส่งไปที่ Repository Layer เพื่อบันทึกข้อมูลลงฐานข้อมูล
		if err := s.custRepo.Create(ctx, customer); err != nil {
			// error logging
			logger.Log.Error(err.Error())
			return err
		}

		// ส่งอีเมลต้อนรับ
		if err := s.notiSvc.SendEmail(customer.Email, "Welcome to our service!", map[string]any{
			"message": "Thank you for joining us! We are excited to have you as a member.",
		}); err != nil {
			// error logging
			logger.Log.Error(err.Error())
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// สร้าง DTO Response
	resp := dto.NewCreateCustomerResponse(customer.ID)
	return resp, nil
}
