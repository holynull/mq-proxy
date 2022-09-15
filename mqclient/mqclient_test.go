package mqclient

import (
	"context"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func TestMqClient(t *testing.T) {
	addr := "amqp://guest:guest@localhost:5672/"
	sender := New("mq_client_test", addr)
	consumer := New("mq_client_test", addr)
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second*20))
	defer cancel()
	go runComsumer(consumer, ctx, t)
	runSender(sender, ctx, t)
}

func runSender(sender *Client, ctx context.Context, t *testing.T) {
	message := []byte("I'm a test message.")
loop:
	for {
		select {
		// Attempt to push a message every 2 seconds
		case <-time.After(time.Second * 2):
			if err := sender.Push(message); err != nil {
				t.Errorf("Push failed: %s\n", err)
			} else {
				t.Logf("Push succeeded!")
			}
		case <-ctx.Done():
			sender.Close()
			break loop
		}
	}
}

func runComsumer(consumer *Client, ctx context.Context, t *testing.T) {
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
		case <-ctx.Done():
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
			t.Logf("Received message: %s\n", delivery.Body)
			if err := delivery.Ack(false); err != nil {
				t.Errorf("Error acknowledging message: %s\n", err)
			}
			<-time.After(time.Second * 2)
		}
	}
}
