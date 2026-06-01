package testkit_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	testkit "github.com/trogers1052/trading-testkit"
)

// writeMigrations lays down a minimal golang-migrate compatible migration set
// in a temp dir and returns the path. Each migration needs an .up.sql (and a
// matching .down.sql) named with a zero-padded version prefix.
func writeMigrations(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	files := map[string]string{
		"000001_create_widgets.up.sql": `CREATE TABLE widgets (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL
		);`,
		"000001_create_widgets.down.sql": `DROP TABLE widgets;`,
		"000002_add_color.up.sql":        `ALTER TABLE widgets ADD COLUMN color TEXT;`,
		"000002_add_color.down.sql":      `ALTER TABLE widgets DROP COLUMN color;`,
	}
	for name, body := range files {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644))
	}
	return dir
}

func TestRunMigrations_AppliesUpMigrations(t *testing.T) {
	// Uses WithDatabase + WithStartupTimeout so those option closures are
	// exercised alongside the migration runner.
	pg := testkit.NewPostgresContainer(t,
		testkit.WithDatabase("migration_db"),
		testkit.WithStartupTimeout(90*time.Second),
	)

	migrationsPath := writeMigrations(t)
	pg.RunMigrations(t, migrationsPath)

	// Both migrations should have applied: widgets table with a color column.
	var colCount int
	err := pg.DB.QueryRow(`
		SELECT COUNT(*) FROM information_schema.columns
		WHERE table_name = 'widgets' AND column_name = 'color'
	`).Scan(&colCount)
	require.NoError(t, err)
	assert.Equal(t, 1, colCount, "expected 'color' column added by migration 000002")

	// Re-running should be a no-op (ErrNoChange branch), not an error.
	pg.RunMigrations(t, migrationsPath)
}

func TestPostgresContainer_DefaultConnStrUsesConfiguredDatabase(t *testing.T) {
	pg := testkit.NewPostgresContainer(t, testkit.WithDatabase("custom_named_db"))
	assert.Contains(t, pg.ConnStr, "custom_named_db")
}
