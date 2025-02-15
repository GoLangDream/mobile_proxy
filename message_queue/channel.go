package message_queue

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	defaultPoolSize = 20
	channelTimeout  = 10 * time.Second
)

// ChannelPool 管理消息通道池。
type ChannelPool struct {
	pool   chan chan string // 通道池
	active sync.Map         // 活跃通道的map，key为ID
	size   int              // 通道池的大小
}

// NewChannelPool 创建一个具有默认大小的新通道池。
func NewChannelPool() *ChannelPool {
	size := defaultPoolSize

	pool := make(chan chan string, size)
	for i := 0; i < size; i++ {
		pool <- make(chan string)
	}

	return &ChannelPool{
		pool:   pool,       // 通道池
		active: sync.Map{}, // 活跃通道的map
		size:   size,       // 通道池的大小
	}
}

// GetMessageID 获取一个消息ID，用于后续监听
func (cp *ChannelPool) GetMessageID() (string, error) {
	channel := <-cp.pool
	messageID := uuid.New().String()
	cp.active.Store(messageID, channel)

	return messageID, nil
}

// ReceiveMessage 根据消息ID从队列中读取一个消息，读取完成后将通道放回池子。
func (cp *ChannelPool) ReceiveMessage(messageID string) (string, error) {
	channel, ok := cp.loadChannel(messageID)
	if !ok {
		return "", fmt.Errorf("通道 %s 未找到", messageID)
	}

	defer cp.returnChannel(messageID)

	select {
	case response := <-channel:
		return response, nil
	case <-time.After(channelTimeout):
		return "", fmt.Errorf("通道 %s 超时", messageID)
	}
}

// SendMessage 向特定通道发送消息。
func (cp *ChannelPool) SendMessage(messageID string, message string) error {
	channel, ok := cp.loadChannel(messageID)
	if !ok {
		return fmt.Errorf("通道 %s 未找到", messageID)
	}

	select {
	case channel <- message: // 发送消息
		return nil
	case <-time.After(channelTimeout):
		return fmt.Errorf("向通道 %s 发送消息超时", messageID)
	}
}

// returnChannel 将通道返回到池中。
func (cp *ChannelPool) returnChannel(id string) {
	channel, ok := cp.loadChannel(id)
	if !ok {
		return // 未找到通道，可能已超时
	}

	cp.active.Delete(id)

	select {
	case cp.pool <- channel:
		// 通道已返回到池中
	default:
		// 池已满，丢弃该通道
		close(channel)
		log.Println("通道池已满，正在丢弃通道")
	}
}

// loadChannel 从活跃通道的map中加载通道。
func (cp *ChannelPool) loadChannel(id string) (chan string, bool) {
	channelInterface, ok := cp.active.Load(id)
	if !ok {
		return nil, false
	}

	channel, ok := channelInterface.(chan string)
	if !ok {
		return nil, false
	}

	return channel, true
}
