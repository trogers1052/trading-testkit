// Package testkit provides shared testcontainers helpers for trading platform
// Go services. It eliminates duplicated test infrastructure across services
// by providing ready-to-use PostgreSQL, TimescaleDB, Redis, and Redpanda
// containers with migration support.
package testkit

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// PostgresContainer wraps a testcontainers PostgreSQL instance with a live
// *sql.DB connection, migration support, and automatic cleanup.
type PostgresContainer struct {
	DB        *sql.DB
	container testcontainers.Container
	ConnStr   string
}

// PostgresOption configures a PostgresContainer.
type PostgresOption func(*pgConfig)

type pgConfig struct {
	image          string
	database       string
	username       string
	password       string
	startupTimeout time.Duration
}

func defaultPGConfig() *pgConfig {
	return &pgConfig{
		image:          "postgres:15-alpine",
		database:       "testdb",
		username:       "testuser",
		password:       "testpass",
		startupTimeout: 60 * time.Second,
	}
}

// WithImage overrides the Docker image (e.g. "timescale/timescaledb:latest-pg15").
func WithImage(image string) PostgresOption {
	return func(c *pgConfig) { c.image = image }
}

// WithDatabase sets the database name.
func WithDatabase(db string) PostgresOption {
	return func(c *pgConfig) { c.database = db }
}

// WithStartupTimeout sets the container startup timeout.
func WithStartupTimeout(d time.Duration) PostgresOption {
	return func(c *pgConfig) { c.startupTimeout = d }
}

// NewPostgresContainer starts a PostgreSQL container and returns a connected
// PostgresContainer. The test is failed immediately if the container cannot
// start. Call Cleanup() (usually via t.Cleanup) when done.
//
// The container is automatically skipped when running with -short.
func NewPostgresContainer(t *testing.T, opts ...PostgresOption) *PostgresContainer {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	cfg := defaultPGConfig()
	for _, o := range opts {
		o(cfg)
	}

	ctx := context.Background()

	pgContainer, err := tcpostgres.Run(ctx,
		cfg.image,
		tcpostgres.WithDatabase(cfg.database),
		tcpostgres.WithUsername(cfg.username),
		tcpostgres.WithPassword(cfg.password),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(cfg.startupTimeout),
		),
	)
	if err != nil {
		t.Fatalf("testkit: failed to start postgres container: %v", err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("testkit: failed to get connection string: %v", err)
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatalf("testkit: failed to open database: %v", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		t.Fatalf("testkit: failed to ping database: %v", err)
	}

	pc := &PostgresContainer{
		DB:        db,
		container: pgContainer,
		ConnStr:   connStr,
	}

	t.Cleanup(func() { pc.Cleanup(t) })

	return pc
}

// NewTimescaleContainer is a convenience wrapper that starts a TimescaleDB
// container instead of plain PostgreSQL.
func NewTimescaleContainer(t *testing.T, opts ...PostgresOption) *PostgresContainer {
	t.Helper()
	return NewPostgresContainer(t, append([]PostgresOption{
		WithImage("timescale/timescaledb:latest-pg15"),
	}, opts...)...)
}

// RunMigrations applies all up-migrations from the given directory path.
// The path should be an absolute filesystem path to a directory containing
// golang-migrate compatible migration files (e.g. 000001_create_table.up.sql).
func (pc *PostgresContainer) RunMigrations(t *testing.T, migrationsPath string) {
	t.Helper()

	driver, err := postgres.WithInstance(pc.DB, &postgres.Config{})
	if err != nil {
		t.Fatalf("testkit: failed to create migration driver: %v", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://"+migrationsPath,
		"postgres",
		driver,
	)
	if err != nil {
		t.Fatalf("testkit: failed to create migrate instance: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("testkit: failed to run migrations: %v", err)
	}
}

// ExecSchema runs raw SQL statements against the database. Useful for services
// that use inline schema creation (e.g. CREATE TABLE IF NOT EXISTS) instead of
// migration files.
func (pc *PostgresContainer) ExecSchema(t *testing.T, sql string) {
	t.Helper()
	if _, err := pc.DB.Exec(sql); err != nil {
		t.Fatalf("testkit: failed to execute schema: %v", err)
	}
}

// TruncateTables truncates the given tables with CASCADE. For TimescaleDB
// hypertables, use DeleteFrom instead.
func (pc *PostgresContainer) TruncateTables(t *testing.T, tables ...string) {
	t.Helper()
	for _, table := range tables {
		if _, err := pc.DB.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)); err != nil {
			t.Fatalf("testkit: failed to truncate %s: %v", table, err)
		}
	}
}

// DeleteFrom runs DELETE FROM on the given tables. Use this for TimescaleDB
// hypertables where TRUNCATE can conflict with continuous aggregates.
func (pc *PostgresContainer) DeleteFrom(t *testing.T, tables ...string) {
	t.Helper()
	for _, table := range tables {
		if _, err := pc.DB.Exec(fmt.Sprintf("DELETE FROM %s", table)); err != nil {
			t.Fatalf("testkit: failed to delete from %s: %v", table, err)
		}
	}
}

// Cleanup closes the database connection and terminates the container.
func (pc *PostgresContainer) Cleanup(t *testing.T) {
	t.Helper()
	if pc.DB != nil {
		pc.DB.Close()
	}
	if pc.container != nil {
		if err := pc.container.Terminate(context.Background()); err != nil {
			t.Errorf("testkit: failed to terminate postgres container: %v", err)
		}
	}
}
