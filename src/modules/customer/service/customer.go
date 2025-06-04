package service

import (
	"context"
	"go-mma/modules/customer/dto"
	"go-mma/modules/customer/internal/model"
	"go-mma/modules/customer/internal/repository"
	"go-mma/shared/common/errs"
	"go-mma/shared/common/logger"
	"go-mma/shared/common/storage/sqldb/transactor"

	notiService "go-mma/modules/notification/service"
)

var (
	ErrEmailExists                  = errs.ConflictError("email already exists")
	ErrCustomerNotFound             = errs.ResourceNotFoundError("the customer with given id was not found")
	ErrOrderTotalExceedsCreditLimit = errs.BusinessRuleError("order total exceeds credit limit")
)

type CustomerService interface {
	CreateCustomer(ctx context.Context, req *dto.CreateCustomerRequest) (*dto.CreateCustomerResponse, error)
	GetCustomerByID(ctx context.Context, id int) (*dto.Customer, error)
	ReserveCredit(ctx context.Context, id int, amount int) error
	ReleaseCredit(ctx context.Context, id int, amount int) error
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

func (s *customerService) GetCustomerByID(ctx context.Context, id int) (*dto.Customer, error) {
	customer, err := s.custRepo.FindByID(ctx, id)
	if err != nil {
		// error logging
		logger.Log.Error(err.Error())
		return nil, err
	}

	if customer == nil {
		return nil, ErrCustomerNotFound
	}

	// สร้าง DTO Response
	return &dto.Customer{
		ID:     customer.ID,
		Email:  customer.Email,
		Credit: customer.Credit,
	}, nil
}

func (s *customerService) ReserveCredit(ctx context.Context, id int, amount int) error {
	customer, err := s.custRepo.FindByID(ctx, id)
	if err != nil {
		logger.Log.Error(err.Error())
		return err
	}

	if customer == nil {
		return ErrCustomerNotFound
	}

	if err := customer.ReserveCredit(amount); err != nil {
		return ErrOrderTotalExceedsCreditLimit
	}

	if err := s.custRepo.UpdateCredit(ctx, customer); err != nil {
		logger.Log.Error(err.Error())
		return err
	}

	return nil
}

func (s *customerService) ReleaseCredit(ctx context.Context, id int, amount int) error {
	customer, err := s.custRepo.FindByID(ctx, id)
	if err != nil {
		logger.Log.Error(err.Error())
		return err
	}

	if customer == nil {
		return ErrCustomerNotFound
	}

	customer.ReleaseCredit(amount)

	if err := s.custRepo.UpdateCredit(ctx, customer); err != nil {
		logger.Log.Error(err.Error())
		return err
	}

	return nil
}
