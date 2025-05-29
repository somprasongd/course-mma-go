package handlers

import (
	"context"
	"fmt"
	"go-mma/data/sqldb"
	"time"

	"github.com/gofiber/fiber/v3"
)

type CustomerHandler struct {
	dbCtx sqldb.DBContext
}

func NewCustomerHandler(db sqldb.DBContext) *CustomerHandler {
	return &CustomerHandler{dbCtx: db}
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

	// save new customer to the database
	sql := `INSERT INTO customers (name, credit) VALUES ($1, $2) RETURNING id`
	ctx, cancel := context.WithTimeout(c.Context(), 30*time.Second)
	defer cancel()

	var id int
	if err := h.dbCtx.DB().QueryRowContext(ctx, sql, payload.Name, payload.Credit).Scan(&id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Return a created response
	type CreateCustomerResponse struct {
		ID int `json:"id"`
	}
	resp := &CreateCustomerResponse{ID: id}
	return c.Status(fiber.StatusCreated).JSON(resp)
}
