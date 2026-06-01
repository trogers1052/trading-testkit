# trading-testkit

Shared [testcontainers](https://golang.testcontainers.org/) helpers for the trading
platform's Go services. It removes duplicated test-infrastructure boilerplate by
providing ready-to-use **PostgreSQL**, **TimescaleDB**, **Redis**, and **Redpanda**
(Kafka-compatible) containers with migration support, schema helpers, and automatic
cleanup — plus a small loader for shared JSON event-contract fixtures.

Package name: `testkit`
Module path: `github.com/trogers1052/trading-testkit`

## Install

```bash
go get github.com/trogers1052/trading-testkit
```

```go
import testkit "github.com/trogers1052/trading-testkit"
```

> **Requires Docker.** Every container helper starts a real Docker container.
> All helpers automatically `t.Skip` when the test binary is run with `-short`,
> so unit-only runs (`go test -short ./...`) never need Docker.

## Why

Each Go service needs the same integration-test scaffolding: spin up Postgres/Timescale,
apply migrations, get a Redis address, create a Kafka topic, then tear it all down.
This library centralizes that so services share one battle-tested implementation.

## Container helpers

Every `New*Container` helper takes `*testing.T`, fails the test immediately on any
startup error, registers its own `t.Cleanup`, and skips under `-short`.

### PostgreSQL / TimescaleDB

```go
func TestRepository(t *testing.T) {
    pg := testkit.NewPostgresContainer(t)        // postgres:15-alpine by default
    // pg.DB is a *sql.DB; pg.ConnStr is the connection string

    // Apply golang-migrate files (absolute path to a dir of *.up.sql / *.down.sql)
    pg.RunMigrations(t, "/abs/path/to/migrations")

    // ...exercise your repository against pg.DB...
}
```

Configure via functional options:

```go
pg := testkit.NewPostgresContainer(t,
    testkit.WithImage("postgres:16-alpine"),
    testkit.WithDatabase("trading_platform"),
    testkit.WithStartupTimeout(90*time.Second),
)
```

For TimescaleDB, use the convenience wrapper (defaults the image to
`timescale/timescaledb:latest-pg15`; all `PostgresOption`s still apply):

```go
ts := testkit.NewTimescaleContainer(t, testkit.WithDatabase("market_data"))
```

Schema and cleanup helpers on `*PostgresContainer`:

```go
// Inline schema instead of migration files:
pg.ExecSchema(t, `CREATE TABLE IF NOT EXISTS quotes (sym TEXT, price NUMERIC)`)

// Reset state between sub-tests:
pg.TruncateTables(t, "quotes", "orders")  // TRUNCATE ... CASCADE
pg.DeleteFrom(t, "ts_hypertable")         // DELETE FROM — use for Timescale hypertables

// Cleanup() is registered automatically via t.Cleanup; call manually only if needed.
```

Defaults: image `postgres:15-alpine`, database `testdb`, user `testuser`,
password `testpass`, startup timeout 60s.

### Redis

```go
func TestCache(t *testing.T) {
    r := testkit.NewRedisContainer(t)   // redis:7-alpine by default
    // r.Addr is the host:port to dial

    client := redis.NewClient(&redis.Options{Addr: r.Addr})
    // ...
}

// Override the image:
r := testkit.NewRedisContainer(t, testkit.WithRedisImage("redis:8-alpine"))
```

### Redpanda (Kafka-compatible)

```go
func TestConsumer(t *testing.T) {
    rp := testkit.NewRedpandaContainer(t)   // redpandadata/redpanda:v24.1.1 by default
    // rp.Brokers is the Kafka seed broker address

    rp.CreateTopic(t, "stock.quotes.realtime", 1 /* partitions */)

    // ...produce/consume against rp.Brokers with your Kafka client...
}

// Override the image:
rp := testkit.NewRedpandaContainer(t, testkit.WithRedpandaImage("redpandadata/redpanda:v24.2.1"))
```

## Contract fixtures

Shared JSON event contracts are embedded in the library and loaded by filename.
`LoadContract` returns the raw bytes and **panics** if the fixture is missing
(it is intended for use in tests where a missing fixture is a programming error).

```go
raw := testkit.LoadContract("trade_event.json")

var evt TradeEvent
if err := json.Unmarshal(raw, &evt); err != nil {
    t.Fatalf("contract did not match struct: %v", err)
}
```

Available fixtures (under `contracts/`):

- `decision_event.json`
- `positions_event.json`
- `quote_event.json`
- `ranking_event.json`
- `stock_event.json`
- `trade_event.json`
- `watchlist_event_added.json`
- `watchlist_event_updated.json`

## Running tests with and without Docker

```bash
go test -short ./...   # unit tests only; all container helpers skip
go test ./...          # full integration tests; requires a running Docker daemon
```

## Exported API summary

| Symbol | Kind | Purpose |
|--------|------|---------|
| `NewPostgresContainer` | func | Start a PostgreSQL container |
| `NewTimescaleContainer` | func | Start a TimescaleDB container (Postgres wrapper) |
| `WithImage` / `WithDatabase` / `WithStartupTimeout` | option | Configure a Postgres/Timescale container |
| `(*PostgresContainer).RunMigrations` | method | Apply golang-migrate up-migrations from a directory |
| `(*PostgresContainer).ExecSchema` | method | Run raw inline SQL schema |
| `(*PostgresContainer).TruncateTables` | method | `TRUNCATE ... CASCADE` named tables |
| `(*PostgresContainer).DeleteFrom` | method | `DELETE FROM` named tables (Timescale hypertables) |
| `(*PostgresContainer).Cleanup` | method | Close DB + terminate container |
| `PostgresContainer.DB` / `.ConnStr` | field | `*sql.DB` and connection string |
| `NewRedisContainer` | func | Start a Redis container |
| `WithRedisImage` | option | Configure the Redis image |
| `(*RedisContainer).Cleanup` | method | Terminate container |
| `RedisContainer.Addr` | field | host:port to dial |
| `NewRedpandaContainer` | func | Start a Redpanda (Kafka) container |
| `WithRedpandaImage` | option | Configure the Redpanda image |
| `(*RedpandaContainer).CreateTopic` | method | Create a Kafka topic |
| `(*RedpandaContainer).Cleanup` | method | Terminate container |
| `RedpandaContainer.Brokers` | field | Kafka seed broker address |
| `LoadContract` | func | Load an embedded JSON contract fixture by filename |

## License

[MIT](LICENSE)

---

## Built with Claude Code

A large portion of this project — implementation, tests, and documentation — was written in pair-programming sessions with [Claude Code](https://claude.com/claude-code), Anthropic's agentic command-line tool.
