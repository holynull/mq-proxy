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
		client := parentproxy.NewProxyClient()
		go client.StartVsockClient(uint32(cid), uint32(port))
	Loop:
		for {
			status := <-client.StatusChan
			switch status {
			case my_vsock.CONNECTED_OK:
				client.SendMessageToInside([]byte("Message request some inside service."))
			case my_vsock.VSOCK_EOF:
				break Loop

			}
		}
	case "server":
		server := parentproxy.NewProxyServer()
		go server.StartServer()
		for {
			select {
			case pStatus := <-server.ProxyStatusChan:
				if pStatus == parentproxy.PROXY_READY {
					server.BroadcastMessage([]byte("lllladfadfadfadfa"))
					server.SendMessageToNode([]byte("Send a message to node 2"), "party_node_2")
				}
			case msgFromOutside := <-server.MessageFromOutSide:
				log.Printf("Get message from outside: %s", msgFromOutside)
			}
		}
	default:
		log.Printf("No target match. %s", target)
	}
}
