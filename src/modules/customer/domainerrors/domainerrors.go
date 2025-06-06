package domainerrors

import "go-mma/shared/common/errs"

var (
	ErrCreditValue        = errs.BusinessRuleError("credit must be greater than 0")
	ErrEmailExists        = errs.ConflictError("email already exists")
	ErrCustomerNotFound   = errs.ResourceNotFoundError("the customer with given id was not found")
	ErrInsufficientCredit = errs.BusinessRuleError("insufficient credit")
)
