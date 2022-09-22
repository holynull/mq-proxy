package main

import (
	"flag"
	"log"

	"github.com/holynull/mq-proxy/parentproxy"
	"github.com/holynull/my-vsock/my_vsock"
)

func main() {
	var target string
	var cid uint
	var port uint
	flag.UintVar(&cid, "cid", 0, "CID")
	flag.UintVar(&port, "port", 3000, "CID")
	flag.StringVar(&target, "target", "", "client or server")
	flag.Parse()
	switch target {
	case "client":
		// 初始化一个client
		client := parentproxy.NewProxyClient()
		// 启动client的vsock连接；vsock就绪后，启动mq的client连接；
		go client.StartVsockClient(uint32(cid), uint32(port))
	Loop:
		for {
			status := <-client.StatusChan // 接收vsock连接状态
			switch status {
			case my_vsock.CONNECTED_OK: // vsock连接成功
				// 向enclave或者容器内部发送一则消息
				client.SendMessageToInside([]byte("Message request some inside service."))
			case my_vsock.VSOCK_EOF: // vsock服务器端断开连接
				break Loop
			}
		}
	case "server":
		// 初始化服务器端；即enclave或者容器端
		server := parentproxy.NewProxyServer()
		// 启动vsock服务端
		go server.StartServer()
		for {
			select {
			case pStatus := <-server.ProxyStatusChan: // 宿主端代理的服务状态
				if pStatus == parentproxy.PROXY_READY { // 宿主端代理服务状态准备就绪
					server.BroadcastMessage([]byte("Broadcast Message!"))                         // 试着广播一个消息
					server.SendMessageToNode([]byte("Send a message to node 2!"), "party_node_2") // 向node id为party_node_2的节点发送一则消息
					server.Encrypt([]byte("Hello world!"))                                        // kms加密
				}
			case msgFromOutside := <-server.MessageFromOutSide: // 接收从宿主端发送进来的其他信息
				log.Printf("Get message from outside: %s", msgFromOutside)
			}
		}
	default:
		log.Printf("No target match. %s", target)
	}
}
