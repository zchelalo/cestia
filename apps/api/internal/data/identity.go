package data

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	cestia "github.com/zchelalo/cestia/apps/api/internal/db/sqlc/cestia"
)

func CreateUser(ctx context.Context, q Querier, user *User) error {
	return q.Queries().CreateUser(ctx, cestia.CreateUserParams{
		ID:              user.ID,
		Email:           user.Email,
		DisplayName:     user.DisplayName,
		PasswordHash:    user.PasswordHash,
		EmailVerifiedAt: user.EmailVerifiedAt,
		Locale:          user.Locale,
		Timezone:        user.Timezone,
		CreatedAt:       user.CreatedAt,
	})
}

func userFromRow(row cestia.User) User {
	return User{
		ID:              row.ID,
		Email:           row.Email,
		DisplayName:     row.DisplayName,
		PasswordHash:    row.PasswordHash,
		EmailVerifiedAt: row.EmailVerifiedAt,
		Locale:          row.Locale,
		Timezone:        row.Timezone,
		DisabledAt:      row.DisabledAt,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}

func UserByEmail(ctx context.Context, q Querier, email string) (User, error) {
	row, err := q.Queries().GetUserByEmail(ctx, email)
	if err != nil {
		return User{}, err
	}
	return userFromRow(row), nil
}

func UserByID(ctx context.Context, q Querier, id uuid.UUID) (User, error) {
	row, err := q.Queries().GetUserByID(ctx, id)
	if err != nil {
		return User{}, err
	}
	return userFromRow(row), nil
}

func UserForUpdate(ctx context.Context, q Querier, id uuid.UUID) (User, error) {
	row, err := q.Queries().GetUserByIDForUpdate(ctx, id)
	if err != nil {
		return User{}, err
	}
	return userFromRow(row), nil
}

func UserForShare(ctx context.Context, q Querier, id uuid.UUID) (User, error) {
	row, err := q.Queries().GetUserByIDForShare(ctx, id)
	if err != nil {
		return User{}, err
	}
	return userFromRow(row), nil
}

func UpdateProfile(ctx context.Context, q Querier, id uuid.UUID, name, locale, timezone string, at time.Time) (User, error) {
	row, err := q.Queries().UpdateUserProfile(ctx, cestia.UpdateUserProfileParams{
		ID: id, DisplayName: name, Locale: locale, Timezone: timezone, UpdatedAt: at,
	})
	if err != nil {
		return User{}, err
	}
	return userFromRow(row), nil
}

func MarkEmailVerified(ctx context.Context, q Querier, id uuid.UUID, at time.Time) error {
	return q.Queries().MarkUserEmailVerified(ctx, cestia.MarkUserEmailVerifiedParams{ID: id, At: at})
}

func SetPassword(ctx context.Context, q Querier, id uuid.UUID, hash string, at time.Time) error {
	return q.Queries().SetUserPassword(ctx, cestia.SetUserPasswordParams{
		ID: id, PasswordHash: hash, At: at,
	})
}

func DeactivateUser(ctx context.Context, q Querier, id uuid.UUID, at time.Time) error {
	return q.Queries().DeactivateUser(ctx, cestia.DeactivateUserParams{ID: id, At: at})
}

func CountOwnedGroups(ctx context.Context, q Querier, userID uuid.UUID) (int, error) {
	count, err := q.Queries().CountOwnedGroups(ctx, userID)
	return int(count), err
}

func UserExists(ctx context.Context, q Querier, id uuid.UUID) (bool, error) {
	return q.Queries().UserExists(ctx, id)
}

func CreateSession(ctx context.Context, q Querier, session *Session) error {
	return q.Queries().CreateSession(ctx, cestia.CreateSessionParams{
		ID: session.ID, UserID: session.UserID, TokenHash: session.TokenHash,
		CsrfHash: session.CSRFHash, ExpiresAt: session.ExpiresAt, CreatedAt: session.CreatedAt,
	})
}

func sessionFromRow(row cestia.Session) Session {
	return Session{
		ID: row.ID, UserID: row.UserID, TokenHash: row.TokenHash, CSRFHash: row.CsrfHash,
		ExpiresAt: row.ExpiresAt, RevokedAt: row.RevokedAt, CreatedAt: row.CreatedAt,
	}
}

func SessionUserByToken(ctx context.Context, q Querier, tokenHash []byte, now time.Time) (Session, User, error) {
	row, err := q.Queries().GetSessionByToken(ctx, cestia.GetSessionByTokenParams{
		TokenHash: tokenHash, Now: now,
	})
	if err != nil {
		return Session{}, User{}, err
	}
	session := sessionFromRow(row)
	user, err := UserByID(ctx, q, row.UserID)
	return session, user, err
}

func RevokeSession(ctx context.Context, q Querier, userID, sessionID uuid.UUID, at time.Time) (bool, error) {
	rows, err := q.Queries().RevokeSession(ctx, cestia.RevokeSessionParams{
		ID: sessionID, UserID: userID, At: at,
	})
	return rows == 1, err
}

func RevokeAllSessions(ctx context.Context, q Querier, userID uuid.UUID, at time.Time) error {
	return q.Queries().RevokeAllSessions(ctx, cestia.RevokeAllSessionsParams{UserID: userID, At: at})
}

func ListSessions(ctx context.Context, q Querier, userID uuid.UUID, now time.Time) ([]Session, error) {
	rows, err := q.Queries().ListActiveSessions(ctx, cestia.ListActiveSessionsParams{UserID: userID, Now: now})
	if err != nil {
		return nil, err
	}
	out := make([]Session, 0, len(rows))
	for _, row := range rows {
		out = append(out, sessionFromRow(row))
	}
	return out, nil
}

type EmailAction struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	Purpose    string
	TokenHash  []byte
	Generation int
	ExpiresAt  time.Time
	ConsumedAt *time.Time
}

