package data_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zchelalo/cestia/apps/api/internal/data"
	cestia "github.com/zchelalo/cestia/apps/api/internal/db/sqlc/cestia"
	"github.com/zchelalo/cestia/apps/api/internal/migrations"
)

func integrationGroup(t *testing.T, ctx context.Context, store *data.Store, ownerID uuid.UUID) uuid.UUID {
	t.Helper()
	now := time.Now().UTC()
	group := data.Group{
		ID: uuid.Must(uuid.NewV7()), Name: "Concurrency", OwnerUserID: ownerID,
		CreatedBy: ownerID, CreatedAt: now, UpdatedAt: now,
	}
	if err := store.Within(ctx, func(q data.Querier) error {
		if err := data.CreateGroup(ctx, q, group); err != nil {
			return err
		}
		return data.AddGroupMember(ctx, q, group.ID, ownerID, "admin", now)
	}); err != nil {
		t.Fatalf("insert group: %v", err)
	}
	return group.ID
}

func waitForApplicationLock(t *testing.T, pool *pgxpool.Pool, name string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		waiting, err := cestia.New(pool).CountLockWaitsForApplication(context.Background(), name)
		if err != nil {
			t.Fatalf("count lock waits: %v", err)
		}
		if waiting > 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("application %q never waited on a lock", name)
}

