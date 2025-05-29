package handlers

import (
	"fmt"
	"go-mma/data/sqldb"

	"github.com/gofiber/fiber/v3"
)

type CustomerHandler struct {
	db sqldb.DBContext
}

func NewCustomerHandler(db sqldb.DBContext) *CustomerHandler {
	return &CustomerHandler{db: db}
}

func (h *CustomerHandler) CreateCustomer(c fiber.Ctx) error {
	type CreateCustomerRequest struct {
		Name   string `json:"name"`
		Credit int    `json:"credit"`
	}
	payload := CreateCustomerRequest{}

	if err := c.Bind().Body(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	fmt.Println("Received customer:", payload)

	// Validate payload fields
	if payload.Name == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "name is required"})
	}
	if payload.Credit <= 0 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "credit must be greater than 0"})
	}

	// TODO: save new customer to the database

	// Return a created response
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": "c1"})
}
