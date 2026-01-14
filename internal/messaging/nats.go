package messaging

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
)

// NATSClient handles NATS JetStream messaging
type NATSClient struct {
	conn      *nats.Conn
	js        nats.JetStreamContext
	streamName string
}

// NewNATSClient creates a new NATS client with JetStream
func NewNATSClient(url string) (*NATSClient, error) {
	// Connect to NATS
	nc, err := nats.Connect(url,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(10),
		nats.ReconnectWait(time.Second),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			slog.Warn("nats disconnected", "error", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			slog.Info("nats reconnected", "url", nc.ConnectedUrl())
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Create JetStream context
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	client := &NATSClient{
		conn:       nc,
		js:         js,
		streamName: "RTB_LOGS",
	}

	// Ensure stream exists
	if err := client.ensureStream(); err != nil {
		slog.Warn("failed to create stream", "error", err)
	}

	return client, nil
}

// ensureStream creates the stream if it doesn't exist
func (c *NATSClient) ensureStream() error {
	_, err := c.js.StreamInfo(c.streamName)
	if err == nil {
		return nil // Stream exists
	}

	// Create stream
	_, err = c.js.AddStream(&nats.StreamConfig{
		Name:       c.streamName,
		Subjects:   []string{"bid.>", "rtb.>"},
		Storage:    nats.FileStorage,
		Retention:  nats.LimitsPolicy,
		MaxAge:     24 * time.Hour,
		MaxBytes:   1024 * 1024 * 1024, // 1GB
		Discard:    nats.DiscardOld,
		MaxMsgs:    -1,
		Duplicates: time.Minute,
	})
	if err != nil {
		return fmt.Errorf("failed to create stream: %w", err)
	}

	slog.Info("created JetStream stream", "name", c.streamName)
	return nil
}

// PublishAsync publishes a message asynchronously
func (c *NATSClient) PublishAsync(subject string, data []byte) error {
	if c.js == nil {
		return fmt.Errorf("jetstream not initialized")
	}

	_, err := c.js.PublishAsync(subject, data)
	return err
}

// Publish publishes a message synchronously
func (c *NATSClient) Publish(subject string, data []byte) error {
	if c.js == nil {
		return fmt.Errorf("jetstream not initialized")
	}

	_, err := c.js.Publish(subject, data)
	return err
}

// Subscribe creates a push subscription
func (c *NATSClient) Subscribe(subject string, handler func(msg *nats.Msg)) (*nats.Subscription, error) {
	return c.js.Subscribe(subject, handler, nats.DeliverNew())
}

// QueueSubscribe creates a queue subscription for load balancing
func (c *NATSClient) QueueSubscribe(subject, queue string, handler func(msg *nats.Msg)) (*nats.Subscription, error) {
	return c.js.QueueSubscribe(subject, queue, handler, nats.DeliverNew())
}

// Close closes the NATS connection
func (c *NATSClient) Close() error {
	if c.conn != nil {
		c.conn.Close()
	}
	return nil
}

// IsConnected returns true if connected to NATS
func (c *NATSClient) IsConnected() bool {
	return c.conn != nil && c.conn.IsConnected()
}
