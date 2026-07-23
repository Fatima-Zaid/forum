package database

import (
	"database/sql"
	_ "embed"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

// schemaSQL is embedded at compile time so InitDB doesn't depend on the
// working directory the binary happens to be run from (important inside Docker).
//
//go:embed schema.sql
var schemaSQL string

// DB is the shared connection pool used by the whole app.
var DB *sql.DB

// InitDB opens (creating if needed) the SQLite file at dbPath, applies the
// schema, and stores the connection in the package-level DB variable.
func InitDB(dbPath string) (*sql.DB, error) {
	// _foreign_keys=on is required because SQLite disables FK enforcement
	// by default on every new connection.
	dsn := fmt.Sprintf("file:%s?_foreign_keys=on", dbPath)

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// go-sqlite3 doesn't handle concurrent writers well; a single connection
	// avoids "database is locked" errors under concurrent requests.
	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	if _, err := db.Exec(schemaSQL); err != nil {
		return nil, fmt.Errorf("apply schema: %w", err)
	}

	DB = db
	log.Println("database initialized at", dbPath)
	return db, nil
}

// Close closes the shared connection pool. Call this with defer in main.go.
func Close() {
	if DB != nil {
		DB.Close()
	}
}