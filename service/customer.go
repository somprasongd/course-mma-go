package service

import (
	"context"
	"errors"
	"go-mma/dto"
	"go-mma/model"
	"go-mma/repository"
	"go-mma/util/logger"
)

type CustomerService struct {
	custRepo *repository.CustomerRepository
}

func NewCustomerService(custRepo *repository.CustomerRepository) *CustomerService {
	return &CustomerService{
		custRepo: custRepo,
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
		return nil, errors.New("email already exists")
	}

	// แปลง DTO → Model
	customer = model.NewCustomer(req.Email, req.Credit)

	// ส่งไปที่ Repository Layer เพื่อบันทึกข้อมูลลงฐานข้อมูล
	if err := s.custRepo.Create(ctx, customer); err != nil {
		// error logging
		logger.Log.Error(err.Error())
		return nil, err
	}

	// สร้าง DTO Response
	resp := dto.NewCreateCustomerResponse(customer.ID)
	return resp, nil
}
