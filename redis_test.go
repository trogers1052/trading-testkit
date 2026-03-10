package testkit_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trogers1052/trading-testkit"
)

func TestNewRedisContainer(t *testing.T) {
	rc := testkit.NewRedisContainer(t)

	// Verify address is populated
	require.NotEmpty(t, rc.Addr)
	assert.Contains(t, rc.Addr, ":")

	// We don't import go-redis here to keep testkit's test deps minimal.
	// The address being non-empty confirms the container started and
	// the endpoint was resolved.
}
