package http_server

import (
	"log"

	"github.com/GoLangDream/mobile_proxy/websocket_server"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

func MobilePage(c *fiber.Ctx) error {
	clientID := c.Params("client_id")
	log.Printf("Received request for client ID: %s", clientID)

	clientConn, ok := websocket_server.Clients.Load(clientID)
	if !ok {
		return c.Status(fiber.StatusBadRequest).SendString("Client not found")
	}

	message, err := packageHTTPRequest(c)
	if err != nil {
		return err
	}

	log.Printf("Relaying to WebSocket client %s", clientID)

	websocket_response, err := relayToMobile(clientConn.(*websocket.Conn), message)
	if err != nil {
		return err
	}

	return ProcessMobileResponse(c, websocket_response)
}
