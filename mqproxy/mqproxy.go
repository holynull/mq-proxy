package mqproxy

import (
	"log"
	"net"
	"os"
	"time"

	"github.com/holynull/mq-proxy/mq_network"
	"github.com/holynull/mq-proxy/mqclient"
)

type MqProxy struct {
	logger *log.Logger
	Node   *mq_network.PartyNode
}

func New(nodePartyId, queueName, addr string, conn net.Conn) *MqProxy {
	client := mqclient.New(addr)
	<-time.After(time.Second)
	node, err := mq_network.New(nodePartyId, queueName, client)
	if err != nil {
		return nil
	} else {
		go node.RunComsumer(conn)
		return &MqProxy{
			logger: log.New(os.Stdout, "[mqproxy] ", log.Ldate|log.Ltime|log.Lshortfile),
			Node:   node,
		}
	}
}

func (proxy *MqProxy) Handle(data map[string]interface{}) interface{} {
	if op, ok := data["op"]; ok {
		switch op {
		case OPTERATION_TYPE_BROADCAST:
			if _d, ok := data["data"].(string); ok {
				if err := proxy.Node.BroadcastMessage([]byte(_d)); err != nil {
					proxy.logger.Printf("Broadcast faild.  %v", err)
				}
			} else {
				proxy.logger.Printf("data is not a string. %v", data["data"])
			}
		case OPTERATION_TYPE_P2P:
			_toId, ok := data["toId"].(string)
			if !ok {
				proxy.logger.Printf("toId is not a string. %v", data["toId"])
				return nil
			}
			_d, okd := data["data"].(string)
			if !okd {
				proxy.logger.Printf("data is not a string. %v", data["data"])
				return nil
			}
			if err := proxy.Node.SendMessageToNode([]byte(_d), _toId); err != nil {
				proxy.logger.Printf("Send message to node failed. %v", err)
			}
		default:
			proxy.logger.Printf("No operation match. %s", op)
		}
	} else {
		proxy.logger.Printf("op is not a string. %v", data["op"])
	}
	return nil
}
