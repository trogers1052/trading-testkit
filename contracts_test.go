package testkit_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	testkit "github.com/trogers1052/trading-testkit"
)

// LoadContract is pure (embedded FS) and needs no Docker, so these run in
// every mode including -short.

func TestLoadContract_ReturnsValidJSON(t *testing.T) {
	fixtures := []string{
		"decision_event.json",
		"positions_event.json",
		"quote_event.json",
		"ranking_event.json",
		"stock_event.json",
		"trade_event.json",
		"watchlist_event_added.json",
		"watchlist_event_updated.json",
	}
	for _, name := range fixtures {
		name := name
		t.Run(name, func(t *testing.T) {
			data := testkit.LoadContract(name)
			require.NotEmpty(t, data, "expected non-empty contract bytes")

			var v any
			require.NoError(t, json.Unmarshal(data, &v), "contract %s should be valid JSON", name)
		})
	}
}

func TestLoadContract_MissingFixture_Panics(t *testing.T) {
	assert.PanicsWithValue(t,
		"testkit: contract fixture not found: does_not_exist.json: open contracts/does_not_exist.json: file does not exist",
		func() { testkit.LoadContract("does_not_exist.json") },
	)
}
