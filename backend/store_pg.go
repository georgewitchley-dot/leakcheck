//go:build !nopg

package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const schema = `
CREATE TABLE IF NOT EXISTS range_cache (
	prefix     CHAR(5) PRIMARY KEY,
	body       TEXT NOT NULL,
	fetched_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS check_stats (
	day    DATE PRIMARY KEY,
	checks BIGINT NOT NULL DEFAULT 0
);`

type PGStore struct{ pool *pgxpool.Pool }

func openStore(ctx context.Context, url string) (Store, error) {
	if url == "" {
		return nil, errors.New("DATABASE_URL is not set")
	}
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	cfg.MaxConns = 10

	var pool *pgxpool.Pool
	// Retry while Postgres starts up (compose / fresh deploys).
	for i := 0; i < 30; i++ {
		pool, err = pgxpool.NewWithConfig(ctx, cfg)
		if err == nil {
			if err = pool.Ping(ctx); err == nil {
				break
			}
			pool.Close()
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	if _, err := pool.Exec(ctx, schema); err != nil {
		pool.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return &PGStore{pool: pool}, nil
}

func (s *PGStore) GetRange(ctx context.Context, prefix string) (string, time.Time, bool, error) {
	var body string
	var at time.Time
	err := s.pool.QueryRow(ctx, `SELECT body, fetched_at FROM range_cache WHERE prefix = $1`, prefix).Scan(&body, &at)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", time.Time{}, false, nil
	}
	if err != nil {
		return "", time.Time{}, false, err
	}
	return body, at, true, nil
}

func (s *PGStore) PutRange(ctx context.Context, prefix, body string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO range_cache (prefix, body, fetched_at) VALUES ($1, $2, now())
		ON CONFLICT (prefix) DO UPDATE SET body = EXCLUDED.body, fetched_at = EXCLUDED.fetched_at`, prefix, body)
	return err
}

func (s *PGStore) IncrementChecks(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO check_stats (day, checks) VALUES (CURRENT_DATE, 1)
		ON CONFLICT (day) DO UPDATE SET checks = check_stats.checks + 1`)
	return err
}

func (s *PGStore) TotalChecks(ctx context.Context) (int64, error) {
	var n int64
	err := s.pool.QueryRow(ctx, `SELECT COALESCE(SUM(checks), 0)::BIGINT FROM check_stats`).Scan(&n)
	return n, err
}

func (s *PGStore) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }
func (s *PGStore) Close()                         { s.pool.Close() }
