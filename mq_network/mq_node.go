package mq_network

import (
	"context"
	"log"
	"net"
	"os"
	"time"

	"github.com/holynull/mq-proxy/mqclient"
	"github.com/holynull/my-vsock/my_vsock"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	BROADCAST_EXCHANGE_NAME = "mpc_broadcast_exchange"
	P2P_EXCHANGE_NAME       = "mpc_p2p_exchange"
)

type PartyNode struct {
	logger      *log.Logger
	NodePartyId string
	QueueName   string
	mqClient    *mqclient.Client
	MessageChan chan string
}

func New(nodePartyId string, queueName string, client *mqclient.Client) (*PartyNode, error) {
	if _, err := client.Channel.QueueDeclare(
		queueName,
		false,
		false,
		false,
		false,
		nil,
	); err != nil {
		return nil, err
	}
	if err := client.Channel.ExchangeDeclare(BROADCAST_EXCHANGE_NAME, "fanout", false, false, false, false, nil); err != nil {
		return nil, err
	}
	if err := client.Channel.ExchangeDeclare(P2P_EXCHANGE_NAME, "direct", false, false, false, false, nil); err != nil {
		return nil, err
	}
	if err := client.Channel.QueueBind(queueName, nodePartyId, BROADCAST_EXCHANGE_NAME, false, nil); err != nil {
		return nil, err
	}
	if err := client.Channel.QueueBind(queueName, nodePartyId, P2P_EXCHANGE_NAME, false, nil); err != nil {
		return nil, err
	}
	return &PartyNode{
		logger:      log.New(os.Stdout, "[mqclient] ", log.Ldate|log.Ltime|log.Lshortfile),
		NodePartyId: nodePartyId,
		QueueName:   queueName,
		mqClient:    client,
	}, nil
}

func (node *PartyNode) BroadcastMessage(data []byte) error {
	return node.mqClient.PushToExchange(data, BROADCAST_EXCHANGE_NAME, "")
}

func (node *PartyNode) SendMessageToNode(data []byte, toNodePartyId string) error {
	return node.mqClient.PushToExchange(data, P2P_EXCHANGE_NAME, toNodePartyId)
}

func (node *PartyNode) RunComsumer(conn net.Conn) {
	<-time.After(time.Second)
	deliveries, err := node.mqClient.Consume(node.QueueName)
	if err != nil {
		node.logger.Printf("Could not start consuming: %s\n", err)
		return
	}

	// This channel will receive a notification when a channel closed event
	// happens. This must be different than Client.notifyChanClose because the
	// library sends only one notification and Client.notifyChanClose already has
	// a receiver in handleReconnect().
	// Recommended to make it buffered to avoid deadlocks
	chClosedCh := make(chan *amqp.Error, 1)
	node.mqClient.Channel.NotifyClose(chClosedCh)

	for {
		select {
		case <-context.Background().Done():
			node.mqClient.Close()
			return

		case amqErr := <-chClosedCh:
			// This case handles the event of closed channel e.g. abnormal shutdown
			node.logger.Printf("AMQP Channel closed due to: %s\n", amqErr)

			deliveries, err = node.mqClient.Consume(node.QueueName)
			if err != nil {
				// If the AMQP channel is not ready, it will continue the loop. Next
				// iteration will enter this case because chClosedCh is closed by the
				// library
				node.logger.Printf("Error trying to consume, will try again")
				continue
			}

			// Re-set channel to receive notifications
			// The library closes this channel after abnormal shutdown
			chClosedCh = make(chan *amqp.Error, 1)
			node.mqClient.Channel.NotifyClose(chClosedCh)

		case delivery := <-deliveries:
			// Ack a message every 2 seconds
			node.logger.Printf("[%s] Received message: %s\n", node.QueueName, delivery.Body)
			if conn == nil {
				node.MessageChan <- string(delivery.Body)
			} else {
				my_vsock.SendMsg(string(delivery.Body), conn)
			}
			if err := delivery.Ack(false); err != nil {
				node.logger.Printf("Error acknowledging message: %s\n", err)
			}
			<-time.After(time.Second * 2)
		}
	}
}
