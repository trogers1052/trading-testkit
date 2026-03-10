package testkit

import (
	"context"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
)

// RedisContainer wraps a testcontainers Redis instance.
type RedisContainer struct {
	// Addr is the host:port address for connecting (e.g. "localhost:55001").
	Addr      string
	container testcontainers.Container
}

// RedisOption configures a RedisContainer.
type RedisOption func(*redisConfig)

type redisConfig struct {
	image          string
	startupTimeout time.Duration
}

func defaultRedisConfig() *redisConfig {
	return &redisConfig{
		image:          "redis:7-alpine",
		startupTimeout: 30 * time.Second,
	}
}

// WithRedisImage overrides the Redis Docker image.
func WithRedisImage(image string) RedisOption {
	return func(c *redisConfig) { c.image = image }
}

// NewRedisContainer starts a Redis container and returns a RedisContainer with
// the connection address. Skipped in -short mode.
func NewRedisContainer(t *testing.T, opts ...RedisOption) *RedisContainer {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	cfg := defaultRedisConfig()
	for _, o := range opts {
		o(cfg)
	}

	ctx := context.Background()

	redisC, err := tcredis.Run(ctx,
		cfg.image,
		testcontainers.WithWaitStrategy(
			wait.ForLog("Ready to accept connections").
				WithStartupTimeout(cfg.startupTimeout),
		),
	)
	if err != nil {
		t.Fatalf("testkit: failed to start redis container: %v", err)
	}

	endpoint, err := redisC.Endpoint(ctx, "")
	if err != nil {
		t.Fatalf("testkit: failed to get redis endpoint: %v", err)
	}

	rc := &RedisContainer{
		Addr:      endpoint,
		container: redisC,
	}

	t.Cleanup(func() { rc.Cleanup(t) })

	return rc
}

// Cleanup terminates the Redis container.
func (rc *RedisContainer) Cleanup(t *testing.T) {
	t.Helper()
	if rc.container != nil {
		if err := rc.container.Terminate(context.Background()); err != nil {
			t.Errorf("testkit: failed to terminate redis container: %v", err)
		}
	}
}
