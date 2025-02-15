package websocket_server

import (
	"log"

	"sync"

	"github.com/gofiber/websocket/v2"
)

// Clients 存储已连接的客户端，键为客户端ID
var Clients sync.Map

func ClientRegister(c *websocket.Conn) {
	clientID, err := registerClient(c)
	if err != nil {
		log.Println("客户端注册失败:", err)
		return
	}

	c.SetCloseHandler(func(code int, text string) error {
		log.Printf("客户端 %s 关闭连接，code: %d, text: %s", clientID, code, text)
		clientUnregister(clientID)
		return nil
	})

	go func() {
		clientUnregister(clientID)
	}()
	select {} // 保持连接打开
}

func registerClient(c *websocket.Conn) (string, error) {
	_, msg, err := c.ReadMessage()
	if err != nil {
		return "", err
	}

	clientID := string(msg)
	log.Printf("新的WebSocket连接来自客户端: %s", clientID)

	Clients.Store(clientID, c)
	return clientID, nil
}

// clientUnregister 注销客户端
func clientUnregister(clientID string) {

	defer func() {
		cleanupClient(clientID)
	}()

	log.Printf("注销客户端 %s", clientID)

	clientConn, ok := Clients.Load(clientID)
	if !ok {
		log.Printf("找不到客户端 %s 的连接", clientID)
		return
	}

	conn, ok := clientConn.(*websocket.Conn)
	if !ok {
		log.Printf("类型断言失败，无法关闭客户端 %s 的连接", clientID)
		return
	}

	if err := conn.Close(); err != nil {
		log.Printf("关闭客户端 %s 的连接时发生错误: %v", clientID, err)
	} else {
		log.Printf("关闭客户端 %s 的连接", clientID)
	}
}

func cleanupClient(clientID string) {
	Clients.Delete(clientID)
	log.Printf("客户端断开连接: %s", clientID)
}
