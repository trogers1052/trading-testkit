package testkit_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trogers1052/trading-testkit"
)

func TestNewPostgresContainer(t *testing.T) {
	pg := testkit.NewPostgresContainer(t)

	// Verify connection is live
	err := pg.DB.Ping()
	require.NoError(t, err)

	// Verify we can run queries
	var result int
	err = pg.DB.QueryRow("SELECT 1").Scan(&result)
	require.NoError(t, err)
	assert.Equal(t, 1, result)

	// Verify connection string is populated
	assert.NotEmpty(t, pg.ConnStr)
	assert.Contains(t, pg.ConnStr, "testdb")
}

func TestNewPostgresContainer_ExecSchema(t *testing.T) {
	pg := testkit.NewPostgresContainer(t)

	pg.ExecSchema(t, `
		CREATE TABLE test_items (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100) NOT NULL
		)
	`)

	// Insert and query
	_, err := pg.DB.Exec("INSERT INTO test_items (name) VALUES ('hello')")
	require.NoError(t, err)

	var name string
	err = pg.DB.QueryRow("SELECT name FROM test_items WHERE id = 1").Scan(&name)
	require.NoError(t, err)
	assert.Equal(t, "hello", name)
}

func TestNewPostgresContainer_TruncateTables(t *testing.T) {
	pg := testkit.NewPostgresContainer(t)

	pg.ExecSchema(t, `CREATE TABLE items (id SERIAL PRIMARY KEY, val TEXT)`)

	_, err := pg.DB.Exec("INSERT INTO items (val) VALUES ('a'), ('b'), ('c')")
	require.NoError(t, err)

	pg.TruncateTables(t, "items")

	var count int
	err = pg.DB.QueryRow("SELECT COUNT(*) FROM items").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestNewTimescaleContainer(t *testing.T) {
	pg := testkit.NewTimescaleContainer(t)

	// Verify TimescaleDB extension is available
	var extName string
	err := pg.DB.QueryRow("SELECT extname FROM pg_extension WHERE extname = 'timescaledb'").Scan(&extName)
	// Extension may not be auto-created, but the image should support it
	if err != nil {
		// Try creating it
		_, err = pg.DB.Exec("CREATE EXTENSION IF NOT EXISTS timescaledb")
		require.NoError(t, err, "TimescaleDB extension should be available in the image")
	}

	// Verify we can create a hypertable
	pg.ExecSchema(t, `
		CREATE EXTENSION IF NOT EXISTS timescaledb;
		CREATE TABLE ts_test (
			time TIMESTAMPTZ NOT NULL,
			value DOUBLE PRECISION
		);
		SELECT create_hypertable('ts_test', 'time');
	`)

	// Insert data
	_, err = pg.DB.Exec("INSERT INTO ts_test (time, value) VALUES (NOW(), 42.0)")
	require.NoError(t, err)

	var val float64
	err = pg.DB.QueryRow("SELECT value FROM ts_test LIMIT 1").Scan(&val)
	require.NoError(t, err)
	assert.Equal(t, 42.0, val)
}

func TestNewPostgresContainer_DeleteFrom(t *testing.T) {
	pg := testkit.NewPostgresContainer(t)

	pg.ExecSchema(t, `CREATE TABLE deletable (id SERIAL PRIMARY KEY, val TEXT)`)

	_, err := pg.DB.Exec("INSERT INTO deletable (val) VALUES ('x'), ('y')")
	require.NoError(t, err)

	pg.DeleteFrom(t, "deletable")

	var count int
	err = pg.DB.QueryRow("SELECT COUNT(*) FROM deletable").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}
