package main

import (
	"flag"
	"log"

	"github.com/holynull/mq-proxy/parentproxy"
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
		client.StartVsockClient(uint32(cid), uint32(port))
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
