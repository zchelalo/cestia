package shopping

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zchelalo/cestia/apps/api/internal/apperr"
	"github.com/zchelalo/cestia/apps/api/internal/data"
	cestia "github.com/zchelalo/cestia/apps/api/internal/db/sqlc/cestia"
	"github.com/zchelalo/cestia/apps/api/internal/migrations"
)

type idempotencyFixture struct {
	ctx   context.Context
	store *data.Store
	user  data.User
	list  data.List
	items []data.Item
	now   time.Time
}

func newIdempotencyFixture(t *testing.T, itemCount int) idempotencyFixture {
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

	store := data.NewFromPool(pool)
	now := time.Now().UTC()
	user := data.User{
		ID: uuid.Must(uuid.NewV7()), Email: "idem-" + uuid.Must(uuid.NewV7()).String() + "@example.com",
		DisplayName: "Idempotency", PasswordHash: "$2a$10$integration", Locale: "es",
		Timezone: "America/Mexico_City", CreatedAt: now, UpdatedAt: now,
	}
	if err := data.CreateUser(ctx, store, &user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	list := data.List{
		ID: uuid.Must(uuid.NewV7()), Name: "Idempotency", PersonalOwnerID: &user.ID,
		CreatedBy: user.ID, CreatedAt: now, UpdatedAt: now,
	}
	if err := data.CreateList(ctx, store, list); err != nil {
		t.Fatalf("create list: %v", err)
	}
	items := make([]data.Item, itemCount)
	for i := range items {
		items[i] = data.Item{
			ID: uuid.Must(uuid.NewV7()), ListID: list.ID, Name: "Item", Quantity: "1",
			Unit: "unit", CreatedBy: user.ID, CreatedAt: now, UpdatedAt: now,
		}
		if err := data.CreateItem(ctx, store, items[i]); err != nil {
			t.Fatalf("create item: %v", err)
		}
	}
	return idempotencyFixture{ctx: ctx, store: store, user: user, list: list, items: items, now: now}
}

func TestIntegrationIdempotencyKeyIsBoundToItem(t *testing.T) {
	fixture := newIdempotencyFixture(t, 2)
	service := New(fixture.store)
	key := "integration-" + uuid.Must(uuid.NewV7()).String()

	for index, item := range fixture.items {
		_, err := service.RecordPurchase(fixture.ctx, fixture.user.ID, item.ID, PurchaseInput{
			ExpectedVersion: 1,
			ActualUnitPrice: "10",
			PurchasedBy:     fixture.user.ID,
			PaidBy:          &fixture.user.ID,
			PurchasedAt:     fixture.now,
			IdempotencyKey:  key,
		})
		if index == 0 {
			if err != nil {
				t.Fatalf("first purchase: %v", err)
			}
			continue
		}
		var appErr *apperr.Error
		if !errors.As(err, &appErr) || appErr.Code != "idempotency_key_reused" {
			t.Fatalf("second resource should conflict, got %v", err)
		}
	}
}

func TestIntegrationExpiredIdempotencyKeyRenewsResource(t *testing.T) {
	fixture := newIdempotencyFixture(t, 2)
	queries := fixture.store.Queries()
	key := "renew-" + uuid.Must(uuid.NewV7()).String()
	expired := fixture.now.Add(-time.Minute)

	if err := queries.UpsertIdempotencyRequest(fixture.ctx, cestia.UpsertIdempotencyRequestParams{
		ActorID: fixture.user.ID, Operation: "purchase.record", Key: key, RequestHash: "expired",
		ResponseStatus: 200, ResponseBody: []byte(`{"ok":true}`),
		ResourceID: &fixture.items[0].ID, ExpiresAt: expired, CreatedAt: expired,
	}); err != nil {
		t.Fatalf("store expired idempotency record: %v", err)
	}
	if _, err := queries.GetIdempotencyRequest(fixture.ctx, cestia.GetIdempotencyRequestParams{
		ActorID: fixture.user.ID, Operation: "purchase.record", Key: key, Now: fixture.now,
	}); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("expired record must be invisible, got %v", err)
	}

	if err := queries.UpsertIdempotencyRequest(fixture.ctx, cestia.UpsertIdempotencyRequestParams{
		ActorID: fixture.user.ID, Operation: "purchase.record", Key: key, RequestHash: "renewed",
		ResponseStatus: 200, ResponseBody: []byte(`{"ok":true}`),
		ResourceID: &fixture.items[1].ID, ExpiresAt: fixture.now.Add(time.Hour), CreatedAt: fixture.now,
	}); err != nil {
		t.Fatalf("renew idempotency record: %v", err)
	}
	stored, err := queries.GetIdempotencyRequest(fixture.ctx, cestia.GetIdempotencyRequestParams{
		ActorID: fixture.user.ID, Operation: "purchase.record", Key: key, Now: fixture.now,
	})
	if err != nil {
		t.Fatalf("load renewed record: %v", err)
	}
	if stored.ResourceID == nil || *stored.ResourceID != fixture.items[1].ID {
		t.Fatalf("renewed record kept the previous resource: %#v", stored.ResourceID)
	}
}