func CreateEmailAction(ctx context.Context, q Querier, action *EmailAction, at time.Time) error {
	return q.Queries().CreateEmailAction(ctx, cestia.CreateEmailActionParams{
		ID: action.ID, UserID: action.UserID, Purpose: action.Purpose, TokenHash: action.TokenHash,
		Generation: int32(action.Generation), ExpiresAt: action.ExpiresAt, CreatedAt: at,
	})
}

func emailActionFromRow(row cestia.GetEmailActionByTokenForUpdateRow) EmailAction {
	return EmailAction{
		ID: row.ID, UserID: row.UserID, Purpose: row.Purpose, TokenHash: row.TokenHash,
		Generation: int(row.Generation), ExpiresAt: row.ExpiresAt, ConsumedAt: row.ConsumedAt,
	}
}

func EmailActionByToken(ctx context.Context, q Querier, hash []byte, purpose string, now time.Time) (EmailAction, error) {
	row, err := q.Queries().GetEmailActionByTokenForUpdate(ctx, cestia.GetEmailActionByTokenForUpdateParams{
		TokenHash: hash, Purpose: purpose, Now: now,
	})
	if err != nil {
		return EmailAction{}, err
	}
	return emailActionFromRow(row), nil
}

// EmailActionByTokenRead locates the owner without locking the action. Callers
// lock the user first and then re-read the action with EmailActionByToken, so
// the lock order stays user -> action.
func EmailActionByTokenRead(ctx context.Context, q Querier, hash []byte, purpose string, now time.Time) (EmailAction, error) {
	row, err := q.Queries().GetEmailActionByToken(ctx, cestia.GetEmailActionByTokenParams{
		TokenHash: hash, Purpose: purpose, Now: now,
	})
	if err != nil {
		return EmailAction{}, err
	}
	return EmailAction{
		ID: row.ID, UserID: row.UserID, Purpose: row.Purpose, TokenHash: row.TokenHash,
		Generation: int(row.Generation), ExpiresAt: row.ExpiresAt, ConsumedAt: row.ConsumedAt,
	}, nil
}

func ConsumeEmailAction(ctx context.Context, q Querier, id uuid.UUID, at time.Time) error {
	return q.Queries().ConsumeEmailAction(ctx, cestia.ConsumeEmailActionParams{ID: id, At: at})
}

func InvalidateEmailActions(ctx context.Context, q Querier, userID uuid.UUID, purpose string, at time.Time) error {
	queries := q.Queries()
	if err := queries.CancelEmailDeliveriesForUserPurpose(ctx, cestia.CancelEmailDeliveriesForUserPurposeParams{
		At: at, UserID: userID, Purpose: purpose,
	}); err != nil {
		return err
	}
	return queries.ConsumeEmailActionsForUserPurpose(ctx, cestia.ConsumeEmailActionsForUserPurposeParams{
		At: at, UserID: userID, Purpose: purpose,
	})
}

func CreateEmailDeliveryForAction(ctx context.Context, q Querier, id, actionID uuid.UUID, recipient string, generation int, at time.Time) error {
	return q.Queries().CreateEmailDeliveryForAction(ctx, cestia.CreateEmailDeliveryForActionParams{
		ID: id, ActionID: actionID, Generation: int32(generation), Recipient: recipient, At: at,
	})
}

func InsertActivity(ctx context.Context, q Querier, event Activity) error {
	payload, err := json.Marshal(event.Payload)
	if err != nil {
		return err
	}
	return q.Queries().InsertActivity(ctx, cestia.InsertActivityParams{
		ID: event.ID, ScopeType: event.ScopeType, ActorID: event.ActorID, GroupID: event.GroupID,
		ListID: event.ListID, ItemID: event.ItemID, PurchaseID: event.PurchaseID,
		Action: event.Action, Payload: payload, CreatedAt: event.CreatedAt,
	})
}

func RateLimit(ctx context.Context, q Querier, purpose, subjectHash string, windowStart time.Time, limit int) (bool, int, error) {
	count, err := q.Queries().IncrementRateLimit(ctx, cestia.IncrementRateLimitParams{
		Purpose: purpose, SubjectHash: subjectHash, WindowStart: windowStart,
	})
	return int(count) <= limit, int(count), err
}

func CleanupExpired(ctx context.Context, q Querier, now time.Time) error {
	queries := q.Queries()
	if err := queries.DeleteExpiredIdempotency(ctx, now); err != nil {
		return err
	}
	return queries.DeleteOldRateLimits(ctx, now.Add(-48*time.Hour))
}

var ErrNoRows = pgx.ErrNoRows
var ErrNotFound = errors.New("not found")
