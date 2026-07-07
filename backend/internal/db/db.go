package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStore wraps the standard sql.DB to provide specific repository methods.
type SQLiteStore struct {
	*sql.DB
}

// NewSQLiteStore initializes a new SQLite connection and applies migrations.
// It takes the dataSourceName (file path) and the path to the migrations folder.
func NewSQLiteStore(dataSourceName, migrationsPath string) (*SQLiteStore, error) {
	dir := filepath.Dir(dataSourceName)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("failed to create database directory %q: %v", dir, err)
	}

	// Open the database connection
	dsn := dataSourceName + "?_foreign_keys=on&_journal_mode=WAL"

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// Verify the connection
	if err := db.Ping(); err != nil {
		db.Close() // Clean up if ping fails
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	// Apply migrations
	if err := applyMigrations(db, migrationsPath); err != nil {
		db.Close() // Clean up if migrations fail so as not to leak connections
		return nil, fmt.Errorf("failed to apply migrations: %v", err)
	}

	return &SQLiteStore{db}, nil
}

func applyMigrations(db *sql.DB, migrationsPath string) error {
	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %v", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://"+migrationsPath,
		"sqlite3", driver)
	if err != nil {
		return fmt.Errorf("failed to initialize migration: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to apply migrations: %v", err)
	}

	log.Println("Migrations applied successfully")
	return nil
}