func storeLegacyIdempotencyRecord(
	t *testing.T,
	fixture idempotencyFixture,
	operation, key string,
	body []byte,
	storedPurchase data.Purchase,
) {
	t.Helper()

	response, err := json.Marshal(storedPurchase)
	if err != nil {
		t.Fatalf("marshal legacy response: %v", err)
	}
	sum := sha256.Sum256(body)
	legacyHash := hex.EncodeToString(sum[:])
	if err := fixture.store.Queries().UpsertIdempotencyRequest(fixture.ctx, cestia.UpsertIdempotencyRequestParams{
		ActorID: fixture.user.ID, Operation: operation, Key: key, RequestHash: legacyHash,
		ResponseStatus: 200, ResponseBody: response, ResourceID: nil,
		ExpiresAt: fixture.now.Add(time.Hour), CreatedAt: fixture.now,
	}); err != nil {
		t.Fatalf("store legacy idempotency record: %v", err)
	}
}

func legacyPurchase(fixture idempotencyFixture, item data.Item) data.Purchase {
	priceText := "10.0000"
	return data.Purchase{
		ID: uuid.Must(uuid.NewV7()), ItemID: item.ID, ListID: item.ListID, ItemName: item.Name,
		Quantity: item.Quantity, Unit: item.Unit, ActualUnitPrice: &priceText, Currency: "MXN",
		RecordedBy: fixture.user.ID, PurchasedBy: fixture.user.ID, PaidBy: &fixture.user.ID,
		PurchasedAt: fixture.now, Version: 1, CreatedAt: fixture.now, UpdatedAt: fixture.now,
	}
}

func TestIntegrationLegacyIdempotencyRecordIsUpgraded(t *testing.T) {
	fixture := newIdempotencyFixture(t, 1)
	item := fixture.items[0]
	key := "legacy-" + uuid.Must(uuid.NewV7()).String()
	storedPurchase := legacyPurchase(fixture, item)
	input := PurchaseInput{
		ExpectedVersion: 1, ActualUnitPrice: "10", PurchasedBy: fixture.user.ID,
		PaidBy: &fixture.user.ID, PurchasedAt: fixture.now, IdempotencyKey: key,
	}
	body, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal legacy request: %v", err)
	}
	storeLegacyIdempotencyRecord(t, fixture, "purchase.record", key, body, storedPurchase)

	queries := fixture.store.Queries()
	service := New(fixture.store)
	recovered, err := service.RecordPurchase(fixture.ctx, fixture.user.ID, item.ID, input)
	if err != nil {
		t.Fatalf("recover legacy record: %v", err)
	}
	if recovered.ID != storedPurchase.ID {
		t.Fatalf("recovered purchase = %s, want %s", recovered.ID, storedPurchase.ID)
	}
	if _, err := data.ActivePurchaseForItem(fixture.ctx, fixture.store, item.ID, false); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("legacy replay must not record a new purchase, got %v", err)
	}

	stored, err := queries.GetIdempotencyRequest(fixture.ctx, cestia.GetIdempotencyRequestParams{
		ActorID: fixture.user.ID, Operation: "purchase.record", Key: key, Now: fixture.now,
	})
	if err != nil {
		t.Fatalf("load upgraded record: %v", err)
	}
	if stored.ResourceID == nil || *stored.ResourceID != item.ID {
		t.Fatalf("legacy record was not bound to its resource: %#v", stored.ResourceID)
	}
	if want := requestFingerprint("purchase.record", item.ID, body); stored.RequestHash != want {
		t.Fatalf("legacy record was not upgraded to the strict fingerprint")
	}

	replayed, err := service.RecordPurchase(fixture.ctx, fixture.user.ID, item.ID, input)
	if err != nil {
		t.Fatalf("replay upgraded record: %v", err)
	}
	if replayed.ID != storedPurchase.ID {
		t.Fatalf("replayed purchase = %s, want %s", replayed.ID, storedPurchase.ID)
	}
}

