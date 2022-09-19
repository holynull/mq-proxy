package parentproxy

import (
	"encoding/json"

	"github.com/holynull/my-vsock/my_vsock"
)

func (server *ProxyVsockServer) BroadcastMessage(data []byte) {
	msg := make(map[string]interface{}, 0)
	msg["type"] = PROXY_TYPE_RABBITMQ
	_d := make(map[string]interface{}, 0)
	_d["data"] = string(data)
	msg["data"] = _d
	if b, err := json.Marshal(msg); err != nil {
		server.logger.Printf("Marshal message failed. %v", err)
	} else {
		my_vsock.SendMsg(string(b), *server.ProxyConn)
	}
}

func (server *ProxyVsockServer) SendMessageToNode(data []byte, toNodeId string) {
	msg := make(map[string]interface{}, 0)
	msg["type"] = PROXY_TYPE_RABBITMQ
	_d := make(map[string]interface{}, 0)
	_d["data"] = string(data)
	_d["toId"] = toNodeId
	msg["data"] = _d
	if b, err := json.Marshal(msg); err != nil {
		server.logger.Printf("Marshal message failed. %v", err)
	} else {
		my_vsock.SendMsg(string(b), *server.ProxyConn)
	}
}
