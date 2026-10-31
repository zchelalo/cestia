package migrations

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	cestia "github.com/zchelalo/cestia/apps/api/internal/db/sqlc/cestia"
)

//go:embed *.up.sql
var files embed.FS

const journalMigration = "000000_schema_migrations"

// Up applies every pending migration in one transaction while holding a
// database-wide advisory lock. Concurrent migrators serialize on the lock and
// either see the journal already advanced or wait for the winner's commit.
func Up(ctx context.Context, pool *pgxpool.Pool) error {
	entries, err := fs.Glob(files, "*.up.sql")
	if err != nil {
		return fmt.Errorf("list migrations: %w", err)
	}
	sort.Strings(entries)

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin migrations: %w", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()

	queries := cestia.New(tx)
	if err := queries.AcquireSchemaMigrationLock(ctx); err != nil {
		return fmt.Errorf("acquire migration lock: %w", err)
	}
	if err := bootstrapJournal(ctx, tx, queries); err != nil {
		return err
	}
	for _, name := range entries {
		version := strings.TrimSuffix(name, ".up.sql")
		if version == journalMigration {
			continue
		}
		applied, err := queries.SchemaMigrationExists(ctx, version)
		if err != nil {
			return fmt.Errorf("check migration %s: %w", version, err)
		}
		if applied {
			continue
		}
		body, err := files.ReadFile(name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		if _, err := tx.Exec(ctx, string(body)); err != nil {
			return fmt.Errorf("apply migration %s: %w", version, err)
		}
		if err := queries.RecordSchemaMigration(ctx, version); err != nil {
			return fmt.Errorf("record migration %s: %w", version, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit migrations: %w", err)
	}
	return nil
}

func bootstrapJournal(ctx context.Context, tx pgx.Tx, queries *cestia.Queries) error {
	body, err := files.ReadFile(journalMigration + ".up.sql")
	if err != nil {
		return fmt.Errorf("read migration journal: %w", err)
	}
	if _, err := tx.Exec(ctx, string(body)); err != nil {
		return fmt.Errorf("bootstrap migration journal: %w", err)
	}
	if err := queries.RecordSchemaMigration(ctx, journalMigration); err != nil {
		return fmt.Errorf("record migration journal: %w", err)
	}
	return nil
}
