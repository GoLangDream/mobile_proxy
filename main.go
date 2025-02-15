package main

import (
	"log"

	"github.com/GoLangDream/mobile_proxy/global"
	"github.com/GoLangDream/mobile_proxy/http_server"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

func main() {
	global.InitSystem()

	app := fiber.New()

	app.Get("/ws", websocket.New(global.ClientManager.ClientRegister))

	app.All("/mobile/:client_id/*", http_server.MobilePage)

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})

	log.Fatal(app.Listen(":3000"))
}
