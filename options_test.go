package testkit_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	testkit "github.com/trogers1052/trading-testkit"
)

// These exercise the option closures for the Redis and Redpanda containers by
// passing them through to the real container constructors (Docker-backed,
// skipped in -short mode).

func TestNewRedisContainer_WithRedisImage(t *testing.T) {
	rc := testkit.NewRedisContainer(t, testkit.WithRedisImage("redis:7-alpine"))
	require.NotEmpty(t, rc.Addr)
	assert.Contains(t, rc.Addr, ":")
}

func TestNewRedpandaContainer_WithRedpandaImage_AndCreateTopic(t *testing.T) {
	rp := testkit.NewRedpandaContainer(t, testkit.WithRedpandaImage("redpandadata/redpanda:v24.1.1"))
	require.NotEmpty(t, rp.Brokers)

	// CreateTopic drives the Sarama admin client against the live broker.
	rp.CreateTopic(t, "testkit.topic", 1)
}
