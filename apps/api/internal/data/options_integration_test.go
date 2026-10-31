package data_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zchelalo/cestia/apps/api/internal/data"
)

func TestIntegrationOptionCursorHandlesDelimitersAndRejectsInvalid(t *testing.T) {
	pool := integrationPool(t)
	ctx := context.Background()
	store := data.NewFromPool(pool)
	ownerID := insertIntegrationUser(t, ctx, store)
	now := time.Now().UTC()

	for _, name := range []string{"Casa | Playa", "Despensa"} {
		group := data.Group{
			ID: uuid.Must(uuid.NewV7()), Name: name, OwnerUserID: ownerID,
			CreatedBy: ownerID, CreatedAt: now, UpdatedAt: now,
		}
		if err := store.Within(ctx, func(q data.Querier) error {
			if err := data.CreateGroup(ctx, q, group); err != nil {
				return err
			}
			return data.AddGroupMember(ctx, q, group.ID, ownerID, "admin", now)
		}); err != nil {
			t.Fatalf("create group %s: %v", name, err)
		}
	}

	first, cursor, err := data.ListGroupOptions(ctx, store, ownerID, "", "", 1)
	if err != nil {
		t.Fatalf("first page: %v", err)
	}
	if len(first) != 1 || cursor == "" {
		t.Fatalf("expected one group and a cursor, got %d and %q", len(first), cursor)
	}
	second, _, err := data.ListGroupOptions(ctx, store, ownerID, "", cursor, 1)
	if err != nil {
		t.Fatalf("second page after pipe in name: %v", err)
	}
	if len(second) != 1 || second[0].Value == first[0].Value {
		t.Fatalf("cursor did not advance correctly: %#v %#v", first, second)
	}

	if _, _, err := data.ListGroupOptions(ctx, store, ownerID, "", "not-a-cursor", 1); !errors.Is(err, data.ErrInvalidOptionCursor) {
		t.Fatalf("invalid cursor should be rejected, got %v", err)
	}
}

func TestIntegrationGroupAndMemberOptionsPagingAndSearch(t *testing.T) {
	pool := integrationPool(t)
	ctx := context.Background()
	store := data.NewFromPool(pool)
	now := time.Now().UTC()

	ownerID := insertIntegrationUser(t, ctx, store)
	groupA := data.Group{
		ID: uuid.Must(uuid.NewV7()), Name: "Alacena Principal", OwnerUserID: ownerID,
		CreatedBy: ownerID, CreatedAt: now, UpdatedAt: now,
	}
	groupB := data.Group{
		ID: uuid.Must(uuid.NewV7()), Name: "Bebidas y Vinos", OwnerUserID: ownerID,
		CreatedBy: ownerID, CreatedAt: now, UpdatedAt: now,
	}
	groupC := data.Group{
		ID: uuid.Must(uuid.NewV7()), Name: "Carnicería", OwnerUserID: ownerID,
		CreatedBy: ownerID, CreatedAt: now, UpdatedAt: now,
	}

	for _, g := range []data.Group{groupA, groupB, groupC} {
		if err := store.Within(ctx, func(q data.Querier) error {
			if err := data.CreateGroup(ctx, q, g); err != nil {
				return err
			}
			return data.AddGroupMember(ctx, q, g.ID, ownerID, "admin", now)
		}); err != nil {
			t.Fatalf("create group %s: %v", g.Name, err)
		}
	}

	page1, next1, err := data.ListGroupOptions(ctx, store, ownerID, "", "", 2)
	if err != nil {
		t.Fatalf("list group options page 1: %v", err)
	}
	if len(page1) != 2 || next1 == "" {
		t.Fatalf("expected 2 items and a cursor, got %d items, cursor=%q", len(page1), next1)
	}

	page2, next2, err := data.ListGroupOptions(ctx, store, ownerID, "", next1, 2)
	if err != nil {
		t.Fatalf("list group options page 2: %v", err)
	}
	if len(page2) == 0 {
		t.Fatalf("expected at least 1 item on page 2")
	}
	_ = next2

	searched, _, err := data.ListGroupOptions(ctx, store, ownerID, "Bebidas", "", 10)
	if err != nil {
		t.Fatalf("search group options: %v", err)
	}
	if len(searched) != 1 || searched[0].Value != groupB.ID.String() {
		t.Fatalf("expected 1 searched group, got %d", len(searched))
	}

	// Member options
	member2ID := insertIntegrationUser(t, ctx, store)
	if err := data.AddGroupMember(ctx, store, groupA.ID, member2ID, "member", now); err != nil {
		t.Fatalf("add member to group A: %v", err)
	}

	memberOpts, _, err := data.ListGroupMemberOptions(ctx, store, groupA.ID, "", "", 10)
	if err != nil {
		t.Fatalf("list member options: %v", err)
	}
	if len(memberOpts) != 2 {
		t.Fatalf("expected 2 member options in group A, got %d", len(memberOpts))
	}
}
