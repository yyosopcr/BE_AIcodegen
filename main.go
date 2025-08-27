package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
)

func main() {
	// Reference jwt package so the import is used (no-op)
	_ = jwt.New(jwt.SigningMethodHS256)

	app := fiber.New()

	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "Hello World"})
	})

	log.Fatal(app.Listen(":3000"))
}
