// Package nats provides NATS JetStream client for job queue operations.
package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/example/background-jobs-demo/domain/job"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

const (
	// StreamName is the name of the JetStream stream for jobs.
	StreamName = "JOBS"
	// SubjectJobs is the subject for job messages.
	SubjectJobs = "jobs.>"
	// SubjectJobsNew is the subject for new job messages.
	SubjectJobsNew = "jobs.new"
	// SubjectDeadLetter is the subject for dead-letter messages.
	SubjectDeadLetter = "jobs.dead_letter"
	// ConsumerName is the name of the durable consumer.
	ConsumerName = "job-workers"
)

// Client provides NATS JetStream operations for job queue.
type Client struct {
	nc       *nats.Conn
	js       jetstream.JetStream
	stream   jetstream.Stream
	consumer jetstream.Consumer
	natsURL  string
}

// Config holds NATS client configuration.
type Config struct {
	URL             string
	MaxDeliverCount int
	AckWait         time.Duration
}

// DefaultConfig returns the default NATS configuration.
func DefaultConfig() Config {
	return Config{
		URL:             "nats://localhost:4222",
		MaxDeliverCount: 5,
		AckWait:         30 * time.Second,
	}
}

// NewClient creates a new NATS JetStream client.
func NewClient(cfg Config) *Client {
	return &Client{
		natsURL: cfg.URL,
	}
}

// Connect establishes connection to NATS and sets up JetStream.
func (c *Client) Connect(ctx context.Context) error {
	// Connect to NATS
	nc, err := nats.Connect(c.natsURL,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(10),
		nats.ReconnectWait(time.Second),
	)
	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}
	c.nc = nc

	// Create JetStream context
	js, err := jetstream.New(nc)
	if err != nil {
		return fmt.Errorf("failed to create JetStream context: %w", err)
	}
	c.js = js

	// Create or update stream
	stream, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:        StreamName,
		Description: "Background jobs queue",
		Subjects:    []string{SubjectJobs},
		Retention:   jetstream.WorkQueuePolicy,
		MaxAge:      24 * time.Hour,
		Storage:     jetstream.FileStorage,
		Replicas:    1,
	})
	if err != nil {
		return fmt.Errorf("failed to create stream: %w", err)
	}
	c.stream = stream

	log.Printf("[nats] Connected to NATS at %s, stream %s ready", c.natsURL, StreamName)
	return nil
}

// CreateConsumer creates a durable consumer for job processing.
func (c *Client) CreateConsumer(ctx context.Context, cfg Config) error {
	consumer, err := c.stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Name:          ConsumerName,
		Durable:       ConsumerName,
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       cfg.AckWait,
		MaxDeliver:    cfg.MaxDeliverCount,
		FilterSubject: SubjectJobsNew,
	})
	if err != nil {
		return fmt.Errorf("failed to create consumer: %w", err)
	}
	c.consumer = consumer

	log.Printf("[nats] Consumer %s created", ConsumerName)
	return nil
}

// PublishJob publishes a job to the queue.
func (c *Client) PublishJob(ctx context.Context, j *job.Job) error {
	if c.js == nil {
		return job.ErrQueueUnavailable
	}

	msg := job.JobMessage{
		Job:       j,
		MessageID: j.ID,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal job message: %w", err)
	}

	ack, err := c.js.Publish(ctx, SubjectJobsNew, data)
	if err != nil {
		return fmt.Errorf("failed to publish job: %w", err)
	}

	log.Printf("[nats] Published job %s to stream %s, sequence %d", j.ID, ack.Stream, ack.Sequence)
	return nil
}

// PublishDeadLetter publishes a job to the dead-letter queue.
func (c *Client) PublishDeadLetter(ctx context.Context, j *job.Job, reason string) error {
	if c.js == nil {
		return job.ErrQueueUnavailable
	}

	msg := job.JobMessage{
		Job:       j,
		MessageID: j.ID,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal dead-letter message: %w", err)
	}

	_, err = c.js.Publish(ctx, SubjectDeadLetter, data)
	if err != nil {
		return fmt.Errorf("failed to publish to dead-letter: %w", err)
	}

	log.Printf("[nats] Published job %s to dead-letter queue: %s", j.ID, reason)
	return nil
}

// Subscribe subscribes to job messages for processing.
// Returns a channel of job messages.
func (c *Client) Subscribe(ctx context.Context) (<-chan *ConsumeMessage, error) {
	if c.consumer == nil {
		return nil, fmt.Errorf("consumer not initialized")
	}

	msgChan := make(chan *ConsumeMessage, 100)

	// Start consuming messages
	go func() {
		defer close(msgChan)

		iter, err := c.consumer.Messages()
		if err != nil {
			log.Printf("[nats] Error creating message iterator: %v", err)
			return
		}
		defer iter.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("[nats] Consumer context cancelled, stopping...")
				return
			default:
				msg, err := iter.Next()
				if err != nil {
					if ctx.Err() != nil {
						return
					}
					log.Printf("[nats] Error fetching message: %v", err)
					continue
				}

				var jobMsg job.JobMessage
				if err := json.Unmarshal(msg.Data(), &jobMsg); err != nil {
					log.Printf("[nats] Error unmarshaling message: %v", err)
					if err := msg.Term(); err != nil {
						log.Printf("[nats] Error terminating message: %v", err)
					}
					continue
				}

				metadata, _ := msg.Metadata()
				deliveryCount := 1
				if metadata != nil {
					deliveryCount = int(metadata.NumDelivered)
				}

				msgChan <- &ConsumeMessage{
					Job:           jobMsg.Job,
					DeliveryCount: deliveryCount,
					msg:           msg,
				}
			}
		}
	}()

	return msgChan, nil
}

// ConsumeMessage wraps a job message with acknowledgment methods.
type ConsumeMessage struct {
	Job           *job.Job
	DeliveryCount int
	msg           jetstream.Msg
}

// Ack acknowledges successful processing of the message.
func (m *ConsumeMessage) Ack() error {
	return m.msg.Ack()
}

// Nak negatively acknowledges the message for redelivery.
func (m *ConsumeMessage) Nak() error {
	return m.msg.Nak()
}

// NakWithDelay negatively acknowledges with a delay before redelivery.
func (m *ConsumeMessage) NakWithDelay(delay time.Duration) error {
	return m.msg.NakWithDelay(delay)
}

// Term terminates the message (no more redeliveries).
func (m *ConsumeMessage) Term() error {
	return m.msg.Term()
}

// Close closes the NATS connection.
func (c *Client) Close() error {
	if c.nc != nil {
		c.nc.Close()
		log.Println("[nats] Connection closed")
	}
	return nil
}

// IsConnected returns true if connected to NATS.
func (c *Client) IsConnected() bool {
	return c.nc != nil && c.nc.IsConnected()
}

// GetStreamInfo returns information about the job stream.
func (c *Client) GetStreamInfo(ctx context.Context) (*jetstream.StreamInfo, error) {
	if c.stream == nil {
		return nil, fmt.Errorf("stream not initialized")
	}
	return c.stream.Info(ctx)
}

// GetConsumerInfo returns information about the consumer.
func (c *Client) GetConsumerInfo(ctx context.Context) (*jetstream.ConsumerInfo, error) {
	if c.consumer == nil {
		return nil, fmt.Errorf("consumer not initialized")
	}
	return c.consumer.Info(ctx)
}
