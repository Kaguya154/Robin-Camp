package internal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func InitDB() {
	var err error
	dsn := os.Getenv("DB_URL")
	DB, err = initDB(context.Background(), dsn)
	if err != nil {
		panic(fmt.Sprintf("failed to initialize database: %v", err))
	}
}

var pragmas = []string{
	"PRAGMA foreign_keys = ON",
	"PRAGMA journal_mode = WAL",
	"PRAGMA synchronous = NORMAL",
	"PRAGMA busy_timeout = 5000",
}

var migrationStatements = []string{
	`CREATE TABLE IF NOT EXISTS movies (
        id TEXT PRIMARY KEY,
        title TEXT NOT NULL UNIQUE,
        release_date TEXT NOT NULL,
        genre TEXT NOT NULL,
        distributor TEXT,
        budget INTEGER,
        mpa_rating TEXT,
        created_at TEXT NOT NULL DEFAULT (STRFTIME('%Y-%m-%dT%H:%M:%fZ', 'now')),
        updated_at TEXT NOT NULL DEFAULT (STRFTIME('%Y-%m-%dT%H:%M:%fZ', 'now'))
    )`,
	`CREATE INDEX IF NOT EXISTS idx_movies_release_date ON movies(release_date)`,
	`CREATE INDEX IF NOT EXISTS idx_movies_genre ON movies(genre)`,
	`CREATE INDEX IF NOT EXISTS idx_movies_created_at ON movies(created_at)`,
	`CREATE TABLE IF NOT EXISTS box_office (
        movie_id TEXT PRIMARY KEY,
        currency TEXT NOT NULL,
        source TEXT NOT NULL,
        last_updated TEXT NOT NULL,
        revenue_worldwide INTEGER NOT NULL,
        revenue_opening_weekend_usa INTEGER,
        FOREIGN KEY(movie_id) REFERENCES movies(id) ON DELETE CASCADE
    )`,
	`CREATE TABLE IF NOT EXISTS ratings (
        movie_id TEXT NOT NULL,
        rater_id TEXT NOT NULL,
        rating REAL NOT NULL CHECK (rating >= 0.5 AND rating <= 5.0),
        updated_at TEXT NOT NULL DEFAULT (STRFTIME('%Y-%m-%dT%H:%M:%fZ', 'now')),
        PRIMARY KEY (movie_id, rater_id),
        FOREIGN KEY(movie_id) REFERENCES movies(id) ON DELETE CASCADE
    )`,
	`CREATE INDEX IF NOT EXISTS idx_ratings_movie ON ratings(movie_id)`,
}

func initDB(ctx context.Context, dsn string) (*sql.DB, error) {
	if dsn == "" {
		return nil, errors.New("sqlite dsn is empty")
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	if err := applyPragmas(ctx, db); err != nil {
		db.Close()
		return nil, err
	}

	if err := runMigrations(ctx, db); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func applyPragmas(ctx context.Context, db *sql.DB) error {
	for _, stmt := range pragmas {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("apply %s: %w", stmt, err)
		}
	}
	return nil
}

func runMigrations(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin migration tx: %w", err)
	}

	for _, stmt := range migrationStatements {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			tx.Rollback()
			return fmt.Errorf("execute migration: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migrations: %w", err)
	}

	return nil
}

// WithTx provides a helper for running code inside a transaction with shared settings.
func WithTx(ctx context.Context, db *sql.DB, fn func(context.Context, *sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	if err := fn(ctx, tx); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

func Now() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}
