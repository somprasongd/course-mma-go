package handler

import (
	"fmt"

	"github.com/gofiber/fiber/v3"
)

type OrderHandler struct {
}

func NewOrderHandler() *OrderHandler {
	return &OrderHandler{}
}

func (h *OrderHandler) CreateOrder(c fiber.Ctx) error {
	type CreateOrderRequest struct {
		CustomerID string `json:"customer_id"`
		OrderTotal int    `json:"order_total"`
	}
	payload := CreateOrderRequest{}

	if err := c.Bind().Body(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	fmt.Println("Received Order:", payload)

	// Validate payload fields
	if payload.CustomerID == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "customer_id is required"})
	}
	if payload.OrderTotal <= 0 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "order_total must be greater than 0"})
	}

	// TODO: get customer by ID from the database
	// customer := getCustomer(order.CustomerID)
	// if customer == nil {
	// 	return return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "the customer with given id was not found"})
	// }

	// TODO: check credit balance of the customer
	// if credit < payload.OrderTotal {
	// 	return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "insufficient credit"})
	// }

	// TODO: reserve credit for the customer

	// TODO: update customer's credit balance in the database

	// TODO: save new Order to the database

	// Return a created response
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": "o1"})
}

func (h *OrderHandler) CancelOrder(c fiber.Ctx) error {
	// Implement the logic to cancel an order
	orderID := c.Params("orderID")
	if orderID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid order id"})
	}

	fmt.Println("Cancelling order:", orderID)

	// TODO: get order details from the database
	// order := getOrder(orderID)
	// if order == nil {
	// 	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "the order with given id was not found"})
	// }

	// TODO: get cutomer details from the database
	// customer := getCustomer(order.CustomerID)
	// if customer == nil {
	// 	return return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "the customer with given id was not found"})
	// }

	// TODO: release credit limit for the customer
	// creditLimit += CreateOrderRequest.OrderTotal
	// TODO: save the customer details to the database

	// TODO: update the order status in the database

	return c.SendStatus(fiber.StatusNoContent)
}
