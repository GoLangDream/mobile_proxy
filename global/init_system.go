package global

import (
	"github.com/GoLangDream/mobile_proxy/websocket_server"
)

var ClientManager *websocket_server.ClientManager

func InitSystem() {
	ClientManager = websocket_server.NewClientManager()
}
