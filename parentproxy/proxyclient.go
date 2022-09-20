package parentproxy

import (
	"encoding/json"
	"log"
	"net"
	"os"

	"github.com/holynull/mq-proxy/mqproxy"
	"github.com/holynull/my-vsock/my_vsock"
	"github.com/spf13/viper"
)

type ProxyVsockClient struct {
	logger     *log.Logger
	Proxies    map[string]Proxy
	Config     *viper.Viper
	Conn       net.Conn
	StatusChan chan string
}

func NewProxyClient() *ProxyVsockClient {
	viper.SetConfigName("config_client")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		log.Printf("Load config failed. %v", err)
	}
	_conf := viper.GetViper()

	return &ProxyVsockClient{
		logger:     log.New(os.Stdout, "[proxy client] ", log.Ldate|log.Ltime|log.Lshortfile),
		Config:     _conf,
		Proxies:    make(map[string]Proxy),
		StatusChan: make(chan string),
	}
}

func (client *ProxyVsockClient) StartVsockClient(cid, port uint32) {
	go my_vsock.ConnetctServer(cid, port)
MyVsockClient:
	for {
		msg := <-my_vsock.MSG_FROM_SERVER_CHAN
		client.logger.Printf("Get message from server: %s", string(msg.Data))
		client.logger.Printf("Server: %s", msg.Conn.RemoteAddr().String())
		switch string(msg.Data) {
		case my_vsock.CONNECTED_OK:
			client.logger.Println("Vsock Connected !")
			client.Conn = msg.Conn
			client.Proxies[PROXY_TYPE_RABBITMQ] = mqproxy.New(client.Config.GetString("node.partyNodeId"), client.Config.GetString("mq.queueName"), client.Config.GetString("mq.addr"), msg.Conn)
			my_vsock.SendMsg(PROXY_READY, msg.Conn)
			client.StatusChan <- my_vsock.CONNECTED_OK
		case my_vsock.VSOCK_EOF:
			client.StatusChan <- my_vsock.VSOCK_EOF
			break MyVsockClient
		default:
			var message Message
			if err := json.Unmarshal(msg.Data, &message); err != nil {
				client.logger.Printf("Unmarshal json message failed. %v", err)
			} else {
				if client.Proxies[message.Type] != nil {
					client.Proxies[message.Type].Handle(message.Data)
				} else {
					client.logger.Printf("No match proxy type: %s", message.Type)
				}
			}
		}
	}
}

func (client *ProxyVsockClient) SendMessageToInside(data []byte) {
	my_vsock.SendMsg(string(data), client.Conn)
}
