package http_server

import (
	"encoding/json"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

// packageHTTPRequest 构造要发送给 WebSocket 客户端的 HTTP 请求消息
func packageHTTPRequest(c *fiber.Ctx) (string, error) {
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
		log.Println("JSON 序列化错误:", err)
		return "", c.Status(fiber.StatusInternalServerError).SendString("Failed to marshal JSON")
	}

	return string(messageBytes), nil
}

// relayToMobile 将消息转发到 WebSocket 连接并等待响应
func relayToMobile(conn *websocket.Conn, message string) ([]byte, error) {
	err := conn.WriteMessage(websocket.TextMessage, []byte(message))
	if err != nil {
		log.Println("WebSocket 写入错误:", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to relay to WebSocket")
	}

	messageType, response, err := conn.ReadMessage()
	if err != nil {
		log.Println("WebSocket 读取错误:", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to read response from WebSocket")
	}

	if messageType == websocket.TextMessage {
		log.Printf("从 WebSocket 客户端接收到文本消息: %s", response)
	} else if messageType == websocket.BinaryMessage {
		log.Printf("从 WebSocket 客户端接收到二进制消息: %v", response)
	} else {
		log.Printf("从 WebSocket 客户端接收到未知消息类型")
	}

	return response, nil
}

// ProcessMobileResponse 处理从 WebSocket 客户端收到的响应
func ProcessMobileResponse(c *fiber.Ctx, response []byte) error {
	log.Printf("从 WebSocket 客户端接收到响应")

	var responseData map[string]interface{}
	err := json.Unmarshal(response, &responseData)
	if err != nil {
		log.Println("JSON 反序列化错误:", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to unmarshal JSON response")
	}

	httpCode, ok := responseData["http_code"].(float64)
	if !ok {
		log.Println("http_code 未找到或不是一个数字")
		return c.Status(fiber.StatusInternalServerError).SendString("http_code not found or invalid")
	}
	log.Printf("响应数据: %s", responseData["body"])
	contentType, _ := responseData["Content-Type"].(string) // Ignore the error, just proceed without setting it if missing.
	bodyString, _ := responseData["body"].(string)          // Handle missing body gracefully

	if contentType != "" {
		c.Set("Content-Type", contentType)
	}

	c.Status(int(httpCode))

	return c.SendString(bodyString)
}
