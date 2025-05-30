package service

import (
	"context"
	"go-mma/dto"
	"go-mma/model"
	"go-mma/repository"
	"log"
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
	// create model
	customer := model.NewCustomer(req.Name, req.Credit)

	// save to database
	if err := s.custRepo.Create(ctx, customer); err != nil {
		log.Println(err)
		return nil, err
	}

	// create response
	resp := dto.NewCreateCustomerResponse(customer.ID)
	return resp, nil
}
