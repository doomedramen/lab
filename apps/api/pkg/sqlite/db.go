package sqlite

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// DB wraps the sql.DB connection and provides configuration
type DB struct {
	*sql.DB
	path string
}

// Config holds database configuration
type Config struct {
	Path           string // Path to SQLite file (default: ./data/metrics.db)
	RetentionDays  int    // Days to retain metrics (default: 30)
	EventRetention int    // Days to retain events (default: 90)
	LogRetention   int    // Days to retain VM logs (default: 7)
}

// New creates a new SQLite database connection
func New(cfg Config) (*DB, error) {
	// Create data directory if it doesn't exist
	dir := filepath.Dir(cfg.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Open database connection
	db, err := sql.Open("sqlite", cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode and other optimizations
	_, err = db.Exec(`
		PRAGMA journal_mode = WAL;
		PRAGMA busy_timeout = 5000;
		PRAGMA synchronous = NORMAL;
		PRAGMA cache_size = 10000;
		PRAGMA temp_store = memory;
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to configure database: %w", err)
	}

	database := &DB{
		DB:   db,
		path: cfg.Path,
	}

	// Run migrations
	if err := database.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return database, nil
}

// migrate runs database migrations from the embedded filesystem
func (d *DB) migrate() error {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}

		content, err := migrationsFS.ReadFile(filepath.Join("migrations", entry.Name()))
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", entry.Name(), err)
		}

		// Execute migration
		_, err = d.Exec(string(content))
		if err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", entry.Name(), err)
		}
	}

	return nil
}

// Path returns the database file path
func (d *DB) Path() string {
	return d.path
}

// Close closes the database connection
func (d *DB) Close() error {
	return d.DB.Close()
}

// Stats returns database statistics
func (d *DB) Stats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get database size
	info, err := os.Stat(d.path)
	if err == nil {
		stats["size_bytes"] = info.Size()
	}

	// Get row counts
	var metricCount, eventCount int
	if err := d.QueryRow("SELECT COUNT(*) FROM metrics").Scan(&metricCount); err == nil {
		stats["metric_count"] = metricCount
	}
	if err := d.QueryRow("SELECT COUNT(*) FROM events").Scan(&eventCount); err == nil {
		stats["event_count"] = eventCount
	}

	return stats, nil
}
