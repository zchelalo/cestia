package identity

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zchelalo/cestia/apps/api/internal/apperr"
	"github.com/zchelalo/cestia/apps/api/internal/data"
	"github.com/zchelalo/cestia/apps/api/internal/migrations"
	"golang.org/x/crypto/bcrypt"
)

func integrationStore(t *testing.T) *data.Store {
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
	return data.NewFromPool(pool)
}

func TestIntegrationPasswordChangeRejectsOldCredentials(t *testing.T) {
	ctx := context.Background()
	store := integrationStore(t)
	now := time.Now().UTC()
	oldPassword := "old-password-123"
	hash, err := bcrypt.GenerateFromPassword([]byte(oldPassword), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	user := data.User{
		ID: uuid.Must(uuid.NewV7()), Email: "password-" + uuid.Must(uuid.NewV7()).String() + "@example.com",
		DisplayName: "Password", PasswordHash: string(hash), Locale: "es",
		Timezone: "America/Mexico_City", CreatedAt: now, UpdatedAt: now,
	}
	if err := data.CreateUser(ctx, store, &user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	service := New(store, "integration-secret", time.Hour, 100)
	if _, err := service.Login(ctx, user.Email, oldPassword, "127.0.0.1"); err != nil {
		t.Fatalf("login with old password: %v", err)
	}
	newPassword := "new-password-456"
	if err := service.ChangePassword(ctx, user.ID, oldPassword, newPassword); err != nil {
		t.Fatalf("change password: %v", err)
	}
	if _, err := service.Login(ctx, user.Email, oldPassword, "127.0.0.1"); err == nil {
		t.Fatal("login with the replaced password must fail")
	}
	sessions, err := service.Sessions(ctx, user.ID)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 0 {
		t.Fatalf("password change must revoke every session, got %d", len(sessions))
	}
	if _, err := service.Login(ctx, user.Email, newPassword, "127.0.0.1"); err != nil {
		t.Fatalf("login with new password: %v", err)
	}
	if err := service.ChangePassword(ctx, user.ID, oldPassword, "another-password-789"); err == nil {
		t.Fatal("stale current password must be rejected")
	} else {
		var appErr *apperr.Error
		if !errors.As(err, &appErr) || appErr.Code != "invalid_current_password" {
			t.Fatalf("unexpected error: %v", err)
		}
	}
}
