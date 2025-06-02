package service

import (
	"context"
	"go-mma/dto"
	"go-mma/model"
	"go-mma/repository"
	"go-mma/util/errs"
	"go-mma/util/logger"
)

var (
	ErrEmailExists = errs.ConflictError("email already exists")
)

type CustomerService struct {
	custRepo *repository.CustomerRepository
	notiSvc  *NotificationService
}

func NewCustomerService(custRepo *repository.CustomerRepository,
	notiSvc *NotificationService,
) *CustomerService {
	return &CustomerService{
		custRepo: custRepo,
		notiSvc:  notiSvc,
	}
}

func (s *CustomerService) CreateCustomer(ctx context.Context, req *dto.CreateCustomerRequest) (*dto.CreateCustomerResponse, error) {
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

	// ส่งไปที่ Repository Layer เพื่อบันทึกข้อมูลลงฐานข้อมูล
	if err := s.custRepo.Create(ctx, customer); err != nil {
		// error logging
		logger.Log.Error(err.Error())
		return nil, err
	}

	// ส่งอีเมลต้อนรับ
	s.notiSvc.SendEmail(customer.Email, "Welcome to our service!", map[string]any{
		"message": "Thank you for joining us! We are excited to have you as a member.",
	})

	// สร้าง DTO Response
	resp := dto.NewCreateCustomerResponse(customer.ID)
	return resp, nil
}
