package metadatastore

import (
	"context"
	"database/sql"
	"fmt"
	_ "modernc.org/sqlite"
	"time"
)

// SQLiteStore provides SQLite-based persistence for projects, pipelines, and schedules
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore creates a new SQLite-based storage instance
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	// Open database with connection pooling parameters.
	// Format: file:path?param=value
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(ON)", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	// SetMaxOpenConns: Maximum number of open connections to the database
	// For SQLite, we want this relatively low since writes are serialized anyway
	db.SetMaxOpenConns(10)

	// SetMaxIdleConns: Maximum number of connections in the idle connection pool
	db.SetMaxIdleConns(5)

	// SetConnMaxLifetime: Maximum amount of time a connection may be reused
	db.SetConnMaxLifetime(time.Hour)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	store := &SQLiteStore{db: db}

	// Verify WAL mode is enabled (or delete mode for in-memory databases in tests).
	var journalMode string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		return nil, fmt.Errorf("failed to check journal mode: %w", err)
	}
	// WAL mode should be enabled for file-based databases.
	// In-memory databases will use "delete" or "memory" mode, which is acceptable for testing.
	if journalMode != "wal" && journalMode != "delete" && journalMode != "memory" {
		return nil, fmt.Errorf("unexpected journal mode: got %s", journalMode)
	}

	var foreignKeysEnabled int
	if err := db.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeysEnabled); err != nil {
		return nil, fmt.Errorf("failed to check foreign key enforcement: %w", err)
	}
	if foreignKeysEnabled != 1 {
		return nil, fmt.Errorf("sqlite foreign key enforcement is disabled")
	}

	// Initialize schema
	if err := store.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return store, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// ResetAll deletes all persisted metadata rows while preserving the schema.
func (s *SQLiteStore) ResetAll() error {
	return s.retryOnBusy(func() error {
		ctx := context.Background()
		conn, err := s.db.Conn(ctx)
		if err != nil {
			return fmt.Errorf("acquire reset connection: %w", err)
		}
		defer conn.Close()

		if _, err := conn.ExecContext(ctx, "PRAGMA foreign_keys = OFF"); err != nil {
			return fmt.Errorf("disable foreign keys for reset: %w", err)
		}
		defer conn.ExecContext(ctx, "PRAGMA foreign_keys = ON")

		rows, err := conn.QueryContext(ctx, `SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
		if err != nil {
			return fmt.Errorf("list metadata tables: %w", err)
		}
		var tableNames []string
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				rows.Close()
				return fmt.Errorf("scan metadata table name: %w", err)
			}
			tableNames = append(tableNames, name)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return fmt.Errorf("iterate metadata tables: %w", err)
		}
		rows.Close()

		tx, err := conn.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin reset transaction: %w", err)
		}
		defer tx.Rollback()

		for _, tableName := range tableNames {
			if _, err := tx.ExecContext(ctx, fmt.Sprintf("DELETE FROM %q", tableName)); err != nil {
				return fmt.Errorf("clear table %s: %w", tableName, err)
			}
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit metadata reset: %w", err)
		}
		return nil
	}, 5)
}

// retryOnBusy retries a database operation if it fails due to SQLITE_BUSY
// This provides an additional safety net on top of the busy_timeout pragma
func (s *SQLiteStore) retryOnBusy(operation func() error, maxRetries int) error {
	var err error
	for i := 0; i < maxRetries; i++ {
		err = operation()
		if err == nil {
			return nil
		}

		// Check if error is SQLITE_BUSY (database is locked)
		if err.Error() == "database is locked (5) (SQLITE_BUSY)" {
			// Exponential backoff: 10ms, 20ms, 40ms, 80ms, 160ms
			backoff := time.Duration(10*(1<<uint(i))) * time.Millisecond
			time.Sleep(backoff)
			continue
		}

		// If it's not a busy error, return immediately
		return err
	}
	return fmt.Errorf("operation failed after %d retries: %w", maxRetries, err)
}

type sqlExecer interface {
	Exec(query string, args ...any) (sql.Result, error)
}