func TestIntegrationGroupLockReReadsMembership(t *testing.T) {
	pool := integrationPool(t)
	ctx := context.Background()
	store := data.NewFromPool(pool)
	ownerID := insertIntegrationUser(t, ctx, store)
	memberID := insertIntegrationUser(t, ctx, store)
	groupID := integrationGroup(t, ctx, store, ownerID)
	now := time.Now().UTC()
	if err := data.AddGroupMember(ctx, store, groupID, memberID, "admin", now); err != nil {
		t.Fatalf("add member: %v", err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin owner tx: %v", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	ownerQuerier := testQuerier{queries: cestia.New(tx)}
	if _, err := data.GroupForUserLocked(ctx, ownerQuerier, groupID, ownerID); err != nil {
		t.Fatalf("lock group: %v", err)
	}
	if err := data.RemoveGroupMember(ctx, ownerQuerier, groupID, memberID, now); err != nil {
		t.Fatalf("remove member: %v", err)
	}

	type outcome struct {
		err error
	}
	done := make(chan outcome, 1)
	entered := make(chan struct{})
	appName := "cestia-group-waiter-" + uuid.Must(uuid.NewV7()).String()
	go func() {
		memberTx, err := pool.Begin(context.Background())
		if err != nil {
			done <- outcome{err: err}
			return
		}
		defer func() { _ = memberTx.Rollback(context.Background()) }()
		queries := cestia.New(memberTx)
		if err := queries.SetLocalApplicationName(context.Background(), appName); err != nil {
			done <- outcome{err: err}
			return
		}
		close(entered)
		_, err = data.GroupForUserLocked(context.Background(), testQuerier{queries: queries}, groupID, memberID)
		done <- outcome{err: err}
	}()

	<-entered
	waitForApplicationLock(t, pool, appName)
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit owner tx: %v", err)
	}
	result := <-done
	if !errors.Is(result.err, pgx.ErrNoRows) {
		t.Fatalf("removed member kept authorization after lock wait: %v", result.err)
	}
}

func TestIntegrationTransferAndDeactivationSerialize(t *testing.T) {
	pool := integrationPool(t)
	ctx := context.Background()
	store := data.NewFromPool(pool)
	ownerID := insertIntegrationUser(t, ctx, store)
	targetID := insertIntegrationUser(t, ctx, store)
	groupID := integrationGroup(t, ctx, store, ownerID)
	now := time.Now().UTC()
	if err := data.AddGroupMember(ctx, store, groupID, targetID, "member", now); err != nil {
		t.Fatalf("add target: %v", err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin transfer: %v", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	transferQuerier := testQuerier{queries: cestia.New(tx)}
	if _, err := data.UserForShare(ctx, transferQuerier, targetID); err != nil {
		t.Fatalf("lock target user: %v", err)
	}
	if err := data.TransferOwner(ctx, transferQuerier, groupID, ownerID, targetID, now); err != nil {
		t.Fatalf("transfer owner: %v", err)
	}

	done := make(chan int, 1)
	entered := make(chan struct{})
	appName := "cestia-deactivate-waiter-" + uuid.Must(uuid.NewV7()).String()
	go func() {
		deactivateTx, err := pool.Begin(context.Background())
		if err != nil {
			done <- -1
			return
		}
		defer func() { _ = deactivateTx.Rollback(context.Background()) }()
		queries := cestia.New(deactivateTx)
		if err := queries.SetLocalApplicationName(context.Background(), appName); err != nil {
			done <- -1
			return
		}
		q := testQuerier{queries: queries}
		close(entered)
		if _, err := data.UserForUpdate(context.Background(), q, targetID); err != nil {
			done <- -1
			return
		}
		owned, err := data.CountOwnedGroups(context.Background(), q, targetID)
		if err != nil {
			done <- -1
			return
		}
		done <- owned
	}()

	<-entered
	waitForApplicationLock(t, pool, appName)
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit transfer: %v", err)
	}
	if owned := <-done; owned != 1 {
		t.Fatalf("deactivation must observe the transferred group, owned=%d", owned)
	}
}

func TestIntegrationReadSnapshotIsStable(t *testing.T) {
	pool := integrationPool(t)
	ctx := context.Background()
	store := data.NewFromPool(pool)
	userID := insertIntegrationUser(t, ctx, store)

	ready := make(chan struct{})
	proceed := make(chan struct{})
	done := make(chan error, 1)
	var first, second string
	go func() {
		done <- store.WithinReadSnapshot(context.Background(), func(q data.Querier) error {
			user, err := data.UserByID(context.Background(), q, userID)
			if err != nil {
				return err
			}
			first = user.DisplayName
			close(ready)
			<-proceed
			user, err = data.UserByID(context.Background(), q, userID)
			if err != nil {
				return err
			}
			second = user.DisplayName
			return nil
		})
	}()
	<-ready
	if _, err := data.UpdateProfile(ctx, store, userID, "Changed", "es", "America/Mexico_City", time.Now().UTC()); err != nil {
		t.Fatalf("update profile: %v", err)
	}
	close(proceed)
	if err := <-done; err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if first != second || first != "Integration" {
		t.Fatalf("snapshot changed mid-transaction: first=%q second=%q", first, second)
	}
}

func TestIntegrationOnlyOneActiveEmailActionPerPurpose(t *testing.T) {
	pool := integrationPool(t)
	ctx := context.Background()
	store := data.NewFromPool(pool)
	userID := insertIntegrationUser(t, ctx, store)
	now := time.Now().UTC()

	create := func() error {
		return data.CreateEmailAction(ctx, store, &data.EmailAction{
			ID: uuid.Must(uuid.NewV7()), UserID: userID, Purpose: "reset_password",
			TokenHash: []byte(uuid.Must(uuid.NewV7()).String()), Generation: 1,
			ExpiresAt: now.Add(time.Hour),
		}, now)
	}
	if err := create(); err != nil {
		t.Fatalf("first email action: %v", err)
	}
	var pgErr *pgconn.PgError
	if err := create(); !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		t.Fatalf("second active action should violate the unique index, got %v", err)
	}
}

func TestIntegrationDeactivatedOwnerIsRejected(t *testing.T) {
	pool := integrationPool(t)
	ctx := context.Background()
	store := data.NewFromPool(pool)
	ownerID := insertIntegrationUser(t, ctx, store)
	integrationGroup(t, ctx, store, ownerID)

	err := store.Within(ctx, func(q data.Querier) error {
		return data.DeactivateUser(ctx, q, ownerID, time.Now().UTC())
	})
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23514" {
		t.Fatalf("deactivating a group owner should be rejected, got %v", err)
	}
}

func TestIntegrationConcurrentMigratorsSerialize(t *testing.T) {
	pool := integrationPool(t)
	ctx := context.Background()

	var wait sync.WaitGroup
	errs := make(chan error, 4)
	for range 4 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			errs <- migrations.Up(ctx, pool)
		}()
	}
	wait.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent migration: %v", err)
		}
	}
}
