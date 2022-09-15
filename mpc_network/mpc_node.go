package mpc_network

import (
	"log"
	"os"

	"github.com/holynull/mq-proxy/mqclient"
)

const (
	BROADCAST_EXCHANGE_NAME = "mpc_broadcast_exchange"
	P2P_EXCHANGE_NAME       = "mpc_p2p_exchange"
)

type PartyNode struct {
	logger      *log.Logger
	NodePartyId string
	mqClient    *mqclient.Client
}

func New(nodePartyId string, client *mqclient.Client) (*PartyNode, error) {

	if err := client.Channel.ExchangeDeclare(BROADCAST_EXCHANGE_NAME, "fanout", false, false, false, false, nil); err != nil {
		return nil, err
	}
	if err := client.Channel.ExchangeDeclare(P2P_EXCHANGE_NAME, "direct", false, false, false, false, nil); err != nil {
		return nil, err
	}
	if err := client.Channel.QueueBind(client.QueueName, nodePartyId, BROADCAST_EXCHANGE_NAME, false, nil); err != nil {
		return nil, err
	}
	if err := client.Channel.QueueBind(client.QueueName, nodePartyId, P2P_EXCHANGE_NAME, false, nil); err != nil {
		return nil, err
	}
	return &PartyNode{
		logger:      log.New(os.Stdout, "[mqclient] ", log.Ldate|log.Ltime|log.Lshortfile),
		NodePartyId: nodePartyId,
		mqClient:    client,
	}, nil
}

func (node *PartyNode) BroadCastMessage(data []byte) error {
	return node.mqClient.PushToExchange(data, BROADCAST_EXCHANGE_NAME, "")
}

func (node *PartyNode) SendMessageToNode(data []byte, toNodePartyId string) error {
	return node.mqClient.PushToExchange(data, P2P_EXCHANGE_NAME, toNodePartyId)
}
