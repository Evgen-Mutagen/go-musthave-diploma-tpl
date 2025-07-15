package repository

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

type Database struct {
	db *sql.DB
}

func NewDatabase(dsn string) (*Database, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	database := &Database{db: db}

	if err := database.Migrate(); err != nil {
		return nil, fmt.Errorf("failed to apply migrations: %w", err)
	}

	return database, nil
}

func (d *Database) Migrate() error {
	driver, err := postgres.WithInstance(d.db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create driver: %w", err)
	}

	absPath, err := filepath.Abs("migrations")
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	migrationsURL := "file://" + filepath.ToSlash(absPath)
	fmt.Printf("Migrations URL: %s\n", migrationsURL) // Для отладки

	m, err := migrate.NewWithDatabaseInstance(
		migrationsURL,
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	return nil
}

func (d *Database) Close() error {
	return d.db.Close()
}

func (d *Database) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return d.db.BeginTx(ctx, nil)
}
