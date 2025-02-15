package main

import (
	"log"

	"github.com/GoLangDream/mobile_proxy/http_server"
	"github.com/GoLangDream/mobile_proxy/websocket_server"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

func main() {
	app := fiber.New()

	app.Get("/ws", websocket.New(websocket_server.ClientRegister))

	app.All("/mobile/:client_id/*", http_server.MobilePage)

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})

	log.Fatal(app.Listen(":3000"))
}
