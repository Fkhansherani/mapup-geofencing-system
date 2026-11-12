// Package db opens the PostgreSQL connection pool and creates the tables.
package db

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// schema.sql is compiled into the binary, so the Docker image needs no extra files.
//
//go:embed schema.sql
var schema string

// Connect opens a pool, waits until the database answers (useful when Docker
// starts the database and the API at the same time) and applies the schema.
func Connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	deadline := time.Now().Add(30 * time.Second)
	for {
		err = pool.Ping(ctx)
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			pool.Close()
			return nil, fmt.Errorf("database not reachable: %w", err)
		}
		log.Printf("waiting for database: %v", err)

		select {
		case <-ctx.Done():
			pool.Close()
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}

	if _, err := pool.Exec(ctx, schema); err != nil {
		pool.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}

	return pool, nil
}