func TestIntegrationPurchaseHistorySurvivesItemDeletion(t *testing.T) {
	fixture := newIdempotencyFixture(t, 1)
	item := fixture.items[0]
	service := New(fixture.store)

	purchase, err := service.RecordPurchase(fixture.ctx, fixture.user.ID, item.ID, PurchaseInput{
		ExpectedVersion: 1, ActualUnitPrice: "10", PurchasedBy: fixture.user.ID,
		PaidBy: &fixture.user.ID, PurchasedAt: fixture.now,
		IdempotencyKey: "delete-item-purchase-" + uuid.Must(uuid.NewV7()).String(),
	})
	if err != nil {
		t.Fatalf("record purchase: %v", err)
	}
	current, _, err := data.ItemForUser(fixture.ctx, fixture.store, item.ID, fixture.user.ID, false)
	if err != nil {
		t.Fatalf("read item: %v", err)
	}
	if err := service.DeleteItem(fixture.ctx, fixture.user.ID, item.ID, current.Version); err != nil {
		t.Fatalf("delete item: %v", err)
	}

	corrected, err := service.CorrectPurchase(fixture.ctx, fixture.user.ID, purchase.ID, CorrectionInput{
		ExpectedVersion: purchase.Version, ActualUnitPrice: "12", PurchasedBy: fixture.user.ID,
		PaidBy: &fixture.user.ID, PurchasedAt: fixture.now, Reason: "Corrección histórica",
		IdempotencyKey: "delete-item-correction-" + uuid.Must(uuid.NewV7()).String(),
	})
	if err != nil {
		t.Fatalf("correct purchase of a deleted item: %v", err)
	}
	if corrected.ActualUnitPrice == nil || *corrected.ActualUnitPrice != "12.0000" {
		t.Fatalf("corrected price = %v, want 12.0000", corrected.ActualUnitPrice)
	}
}

func TestIntegrationLegacyIdempotencyRecordRejectsOtherResource(t *testing.T) {
	fixture := newIdempotencyFixture(t, 2)
	itemA, itemB := fixture.items[0], fixture.items[1]
	key := "legacy-other-" + uuid.Must(uuid.NewV7()).String()
	input := PurchaseInput{
		ExpectedVersion: 1, ActualUnitPrice: "10", PurchasedBy: fixture.user.ID,
		PaidBy: &fixture.user.ID, PurchasedAt: fixture.now, IdempotencyKey: key,
	}
	body, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal legacy request: %v", err)
	}
	storeLegacyIdempotencyRecord(t, fixture, "purchase.record", key, body, legacyPurchase(fixture, itemA))

	service := New(fixture.store)
	_, err = service.RecordPurchase(fixture.ctx, fixture.user.ID, itemB.ID, input)
	var appErr *apperr.Error
	if !errors.As(err, &appErr) || appErr.Code != "idempotency_key_reused" {
		t.Fatalf("legacy record for another resource should conflict, got %v", err)
	}
	if _, err := data.ActivePurchaseForItem(fixture.ctx, fixture.store, itemB.ID, false); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("conflicting request must not record a purchase, got %v", err)
	}
}
