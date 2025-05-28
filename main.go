package main

import (
	"fmt"
	"go-mma/config"
	"log"

	"github.com/gofiber/fiber/v3"
)

func main() {
	config, err := config.Load()
	if err != nil {
		log.Panic(err)
	}

	app := fiber.New()

	app.Get("/", func(c fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})

	app.Listen(fmt.Sprintf(":%d", config.HTTPPort))
}
