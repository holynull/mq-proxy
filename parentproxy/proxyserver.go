package parentproxy

import (
	"log"
	"net"
	"os"

	"github.com/holynull/my-vsock/my_vsock"
	"github.com/spf13/viper"
)

type ProxyVsockServer struct {
	logger    *log.Logger
	Config    *viper.Viper
	ProxyConn *net.Conn
}

func NewProxyServer() *ProxyVsockServer {
	viper.SetConfigName("config_server")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		log.Printf("Load config failed. %v", err)
	}
	_conf := viper.GetViper()
	return &ProxyVsockServer{
		logger: log.New(os.Stdout, "[proxy server] ", log.Ldate|log.Ltime|log.Lshortfile),
		Config: _conf,
	}
}

func (server *ProxyVsockServer) StartServer() {
	port := server.Config.GetUint32("server.port")
	go my_vsock.StartServer(port)
	server.logger.Printf("Vsock server listen on: %d", port)
	for {
		server.logger.Println("Waiting for message.")
		msg := <-my_vsock.RECV_MSG_CHAN
		server.logger.Printf("Get message: %s", string(msg.Data))
		server.logger.Printf("From: %s", msg.Conn.RemoteAddr().String())
		switch string(msg.Data) {
		case PROXY_READY:
			server.ProxyConn = &msg.Conn
		default:
			server.logger.Printf("Message is: %s", string(msg.Data))
		}
	}
}
