package testkit_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trogers1052/trading-testkit"
)

func TestNewRedpandaContainer(t *testing.T) {
	rp := testkit.NewRedpandaContainer(t)

	// Verify broker address is populated
	require.NotEmpty(t, rp.Brokers)
	assert.Contains(t, rp.Brokers, ":")
}
