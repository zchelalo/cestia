package data

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	cestia "github.com/zchelalo/cestia/apps/api/internal/db/sqlc/cestia"
)

type Querier interface {
	Queries() *cestia.Queries
}

type Store struct {
	Pool    *pgxpool.Pool
	queries *cestia.Queries
}

func New(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return NewFromPool(pool), nil
}

func NewFromPool(pool *pgxpool.Pool) *Store {
	return &Store{Pool: pool, queries: cestia.New(pool)}
}

func (s *Store) Queries() *cestia.Queries { return s.queries }

func (s *Store) Close() { s.Pool.Close() }

func (s *Store) Within(ctx context.Context, fn func(Querier) error) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if err := fn(&txQuerier{queries: cestia.New(tx)}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

// WithinReadSnapshot runs read-only work against a single repeatable-read
// snapshot. Multi-query reports use it so their totals cannot mix two
// different database states.
func (s *Store) WithinReadSnapshot(ctx context.Context, fn func(Querier) error) error {
	tx, err := s.Pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel:   pgx.RepeatableRead,
		AccessMode: pgx.ReadOnly,
	})
	if err != nil {
		return fmt.Errorf("begin read snapshot: %w", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if err := fn(&txQuerier{queries: cestia.New(tx)}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit read snapshot: %w", err)
	}
	return nil
}

type txQuerier struct {
	queries *cestia.Queries
}

func (q *txQuerier) Queries() *cestia.Queries { return q.queries }
