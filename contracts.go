package testkit

import (
	"embed"
)

//go:embed contracts/*.json
var contractFS embed.FS

// LoadContract returns the raw JSON bytes for a named contract fixture.
// Name should be the filename without the directory prefix, e.g. "trade_event.json".
func LoadContract(name string) []byte {
	data, err := contractFS.ReadFile("contracts/" + name)
	if err != nil {
		panic("testkit: contract fixture not found: " + name + ": " + err.Error())
	}
	return data
}
