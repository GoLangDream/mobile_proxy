package http_server

import (
	"log"

	"github.com/GoLangDream/mobile_proxy/global"
	"github.com/gofiber/fiber/v2"
)

func MobilePage(c *fiber.Ctx) error {
	clientID := c.Params("client_id")
	log.Printf("Received request for client ID: %s", clientID)

	message, err := packageHTTPRequest(c)
	if err != nil {
		return err
	}

	log.Printf("Relaying to WebSocket client %s", clientID)

	messageID, err := global.ClientManager.SendMessage(clientID, message)
	if err != nil {
		return err
	}

	response, err := global.ClientManager.ReceiveMessage(messageID)
	if err != nil {
		return err
	}

	return processMobileResponse(c, []byte(response))
}
