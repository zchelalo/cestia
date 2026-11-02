package shopping

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zchelalo/cestia/apps/api/internal/apperr"
	"github.com/zchelalo/cestia/apps/api/internal/data"
)

func TestIntegrationPersonalItemsAreAssignedToOwner(t *testing.T) {
	fixture := newIdempotencyFixture(t, 0)
	service := New(fixture.store)

	item, err := service.CreateItem(fixture.ctx, fixture.user.ID, fixture.list.ID, ItemInput{
		Name: "Leche", Quantity: "1", Unit: "unit",
	})
	if err != nil {
		t.Fatalf("create personal item: %v", err)
	}
	if item.AssignedTo == nil || *item.AssignedTo != fixture.user.ID {
		t.Fatalf("personal item should be assigned to its owner, got %v", item.AssignedTo)
	}

	other := data.User{
		ID: uuid.Must(uuid.NewV7()), Email: "other-" + uuid.Must(uuid.NewV7()).String() + "@example.com",
		DisplayName: "Other", PasswordHash: "$2a$10$integration", Locale: "es",
		Timezone: "America/Mexico_City", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	if err := data.CreateUser(fixture.ctx, fixture.store, &other); err != nil {
		t.Fatalf("create other user: %v", err)
	}
	if _, err := service.CreateItem(fixture.ctx, fixture.user.ID, fixture.list.ID, ItemInput{
		Name: "Pan", Quantity: "1", Unit: "unit", AssignedTo: &other.ID,
	}); err == nil {
		t.Fatal("assigning a personal item to another user must fail")
	} else {
		var appErr *apperr.Error
		if !errors.As(err, &appErr) || appErr.Code != "invalid_assignee" {
			t.Fatalf("expected invalid_assignee, got %v", err)
		}
	}

	updated, err := service.UpdateItem(fixture.ctx, fixture.user.ID, item.ID, ItemInput{
		Name: "Leche", Quantity: "2", Unit: "unit", Version: item.Version,
	})
	if err != nil {
		t.Fatalf("update personal item: %v", err)
	}
	if updated.AssignedTo == nil || *updated.AssignedTo != fixture.user.ID {
		t.Fatalf("pending personal item should keep its owner, got %v", updated.AssignedTo)
	}
}
