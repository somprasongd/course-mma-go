package dto

import "errors"

type CreateCustomerRequest struct {
	Name   string `json:"name"`
	Credit int    `json:"credit"`
}

func (r *CreateCustomerRequest) Validate() error {
	var errs error
	if r.Name == "" {
		errs = errors.Join(errs, errors.New("name is required"))
	}
	if r.Credit <= 0 {
		errs = errors.Join(errs, errors.New("credit must be greater than 0"))
	}
	return errs
}
