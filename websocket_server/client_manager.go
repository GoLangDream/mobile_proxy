package websocket_server

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	"github.com/GoLangDream/mobile_proxy/message_queue"
	"github.com/gofiber/websocket/v2"
)

const maxClients = 200

// ClientManager 管理 WebSocket 客户端连接.
type ClientManager struct {
	clients     sync.Map     // 存储连接的客户端 (clientID -> websocket.Conn).
	queueLength atomic.Int32 // 跟踪连接的客户端数量.
	msgQueue    *message_queue.ChannelPool
}

// NewClientManager 创建一个新的 ClientManager 实例。
func NewClientManager() *ClientManager {
	return &ClientManager{
		clients:     sync.Map{},
		queueLength: atomic.Int32{},
		msgQueue:    message_queue.NewChannelPool(),
	}
}

// ClientRegister 注册一个新的客户端连接.
func (cm *ClientManager) ClientRegister(connect *websocket.Conn) {
	if cm.queueLength.Load() >= maxClients {
		log.Println("已达到最大客户端限制。连接被拒绝。")
		connect.WriteMessage(websocket.TextMessage, []byte("服务器已满。请稍后再试。"))
		connect.Close()
		return
	}

	clientID, err := cm.registerNewClient(connect)
	if err != nil {
		log.Println("客户端注册失败:", err)
		return
	}

	cm.setupCloseHandler(connect, clientID)
	cm.startClientListener(connect, clientID)
	cm.queueLength.Add(1) // 增加客户端计数.
}

type clientMessage struct {
	MessageID string `json:"message_id"`
	Data      string `json:"data"`
}

// SendMessageAndReceive 发送消息到客户端.
func (cm *ClientManager) SendMessage(clientID string, message string) (string, error) {
	conn, err := cm.getClientConnection(clientID)
	if err != nil {
		return "", fmt.Errorf("获取客户端连接失败: %w", err)
	}

	messageID, err := cm.msgQueue.GetMessageID()
	if err != nil {
		return "", fmt.Errorf("获取消息ID失败: %w", err)
	}

	clientMessage := clientMessage{
		MessageID: messageID,
		Data:      message,
	}

	err = cm.sendMessageToClient(conn, clientMessage)
	if err != nil {
		return "", fmt.Errorf("发送消息到客户端 %s 失败: %w", clientID, err)
	}

	return messageID, nil
}

// ReceiveMessage 接收客户端的消息.
func (cm *ClientManager) ReceiveMessage(messageID string) (string, error) {
	message, err := cm.msgQueue.ReceiveMessage(messageID)
	if err != nil {
		return "", fmt.Errorf("接收消息失败: %w", err)
	}
	return message, nil
}

// unregisterClient 注销一个客户端.
func (cm *ClientManager) unregisterClient(clientID string) {
	log.Printf("正在注销客户端 %s", clientID)

	clientConn, ok := cm.clients.Load(clientID)
	if !ok {
		log.Printf("客户端 %s 未找到", clientID)
		return
	}

	clientConn.(*websocket.Conn).Close()
	cm.clients.Delete(clientID)
	cm.queueLength.Add(-1) // 减少客户端计数.
}

// registerNewClient 注册一个新的客户端.
func (cm *ClientManager) registerNewClient(connect *websocket.Conn) (string, error) {
	clientID, err := cm.getClientID(connect)
	if err != nil {
		return "", fmt.Errorf("获取客户端ID失败: %w", err)
	}

	log.Printf("来自客户端的新 WebSocket 连接: %s", clientID)
	// cm.unregisterClient(clientID) // 确保没有重复的客户端 ID.
	cm.clients.Store(clientID, connect)

	return clientID, nil
}

// getClientID 从 WebSocket 连接中读取客户端 ID.
func (cm *ClientManager) getClientID(connect *websocket.Conn) (string, error) {
	_, msg, err := connect.ReadMessage()
	if err != nil {
		return "", fmt.Errorf("读取客户端ID失败: %w", err)
	}
	return string(msg), nil
}

// setupCloseHandler 为 WebSocket 连接设置关闭处理程序.
func (cm *ClientManager) setupCloseHandler(connect *websocket.Conn, clientID string) {
	connect.SetCloseHandler(func(code int, text string) error {
		log.Printf("客户端 %s 关闭连接, code: %d, text: %s", clientID, code, text)
		cm.unregisterClient(clientID)
		return nil
	})
}

// startClientListener 启动一个监听器，用于监听客户端消息.
func (cm *ClientManager) startClientListener(connect *websocket.Conn, clientID string) {
	defer cm.unregisterClient(clientID) // 确保在退出时注销客户端.

	for {
		messageType, p, err := connect.ReadMessage()
		if err != nil {
			log.Printf("读取客户端 %s 的消息时出错: %v", clientID, err)
			return
		}

		cm.processClientMessage(clientID, messageType, p)
	}
}

// processClientMessage 处理从客户端收到的消息.
func (cm *ClientManager) processClientMessage(clientID string, messageType int, payload []byte) {
	switch messageType {
	case websocket.TextMessage:
		log.Printf("客户端 %s 发送消息: %s", clientID, string(payload))
		var responseMessage clientMessage
		err := json.Unmarshal(payload, &responseMessage)
		if err != nil {
			log.Printf("解析客户端 %s 的消息体失败: %v", clientID, err)
			return
		}

		cm.msgQueue.SendMessage(responseMessage.MessageID, responseMessage.Data)

	default:
		log.Printf("客户端 %s 发送了未知消息类型", clientID)
	}
}

// getClientConnection 检索客户端的 WebSocket 连接.
func (cm *ClientManager) getClientConnection(clientID string) (*websocket.Conn, error) {
	clientConn, ok := cm.clients.Load(clientID)
	if !ok {
		return nil, fmt.Errorf("找不到客户端 %s", clientID)
	}

	conn, ok := clientConn.(*websocket.Conn)
	if !ok {
		return nil, fmt.Errorf("clientID 的类型断言失败: %s", clientID)
	}
	return conn, nil
}

// sendMessageToClient 发送消息到客户端.
func (cm *ClientManager) sendMessageToClient(conn *websocket.Conn, clientMessage clientMessage) error {
	messageBytes, err := json.Marshal(clientMessage)
	if err != nil {
		log.Printf("JSON 序列化错误: %v", err)
		return err
	}
	return conn.WriteMessage(websocket.TextMessage, messageBytes)
}
