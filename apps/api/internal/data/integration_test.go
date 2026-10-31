package data_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zchelalo/cestia/apps/api/internal/data"
	cestia "github.com/zchelalo/cestia/apps/api/internal/db/sqlc/cestia"
	"github.com/zchelalo/cestia/apps/api/internal/migrations"
)

type testQuerier struct {
	queries *cestia.Queries
}

func (q testQuerier) Queries() *cestia.Queries { return q.queries }

func integrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	databaseURL := os.Getenv("CESTIA_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("CESTIA_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect test database: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := migrations.Up(ctx, pool); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	return pool
}

func rollbackTx(t *testing.T, pool *pgxpool.Pool) (context.Context, pgx.Tx, data.Querier) {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback(context.Background()) })
	return ctx, tx, testQuerier{queries: cestia.New(tx)}
}

func insertIntegrationUser(t *testing.T, ctx context.Context, q data.Querier) uuid.UUID {
	t.Helper()
	id := uuid.Must(uuid.NewV7())
	now := time.Now().UTC()
	if err := data.CreateUser(ctx, q, &data.User{
		ID: id, Email: "integration-" + id.String() + "@example.com", DisplayName: "Integration",
		PasswordHash: "$2a$10$integration", Locale: "es", Timezone: "America/Mexico_City",
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	return id
}

func integrationList(t *testing.T, ctx context.Context, q data.Querier, userID uuid.UUID) data.List {
	t.Helper()
	now := time.Now().UTC()
	list := data.List{
		ID: uuid.Must(uuid.NewV7()), Name: "Integration", PersonalOwnerID: &userID,
		CreatedBy: userID, CreatedAt: now, UpdatedAt: now,
	}
	if err := data.CreateList(ctx, q, list); err != nil {
		t.Fatalf("insert list: %v", err)
	}
	return list
}

func integrationItem(t *testing.T, ctx context.Context, q data.Querier, listID, userID uuid.UUID, quantity string) data.Item {
	t.Helper()
	now := time.Now().UTC()
	item := data.Item{
		ID: uuid.Must(uuid.NewV7()), ListID: listID, Name: "Item", Quantity: quantity,
		Unit: "unit", CreatedBy: userID, CreatedAt: now, UpdatedAt: now,
	}
	if err := data.CreateItem(ctx, q, item); err != nil {
		t.Fatalf("insert item: %v", err)
	}
	return item
}

func TestIntegrationGeneratedMoneyUsesTwoDecimalRounding(t *testing.T) {
	pool := integrationPool(t)
	ctx, _, q := rollbackTx(t, pool)
	userID := insertIntegrationUser(t, ctx, q)
	list := integrationList(t, ctx, q, userID)
	item := integrationItem(t, ctx, q, list.ID, userID, "0.125")
	price := "19.99"
	purchase := data.Purchase{
		ID: uuid.Must(uuid.NewV7()), ItemID: item.ID, ItemName: item.Name, Quantity: item.Quantity,
		Unit: item.Unit, ActualUnitPrice: &price, RecordedBy: userID, PurchasedBy: userID,
		PurchasedAt: time.Now().UTC(), CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	if err := data.CreatePurchase(ctx, q, purchase); err != nil {
		t.Fatalf("insert purchase: %v", err)
	}
	active, err := data.ActivePurchaseForItem(ctx, q, item.ID, false)
	if err != nil {
		t.Fatalf("read purchase: %v", err)
	}
	if active.TotalAmount == nil || *active.TotalAmount != "2.50" {
		t.Fatalf("total = %v, want 2.50", active.TotalAmount)
	}
}

func TestIntegrationOnlyOneActivePurchasePerItem(t *testing.T) {
	pool := integrationPool(t)
	ctx, _, q := rollbackTx(t, pool)
	userID := insertIntegrationUser(t, ctx, q)
	list := integrationList(t, ctx, q, userID)
	item := integrationItem(t, ctx, q, list.ID, userID, "1")
	price := "10"
	insert := func() error {
		now := time.Now().UTC()
		return data.CreatePurchase(ctx, q, data.Purchase{
			ID: uuid.Must(uuid.NewV7()), ItemID: item.ID, ItemName: item.Name,
			Quantity: item.Quantity, Unit: item.Unit, ActualUnitPrice: &price,
			RecordedBy: userID, PurchasedBy: userID, PurchasedAt: now, CreatedAt: now, UpdatedAt: now,
		})
	}
	if err := insert(); err != nil {
		t.Fatalf("first purchase: %v", err)
	}
	if err := insert(); err == nil {
		t.Fatal("second active purchase should violate the partial unique index")
	}
}

func TestIntegrationGroupAlwaysRetainsActiveOwner(t *testing.T) {
	pool := integrationPool(t)
	ctx, tx, q := rollbackTx(t, pool)
	ownerID := insertIntegrationUser(t, ctx, q)
	group := data.Group{
		ID: uuid.Must(uuid.NewV7()), Name: "Integration", OwnerUserID: ownerID,
		CreatedBy: ownerID, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	if err := data.CreateGroup(ctx, q, group); err != nil {
		t.Fatalf("insert group: %v", err)
	}
	if err := data.AddGroupMember(ctx, q, group.ID, ownerID, "admin", time.Now().UTC()); err != nil {
		t.Fatalf("insert owner membership: %v", err)
	}
	if err := data.RemoveGroupMember(ctx, q, group.ID, ownerID, time.Now().UTC()); err != nil {
		t.Fatalf("remove owner membership: %v", err)
	}
	if err := tx.Commit(ctx); err == nil {
		t.Fatal("commit should fail when the group loses its active owner")
	}
}

func TestIntegrationIdempotencyQueriesAreGeneratedAndScoped(t *testing.T) {
	pool := integrationPool(t)
	ctx, _, q := rollbackTx(t, pool)
	queries := q.Queries()
	actorID := insertIntegrationUser(t, ctx, q)
	now := time.Now().UTC()
	if err := queries.AcquireIdempotencyLock(ctx, actorID.String()+":integration:key"); err != nil {
		t.Fatalf("acquire advisory lock: %v", err)
	}
	if err := queries.UpsertIdempotencyRequest(ctx, cestia.UpsertIdempotencyRequestParams{
		ActorID: actorID, Operation: "integration", Key: "key", RequestHash: "hash",
		ResponseStatus: 200, ResponseBody: []byte(`{"ok":true}`), ResourceID: &actorID,
		ExpiresAt: now.Add(time.Hour), CreatedAt: now,
	}); err != nil {
		t.Fatalf("upsert idempotency: %v", err)
	}
	stored, err := queries.GetIdempotencyRequest(ctx, cestia.GetIdempotencyRequestParams{
		ActorID: actorID, Operation: "integration", Key: "key", Now: now,
	})
	if err != nil {
		t.Fatalf("get idempotency: %v", err)
	}
	var body map[string]bool
	if err := json.Unmarshal(stored.ResponseBody, &body); err != nil {
		t.Fatalf("decode idempotency body: %v", err)
	}
	if stored.RequestHash != "hash" || !body["ok"] {
		t.Fatalf("unexpected idempotency row: %#v", stored)
	}
}

func TestIntegrationFailedEmailDeliveryIsTerminal(t *testing.T) {
	pool := integrationPool(t)
	ctx, _, q := rollbackTx(t, pool)
	userID := insertIntegrationUser(t, ctx, q)
	now := time.Now().UTC()
	actionID := uuid.Must(uuid.NewV7())
	if err := data.CreateEmailAction(ctx, q, &data.EmailAction{
		ID: actionID, UserID: userID, Purpose: "verify_email", TokenHash: []byte("hash"),
		Generation: 1, ExpiresAt: now.Add(time.Hour),
	}, now); err != nil {
		t.Fatalf("create email action: %v", err)
	}
	deliveryID := uuid.Must(uuid.NewV7())
	if err := data.CreateEmailDeliveryForAction(ctx, q, deliveryID, actionID, "integration@example.com", 1, now); err != nil {
		t.Fatalf("create delivery: %v", err)
	}
	first, err := data.ClaimEmailDeliveries(ctx, q, uuid.Must(uuid.NewV7()), now, now.Add(time.Minute), 10)
	if err != nil {
		t.Fatalf("claim delivery: %v", err)
	}
	var job data.DeliveryJob
	for _, candidate := range first {
		if candidate.ID == deliveryID {
			job = candidate
			break
		}
	}
	if job.ID == uuid.Nil {
		t.Fatal("delivery was not claimed")
	}
	if _, err := data.MarkDeliveryFailed(ctx, q, job.ID, job.LeaseToken, "terminal failure", now); err != nil {
		t.Fatalf("mark delivery failed: %v", err)
	}
	second, err := data.ClaimEmailDeliveries(ctx, q, uuid.Must(uuid.NewV7()), now.Add(time.Hour), now.Add(time.Hour+time.Minute), 10)
	if err != nil {
		t.Fatalf("reclaim delivery: %v", err)
	}
	for _, candidate := range second {
		if candidate.ID == deliveryID {
			t.Fatal("failed delivery must not be reclaimed automatically")
		}
	}
}
