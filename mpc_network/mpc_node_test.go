package mpc_network

import (
	"testing"
	"time"

	"github.com/holynull/mq-proxy/mqclient"
)

func TestMpcNode(t *testing.T) {
	addr := "amqp://guest:guest@localhost:5672/"
	partyNodeId_1 := "party_node_id_1"
	queueName_1 := "queueName_1"
	client1 := mqclient.New(addr)
	<-time.After(time.Millisecond * 500)
	mpcNode1, err := New(partyNodeId_1, queueName_1, client1)
	if err != nil {
		t.Errorf("Init mpc node failed. %v", err)
		return
	}
	go mpcNode1.RunComsumer(nil)

	partyNodeId_2 := "party_node_id_2"
	queueName_2 := "queueName_2"
	client2 := mqclient.New(addr)
	<-time.After(time.Millisecond * 500)
	mpcNode2, err := New(partyNodeId_2, queueName_2, client2)
	if err != nil {
		t.Errorf("Init mpc node failed. %v", err)
		return
	}
	go mpcNode2.RunComsumer(nil)

	partyNodeId_3 := "party_node_id_3"
	queueName_3 := "queueName_3"
	client3 := mqclient.New(addr)
	<-time.After(time.Millisecond * 500)
	mpcNode3, err := New(partyNodeId_3, queueName_3, client3)
	if err != nil {
		t.Errorf("Init mpc node failed. %v", err)
		return
	}
	go mpcNode3.RunComsumer(nil)

	mpcNode1.BroadcastMessage([]byte("Broadcast this message to every node."))
	mpcNode2.SendMessageToNode([]byte("Node 2 send this message to node 3"), partyNodeId_3)
	for {
		msg := <-endMessageChan
		if msg == "Node 2 send this message to node 3" {
			break
		}
	}
}

var endMessageChan chan string = make(chan string)
