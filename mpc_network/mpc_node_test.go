package mpc_network

import (
	"context"
	"testing"
	"time"

	"github.com/holynull/mq-proxy/mqclient"
	amqp "github.com/rabbitmq/amqp091-go"
)

func TestMpcNode(t *testing.T) {
	addr := "amqp://guest:guest@localhost:5672/"
	partyNodeId_1 := "party_node_id_1"
	client1 := mqclient.New(partyNodeId_1, addr)
	<-time.After(time.Millisecond * 500)
	mpcNode1, err := New(partyNodeId_1, client1)
	if err != nil {
		t.Errorf("Init mpc node failed. %v", err)
		return
	}
	go runComsumer(mpcNode1.mqClient, t)

	partyNodeId_2 := "party_node_id_2"
	client2 := mqclient.New(partyNodeId_2, addr)
	<-time.After(time.Millisecond * 500)
	mpcNode2, err := New(partyNodeId_2, client2)
	if err != nil {
		t.Errorf("Init mpc node failed. %v", err)
		return
	}
	go runComsumer(mpcNode2.mqClient, t)

	partyNodeId_3 := "party_node_id_3"
	client3 := mqclient.New(partyNodeId_3, addr)
	<-time.After(time.Millisecond * 500)
	mpcNode3, err := New(partyNodeId_3, client3)
	if err != nil {
		t.Errorf("Init mpc node failed. %v", err)
		return
	}
	go runComsumer(mpcNode3.mqClient, t)

	mpcNode1.BroadCastMessage([]byte("Broadcast this message to every node."))
	mpcNode2.SendMessageToNode([]byte("Node 2 send this message to node 3"), partyNodeId_3)
	for {
		msg := <-endMessageChan
		if msg == "node 2 send this message to node 3" {
			break
		}
	}
}

var endMessageChan chan string = make(chan string)

func runComsumer(consumer *mqclient.Client, t *testing.T) {
	<-time.After(time.Second)
	deliveries, err := consumer.Consume()
	if err != nil {
		t.Errorf("Could not start consuming: %s\n", err)
		return
	}

	// This channel will receive a notification when a channel closed event
	// happens. This must be different than Client.notifyChanClose because the
	// library sends only one notification and Client.notifyChanClose already has
	// a receiver in handleReconnect().
	// Recommended to make it buffered to avoid deadlocks
	chClosedCh := make(chan *amqp.Error, 1)
	consumer.Channel.NotifyClose(chClosedCh)

	for {
		select {
		case <-context.Background().Done():
			consumer.Close()
			return

		case amqErr := <-chClosedCh:
			// This case handles the event of closed channel e.g. abnormal shutdown
			t.Errorf("AMQP Channel closed due to: %s\n", amqErr)

			deliveries, err = consumer.Consume()
			if err != nil {
				// If the AMQP channel is not ready, it will continue the loop. Next
				// iteration will enter this case because chClosedCh is closed by the
				// library
				t.Errorf("Error trying to consume, will try again")
				continue
			}

			// Re-set channel to receive notifications
			// The library closes this channel after abnormal shutdown
			chClosedCh = make(chan *amqp.Error, 1)
			consumer.Channel.NotifyClose(chClosedCh)

		case delivery := <-deliveries:
			// Ack a message every 2 seconds
			t.Logf("[%s] Received message: %s\n", consumer.QueueName, delivery.Body)
			endMessageChan <- string(delivery.Body)
			if err := delivery.Ack(false); err != nil {
				t.Errorf("Error acknowledging message: %s\n", err)
			}
			<-time.After(time.Second * 2)
		}
	}
}
