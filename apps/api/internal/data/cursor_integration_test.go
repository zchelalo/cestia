package data_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zchelalo/cestia/apps/api/internal/data"
)

func collectPage[T any](t *testing.T, load func(cursor string) ([]T, string, error), cursor string) ([]T, string) {
	t.Helper()
	items, next, err := load(cursor)
	if err != nil {
		t.Fatalf("load page: %v", err)
	}
	return items, next
}

func TestIntegrationCursorPages(t *testing.T) {
	pool := integrationPool(t)
	ctx := context.Background()
	store := data.NewFromPool(pool)
	userID := insertIntegrationUser(t, ctx, store)
	now := time.Now().UTC()
	suffix := now.Format("150405.000000")

	for index, name := range []string{"Alfa", "Beta", "Gamma", "Delta"} {
		list := data.List{
			ID: uuid.Must(uuid.NewV7()), Name: name + " " + suffix, PersonalOwnerID: &userID,
			CreatedBy: userID, CreatedAt: now.Add(time.Duration(index) * time.Second),
			UpdatedAt: now.Add(time.Duration(index) * time.Second),
		}
		if err := data.CreateList(ctx, store, list); err != nil {
			t.Fatalf("create list: %v", err)
		}
	}
	listsPage1, listsCursor := collectPage(t, func(cursor string) ([]data.List, string, error) {
		return data.ListsForScope(ctx, store, userID, nil, false, cursor, 3)
	}, "")
	if len(listsPage1) != 3 || listsCursor == "" {
		t.Fatalf("lists first page: %d items, cursor %q", len(listsPage1), listsCursor)
	}
	listsPage2, listsNext := collectPage(t, func(cursor string) ([]data.List, string, error) {
		return data.ListsForScope(ctx, store, userID, nil, false, cursor, 3)
	}, listsCursor)
	if len(listsPage2) != 1 || listsNext != "" {
		t.Fatalf("lists second page: %d items, cursor %q", len(listsPage2), listsNext)
	}
	for _, first := range listsPage1 {
		if first.ID == listsPage2[0].ID {
			t.Fatal("cursor pages must not repeat lists")
		}
	}
	if _, _, err := data.ListsForScope(ctx, store, userID, nil, false, "not-a-cursor", 3); !errors.Is(err, data.ErrInvalidCursor) {
		t.Fatalf("invalid list cursor: %v", err)
	}

	for index, name := range []string{"Alfa", "Beta", "Gamma", "Delta"} {
		group := data.Group{
			ID: uuid.Must(uuid.NewV7()), Name: name + " " + suffix, OwnerUserID: userID,
			CreatedBy: userID, CreatedAt: now.Add(time.Duration(index) * time.Second),
			UpdatedAt: now.Add(time.Duration(index) * time.Second),
		}
		if err := store.Within(ctx, func(q data.Querier) error {
			if err := data.CreateGroup(ctx, q, group); err != nil {
				return err
			}
			return data.AddGroupMember(ctx, q, group.ID, userID, "admin", now)
		}); err != nil {
			t.Fatalf("create group: %v", err)
		}
	}
	groupsPage1, groupsCursor := collectPage(t, func(cursor string) ([]data.Group, string, error) {
		return data.ListGroups(ctx, store, userID, cursor, 3)
	}, "")
	if len(groupsPage1) != 3 || groupsCursor == "" {
		t.Fatalf("groups first page: %d items, cursor %q", len(groupsPage1), groupsCursor)
	}
	groupsPage2, _ := collectPage(t, func(cursor string) ([]data.Group, string, error) {
		return data.ListGroups(ctx, store, userID, cursor, 3)
	}, groupsCursor)
	if len(groupsPage2) != 1 {
		t.Fatalf("groups second page: %d items", len(groupsPage2))
	}

	itemList := integrationList(t, ctx, store, userID)
	for index := 0; index < 5; index++ {
		item := data.Item{
			ID: uuid.Must(uuid.NewV7()), ListID: itemList.ID, Name: "Item", Quantity: "1",
			Unit: "unit", CreatedBy: userID, CreatedAt: now.Add(time.Duration(index) * time.Second),
			UpdatedAt: now.Add(time.Duration(index) * time.Second),
		}
		if err := data.CreateItem(ctx, store, item); err != nil {
			t.Fatalf("create item: %v", err)
		}
	}
	itemsPage1, itemsCursor := collectPage(t, func(cursor string) ([]data.Item, string, error) {
		return data.ListItems(ctx, store, itemList.ID, "all", cursor, 3)
	}, "")
	if len(itemsPage1) != 3 || itemsCursor == "" {
		t.Fatalf("items first page: %d items, cursor %q", len(itemsPage1), itemsCursor)
	}
	itemsPage2, _ := collectPage(t, func(cursor string) ([]data.Item, string, error) {
		return data.ListItems(ctx, store, itemList.ID, "all", cursor, 3)
	}, itemsCursor)
	if len(itemsPage2) != 2 {
		t.Fatalf("items second page: %d items", len(itemsPage2))
	}
	purchased, _ := collectPage(t, func(cursor string) ([]data.Item, string, error) {
		return data.ListItems(ctx, store, itemList.ID, "purchased", cursor, 3)
	}, "")
	if len(purchased) != 0 {
		t.Fatalf("expected no purchased items, got %d", len(purchased))
	}

	for index := 0; index < 4; index++ {
		if err := data.InsertActivity(ctx, store, data.Activity{
			ID: uuid.Must(uuid.NewV7()), ScopeType: "personal", ActorID: &userID,
			ListID: &itemList.ID, Action: "list.updated", Payload: map[string]any{},
			CreatedAt: now.Add(time.Duration(index) * time.Second),
		}); err != nil {
			t.Fatalf("insert activity: %v", err)
		}
	}
	activityPage1, activityCursor := collectPage(t, func(cursor string) ([]data.Activity, string, error) {
		return data.LastActivity(ctx, store, "personal", userID, nil, cursor, 3)
	}, "")
	if len(activityPage1) != 3 || activityCursor == "" {
		t.Fatalf("activity first page: %d items, cursor %q", len(activityPage1), activityCursor)
	}
	activityPage2, _ := collectPage(t, func(cursor string) ([]data.Activity, string, error) {
		return data.LastActivity(ctx, store, "personal", userID, nil, cursor, 3)
	}, activityCursor)
	if len(activityPage2) != 1 {
		t.Fatalf("activity second page: %d items", len(activityPage2))
	}
}
