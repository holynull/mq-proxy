package mqclient

import (
	"context"
	"errors"
	"log"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Client struct {
	logger          *log.Logger
	QueueName       string
	connection      *amqp.Connection
	Channel         *amqp.Channel
	Done            chan bool
	NotifyConnClose chan *amqp.Error
	NotifyChanClose chan *amqp.Error
	NotifyConfirm   chan amqp.Confirmation
	IsReady         bool
}

const (
	reconnectDelay = 5 * time.Second

	reInitDelay = 2 * time.Second

	resendDelay = 5 * time.Second
)

var (
	errNotConnected  = errors.New("not connected to a server")
	errAlreadyClosed = errors.New("already closed: not connected to the server")
	errShutdown      = errors.New("client is shutting down")
)

// New creates a new consumer state instance, and automatically
// attempts to connect to the server.
func New(queueName, addr string) *Client {
	client := Client{
		QueueName: queueName,
		logger:    log.New(os.Stdout, "[mqclient] ", log.Ldate|log.Ltime|log.Lshortfile),
		Done:      make(chan bool),
	}
	go client.handleReconnect(addr)
	return &client
}

// handleReconnect will wait for a connection error on
// notifyConnClose, and then continuously attempt to reconnect.
func (client *Client) handleReconnect(addr string) {
	for {
		client.IsReady = false
		client.logger.Println("Attempting to connect")

		conn, err := client.connect(addr)

		if err != nil {
			client.logger.Println("Failed to connect. Retrying...")

			select {
			case <-client.Done:
				return
			case <-time.After(reconnectDelay):
			}
			continue
		}

		if done := client.handleReInit(conn); done {
			break
		}
	}
}

// connect will create a new AMQP connection
func (client *Client) connect(addr string) (*amqp.Connection, error) {
	conn, err := amqp.Dial(addr)

	if err != nil {
		return nil, err
	}

	client.changeConnection(conn)
	client.logger.Println("Connected!")
	return conn, nil
}

// handleReconnect will wait for a channel error
// and then continuously attempt to re-initialize both channels
func (client *Client) handleReInit(conn *amqp.Connection) bool {
	for {
		client.IsReady = false

		err := client.init(conn)

		if err != nil {
			client.logger.Println("Failed to initialize channel. Retrying...")

			select {
			case <-client.Done:
				return true
			case <-time.After(reInitDelay):
			}
			continue
		}

		select {
		case <-client.Done:
			return true
		case <-client.NotifyConnClose:
			client.logger.Println("Connection closed. Reconnecting...")
			return false
		case <-client.NotifyChanClose:
			client.logger.Println("Channel closed. Re-running init...")
		}
	}
}

// init will initialize channel & declare queue
func (client *Client) init(conn *amqp.Connection) error {
	ch, err := conn.Channel()

	if err != nil {
		return err
	}
	if _, err := ch.QueueDeclare(
		client.QueueName,
		false,
		false,
		false,
		false,
		nil,
	); err != nil {
		return err
	}
	err = ch.Confirm(false)

	if err != nil {
		return err
	}

	client.changeChannel(ch)
	client.IsReady = true
	client.logger.Println("Setup!")

	return nil
}

// changeConnection takes a new connection to the queue,
// and updates the close listener to reflect this.
func (client *Client) changeConnection(connection *amqp.Connection) {
	client.connection = connection
	client.NotifyConnClose = make(chan *amqp.Error, 1)
	client.connection.NotifyClose(client.NotifyConnClose)
}

// changeChannel takes a new channel to the queue,
// and updates the channel listeners to reflect this.
func (client *Client) changeChannel(channel *amqp.Channel) {
	client.Channel = channel
	client.NotifyChanClose = make(chan *amqp.Error, 1)
	client.NotifyConfirm = make(chan amqp.Confirmation, 1)
	client.Channel.NotifyClose(client.NotifyChanClose)
	client.Channel.NotifyPublish(client.NotifyConfirm)
}

// Push will push data onto the queue, and wait for a confirm.
// If no confirms are received until within the resendTimeout,
// it continuously re-sends messages until a confirm is received.
// This will block until the server sends a confirm. Errors are
// only returned if the push action itself fails, see UnsafePush.
func (client *Client) Push(data []byte) error {
	if !client.IsReady {
		return errors.New("failed to push: not connected")
	}
	for {
		err := client.UnsafePush(data, "", client.QueueName)

		if err != nil {
			client.logger.Println("Push failed. Retrying...")
			select {
			case <-client.Done:
				return errShutdown
			case <-time.After(resendDelay):
			}
			continue
		}
		select {
		case confirm := <-client.NotifyConfirm:
			if confirm.Ack {
				client.logger.Println("Push confirmed!")
				return nil
			}
		case <-time.After(resendDelay):
		}
		client.logger.Println("Push didn't confirm. Retrying...")
	}
}

func (client *Client) PushToExchange(data []byte, exchange string, key string) error {
	if !client.IsReady {
		return errors.New("failed to push: not connected")
	}
	for {
		err := client.UnsafePush(data, exchange, key)
		if err != nil {
			client.logger.Println("Push failed. Retrying...")
			select {
			case <-client.Done:
				return errShutdown
			case <-time.After(resendDelay):
			}
			continue
		}
		select {
		case confirm := <-client.NotifyConfirm:
			if confirm.Ack {
				client.logger.Println("Push confirmed!")
				return nil
			}
		case <-time.After(resendDelay):
		}
		client.logger.Println("Push didn't confirm. Retrying...")
	}
}

// UnsafePush will push to the queue without checking for
// confirmation. It returns an error if it fails to connect.
// No guarantees are provided for whether the server will
// receive the message.
func (client *Client) UnsafePush(data []byte, exchange string, key string) error {
	if !client.IsReady {
		return errNotConnected
	}
	_, err := client.Channel.QueueDeclare(
		client.QueueName,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return client.Channel.PublishWithContext(
		ctx,
		exchange,
		key,
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        data,
		},
	)
}

// Consume will continuously put queue items on the channel.
// It is required to call delivery.Ack when it has been
// successfully processed, or delivery.Nack when it fails.
// Ignoring this will cause data to build up on the server.
func (client *Client) Consume() (<-chan amqp.Delivery, error) {
	if !client.IsReady {
		return nil, errNotConnected
	}

	if err := client.Channel.Qos(
		1,
		0,
		false,
	); err != nil {
		return nil, err
	}

	return client.Channel.Consume(
		client.QueueName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
}

// Close will cleanly shutdown the channel and connection.
func (client *Client) Close() error {
	if !client.IsReady {
		return errAlreadyClosed
	}
	close(client.Done)
	err := client.Channel.Close()
	if err != nil {
		return err
	}
	err = client.connection.Close()
	if err != nil {
		return err
	}

	client.IsReady = false
	return nil
}
