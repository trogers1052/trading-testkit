package testkit

import (
	"context"
	"testing"

	"github.com/IBM/sarama"
	"github.com/testcontainers/testcontainers-go"
	tcredpanda "github.com/testcontainers/testcontainers-go/modules/redpanda"
)

// RedpandaContainer wraps a testcontainers Redpanda instance (Kafka-compatible).
type RedpandaContainer struct {
	// Brokers is the Kafka-compatible broker address (e.g. "localhost:55002").
	Brokers   string
	container testcontainers.Container
}

// RedpandaOption configures a RedpandaContainer.
type RedpandaOption func(*rpConfig)

type rpConfig struct {
	image string
}

func defaultRPConfig() *rpConfig {
	return &rpConfig{
		image: "redpandadata/redpanda:v24.1.1",
	}
}

// WithRedpandaImage overrides the Redpanda Docker image.
func WithRedpandaImage(image string) RedpandaOption {
	return func(c *rpConfig) { c.image = image }
}

// NewRedpandaContainer starts a Redpanda container and returns a
// RedpandaContainer with the Kafka broker address. Skipped in -short mode.
// Uses the Redpanda module's built-in wait strategy.
func NewRedpandaContainer(t *testing.T, opts ...RedpandaOption) *RedpandaContainer {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	cfg := defaultRPConfig()
	for _, o := range opts {
		o(cfg)
	}

	ctx := context.Background()

	rpC, err := tcredpanda.Run(ctx, cfg.image)
	if err != nil {
		t.Fatalf("testkit: failed to start redpanda container: %v", err)
	}

	brokers, err := rpC.KafkaSeedBroker(ctx)
	if err != nil {
		t.Fatalf("testkit: failed to get redpanda broker address: %v", err)
	}

	rc := &RedpandaContainer{
		Brokers:   brokers,
		container: rpC,
	}

	t.Cleanup(func() { rc.Cleanup(t) })

	return rc
}

// CreateTopic creates a Kafka topic via the Sarama admin client.
func (rc *RedpandaContainer) CreateTopic(t *testing.T, topic string, partitions int) {
	t.Helper()

	cfg := sarama.NewConfig()
	admin, err := sarama.NewClusterAdmin([]string{rc.Brokers}, cfg)
	if err != nil {
		t.Fatalf("testkit: failed to create admin client: %v", err)
	}
	defer admin.Close()

	err = admin.CreateTopic(topic, &sarama.TopicDetail{
		NumPartitions:     int32(partitions),
		ReplicationFactor: 1,
	}, false)
	if err != nil {
		t.Fatalf("testkit: failed to create topic %s: %v", topic, err)
	}
}

// Cleanup terminates the Redpanda container.
func (rc *RedpandaContainer) Cleanup(t *testing.T) {
	t.Helper()
	if rc.container != nil {
		if err := rc.container.Terminate(context.Background()); err != nil {
			t.Errorf("testkit: failed to terminate redpanda container: %v", err)
		}
	}
}
