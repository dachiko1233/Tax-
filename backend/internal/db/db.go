// Package db owns the PostgreSQL connection pool and all SQL queries.
package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when a row does not exist (or is not owned by the
// requesting user).
var ErrNotFound = errors.New("not found")

type DB struct {
	Pool *pgxpool.Pool
}

// Connect opens a pooled connection, retrying briefly so the backend can start
// alongside a Postgres container that is still coming up.
func Connect(ctx context.Context, url string) (*DB, error) {
	var pool *pgxpool.Pool
	var err error
	for attempt := 0; attempt < 10; attempt++ {
		pool, err = pgxpool.New(ctx, url)
		if err == nil {
			if pingErr := pool.Ping(ctx); pingErr == nil {
				return &DB{Pool: pool}, nil
			} else {
				err = pingErr
				pool.Close()
			}
		}
		time.Sleep(2 * time.Second)
	}
	return nil, fmt.Errorf("could not connect to database: %w", err)
}

func (d *DB) Close() { d.Pool.Close() }

// mapNoRows converts pgx's ErrNoRows to our ErrNotFound.
func mapNoRows(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
