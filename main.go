package main

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

// clients stores connected clients, key is client ID
var clients sync.Map

func main() {
	app := fiber.New()

	app.Get("/ws", websocket.New(handleWebSocket))

	app.All("/mobile/:client_id/*", handleMobileRequest)

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})

	log.Fatal(app.Listen(":3000"))
}

func handleWebSocket(c *websocket.Conn) {
	var clientID string

	// Read client ID from the first message
	_, msg, err := c.ReadMessage()
	if err != nil {
		log.Println("read client ID:", err)
		return
	}
	clientID = string(msg)
	log.Printf("New WebSocket connection from client: %s", clientID)

	clients.Store(clientID, c)

	defer func() {
		clients.Delete(clientID)
		c.Close()
		log.Printf("Disconnected client: %s", clientID)
	}()

	// Keep the connection alive. Messages are sent from the /mobile endpoint.
	select {}
}

func handleMobileRequest(c *fiber.Ctx) error {
	clientID := c.Params("client_id")
	log.Printf("Received request for client ID: %s", clientID)

	clientConn, ok := clients.Load(clientID)
	if !ok {
		return c.Status(fiber.StatusBadRequest).SendString("Client not found")
	}

	message, err := constructMessage(c)
	if err != nil {
		return err
	}

	log.Printf("Relaying to WebSocket client %s", clientID)

	response, err := relayMessage(clientConn.(*websocket.Conn), message)
	if err != nil {
		return err
	}

	return processResponse(c, response)
}

func constructMessage(c *fiber.Ctx) (string, error) {
	requestPath := "/" + c.Params("*")
	headers := make(map[string]string)

	c.Request().Header.VisitAll(func(key, value []byte) {
		headerKey := string(key)
		if headerKey != ":path" {
			headers[headerKey] = string(value)
		}
	})

	bodyBytes := c.Body()
	body := string(bodyBytes)

	messageData := map[string]interface{}{
		"path":    requestPath,
		"headers": headers,
		"body":    body,
		"method":  c.Method(),
	}

	messageBytes, err := json.Marshal(messageData)
	if err != nil {
		log.Println("JSON marshal error:", err)
		return "", c.Status(fiber.StatusInternalServerError).SendString("Failed to marshal JSON")
	}

	return string(messageBytes), nil
}

func relayMessage(conn *websocket.Conn, message string) ([]byte, error) {
	err := conn.WriteMessage(websocket.TextMessage, []byte(message))
	if err != nil {
		log.Println("websocket write error:", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to relay to WebSocket")
	}

	_, response, err := conn.ReadMessage()
	if err != nil {
		log.Println("websocket read error:", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to read response from WebSocket")
	}

	return response, nil
}

func processResponse(c *fiber.Ctx, response []byte) error {
	log.Printf("Received response from WebSocket client")

	var responseData map[string]interface{}
	err := json.Unmarshal(response, &responseData)
	if err != nil {
		log.Println("JSON unmarshal error:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to unmarshal JSON response")
	}

	httpCode, ok := responseData["http_code"].(float64)
	if !ok {
		log.Println("http_code not found or not a number")
		return c.Status(fiber.StatusInternalServerError).SendString("http_code not found or invalid")
	}
	log.Printf("Response data: %s", responseData["body"])
	contentType, _ := responseData["Content-Type"].(string) // Ignore the error, just proceed without setting it if missing.
	bodyString, _ := responseData["body"].(string)          // Handle missing body gracefully

	if contentType != "" {
		c.Set("Content-Type", contentType)
	}

	c.Status(int(httpCode))

	return c.Send([]byte(bodyString))
}
