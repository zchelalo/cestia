package identity

import (
	"context"
	"encoding/hex"
	"errors"
	"net/mail"
	"strings"
	"time"
	_ "time/tzdata"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/zchelalo/cestia/apps/api/internal/apperr"
	"github.com/zchelalo/cestia/apps/api/internal/data"
	"github.com/zchelalo/cestia/apps/api/internal/secure"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	store         *data.Store
	tokenSecret   string
	sessionTTL    time.Duration
	bcryptCost    int
	registerLimit int
	now           func() time.Time
}

func New(store *data.Store, tokenSecret string, sessionTTL time.Duration, registerLimit int) *Service {
	if registerLimit < 1 {
		registerLimit = 10
	}
	return &Service{
		store: store, tokenSecret: tokenSecret, sessionTTL: sessionTTL,
		bcryptCost: bcrypt.DefaultCost, registerLimit: registerLimit, now: time.Now,
	}
}

type RegisterInput struct {
	Email       string
	Password    string
	DisplayName string
	IP          string
}

type LoginResult struct {
	User         data.User
	SessionToken string
	CSRFToken    string
	ExpiresAt    time.Time
}

func (s *Service) Register(ctx context.Context, in RegisterInput) (data.User, error) {
	email, err := normalizeEmail(in.Email)
	if err != nil {
		return data.User{}, apperr.Unprocessable("invalid_email", "Ingresa un correo válido.")
	}
	name := strings.TrimSpace(in.DisplayName)
	if name == "" || utf8.RuneCountInString(name) > 120 {
		return data.User{}, apperr.Unprocessable("invalid_name", "El nombre debe tener entre 1 y 120 caracteres.")
	}
	if len(in.Password) < 12 || len([]byte(in.Password)) > 72 {
		return data.User{}, apperr.Unprocessable("weak_password", "La contraseña debe tener al menos 12 caracteres y no más de 72 bytes.")
	}
	now := s.now().UTC()
	allowed, _, err := data.RateLimit(ctx, s.store, "register", secureSubject(in.IP), now.Truncate(15*time.Minute), s.registerLimit)
	if err != nil {
		return data.User{}, apperr.Internal(err)
	}
	if !allowed {
		return data.User{}, apperr.TooMany("Demasiados intentos. Intenta de nuevo más tarde.")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), s.bcryptCost)
	if err != nil {
		return data.User{}, apperr.Internal(err)
	}
	user := data.User{
		ID: uuid.Must(uuid.NewV7()), Email: email, DisplayName: name, PasswordHash: string(hash),
		Locale: "es", Timezone: "America/Mexico_City", CreatedAt: now, UpdatedAt: now,
	}
	err = s.store.Within(ctx, func(q data.Querier) error {
		if err := data.CreateUser(ctx, q, &user); err != nil {
			return err
		}
		if err := s.queueEmailAction(ctx, q, user, "verify_email", 24*time.Hour, now); err != nil {
			return err
		}
		return data.InsertActivity(ctx, q, data.Activity{
			ID: uuid.Must(uuid.NewV7()), ScopeType: "user", ActorID: &user.ID, Action: "account.created",
			Payload: map[string]any{"email": user.Email}, CreatedAt: now,
		})
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return data.User{}, apperr.Conflict("email_unavailable", "No fue posible crear la cuenta con ese correo.")
		}
		return data.User{}, apperr.Internal(err)
	}
	return user, nil
}

func (s *Service) Login(ctx context.Context, email, password, ip string) (LoginResult, error) {
	normalized, err := normalizeEmail(email)
	if err != nil {
		return LoginResult{}, apperr.Unauthorized("invalid_credentials", "Correo o contraseña incorrectos.")
	}
	now := s.now().UTC()
	allowed, _, err := data.RateLimit(ctx, s.store, "login", secureSubject(strings.ToLower(normalized)+"|"+ip), now.Truncate(15*time.Minute), 20)
	if err != nil {
		return LoginResult{}, apperr.Internal(err)
	}
	if !allowed {
		return LoginResult{}, apperr.TooMany("Demasiados intentos. Intenta de nuevo más tarde.")
	}
	user, err := data.UserByEmail(ctx, s.store, normalized)
	if errors.Is(err, pgx.ErrNoRows) {
		return LoginResult{}, apperr.Unauthorized("invalid_credentials", "Correo o contraseña incorrectos.")
	}
	if err != nil {
		return LoginResult{}, apperr.Internal(err)
	}
	if user.DisabledAt != nil || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return LoginResult{}, apperr.Unauthorized("invalid_credentials", "Correo o contraseña incorrectos.")
	}
	sessionToken, err := secure.RandomToken(32)
	if err != nil {
		return LoginResult{}, apperr.Internal(err)
	}
	csrfToken, err := secure.RandomToken(24)
	if err != nil {
		return LoginResult{}, apperr.Internal(err)
	}
	expires := now.Add(s.sessionTTL)
	session := data.Session{
		ID: uuid.Must(uuid.NewV7()), UserID: user.ID, TokenHash: secure.Hash(sessionToken),
		CSRFHash: secure.Hash(csrfToken), ExpiresAt: expires, CreatedAt: now,
	}
	err = s.store.Within(ctx, func(q data.Querier) error {
		// The password was checked outside the transaction. Re-read it under
		// the user lock so a concurrent password change cannot let a session
		// authenticated with the old password survive.
		locked, err := data.UserForUpdate(ctx, q, user.ID)
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.Unauthorized("invalid_credentials", "Correo o contraseña incorrectos.")
		}
		if err != nil {
			return err
		}
		if locked.DisabledAt != nil || locked.PasswordHash != user.PasswordHash {
			return apperr.Unauthorized("invalid_credentials", "Correo o contraseña incorrectos.")
		}
		return data.CreateSession(ctx, q, &session)
	})
	if err != nil {
		return LoginResult{}, apperr.From(err)
	}
	return LoginResult{User: user, SessionToken: sessionToken, CSRFToken: csrfToken, ExpiresAt: expires}, nil
}

func (s *Service) Authenticate(ctx context.Context, token string) (data.User, data.Session, error) {
	if token == "" {
		return data.User{}, data.Session{}, apperr.Unauthorized("missing_session", "Tu sesión no es válida.")
	}
	session, user, err := data.SessionUserByToken(ctx, s.store, secure.Hash(token), s.now().UTC())
	if errors.Is(err, pgx.ErrNoRows) {
		return data.User{}, data.Session{}, apperr.Unauthorized("invalid_session", "Tu sesión expiró.")
	}
	if err != nil {
		return data.User{}, data.Session{}, apperr.Internal(err)
	}
	if user.DisabledAt != nil {
		return data.User{}, data.Session{}, apperr.Unauthorized("invalid_session", "Tu sesión no es válida.")
	}
	return user, session, nil
}

func (s *Service) Logout(ctx context.Context, userID, sessionID uuid.UUID) error {
	_, err := data.RevokeSession(ctx, s.store, userID, sessionID, s.now().UTC())
	if err != nil {
		return apperr.Internal(err)
	}
	return nil
}

func (s *Service) VerifyEmail(ctx context.Context, token string) error {
	return s.store.Within(ctx, func(q data.Querier) error {
		hash := secure.Hash(token)
		now := s.now().UTC()
		initial, err := data.EmailActionByTokenRead(ctx, q, hash, "verify_email", now)
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.Gone("invalid_email_action", "El enlace no es válido o ya fue utilizado.")
		}
		if err != nil {
			return apperr.Internal(err)
		}
		if _, err := data.UserForUpdate(ctx, q, initial.UserID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return apperr.Gone("invalid_email_action", "El enlace no es válido o ya fue utilizado.")
			}
			return apperr.Internal(err)
		}
		now = s.now().UTC()
		action, err := data.EmailActionByToken(ctx, q, hash, "verify_email", now)
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.Gone("invalid_email_action", "El enlace no es válido o ya fue utilizado.")
		}
		if err != nil {
			return apperr.Internal(err)
		}
		if action.ID != initial.ID || action.UserID != initial.UserID {
			return apperr.Gone("invalid_email_action", "El enlace no es válido o ya fue utilizado.")
		}
		if err := data.ConsumeEmailAction(ctx, q, action.ID, now); err != nil {
			return apperr.Internal(err)
		}
		if err := data.MarkEmailVerified(ctx, q, action.UserID, now); err != nil {
			return apperr.Internal(err)
		}
		return data.InsertActivity(ctx, q, data.Activity{
			ID: uuid.Must(uuid.NewV7()), ScopeType: "user", ActorID: &action.UserID,
			Action: "account.email_verified", Payload: map[string]any{}, CreatedAt: now,
		})
	})
}

func (s *Service) ResendVerification(ctx context.Context, userID uuid.UUID, email string) error {
	now := s.now().UTC()
	allowed, _, err := data.RateLimit(ctx, s.store, "resend_verification", hex.EncodeToString(secure.Hash(userID.String())), now.Truncate(time.Hour), 5)
	if err != nil {
		return apperr.Internal(err)
	}
	if !allowed {
		return apperr.TooMany("Espera antes de solicitar otro correo.")
	}
	return s.store.Within(ctx, func(q data.Querier) error {
		user, err := data.UserByID(ctx, q, userID)
		if err != nil || user.Verified() {
			return nil
		}
		return s.queueEmailAction(ctx, q, user, "verify_email", 24*time.Hour, now)
	})
}

func (s *Service) ForgotPassword(ctx context.Context, email, ip string) error {
	normalized, err := normalizeEmail(email)
	if err != nil {
		return nil
	}
	now := s.now().UTC()
	allowed, _, err := data.RateLimit(ctx, s.store, "forgot_password", secureSubject(ip), now.Truncate(15*time.Minute), 5)
	if err != nil {
		return apperr.Internal(err)
	}
	if !allowed {
		return apperr.TooMany("Demasiadas solicitudes. Intenta de nuevo más tarde.")
	}
	return s.store.Within(ctx, func(q data.Querier) error {
		user, err := data.UserByEmail(ctx, q, normalized)
		if errors.Is(err, pgx.ErrNoRows) || user.DisabledAt != nil {
			return nil
		}
		if err != nil {
			return apperr.Internal(err)
		}
		return s.queueEmailAction(ctx, q, user, "reset_password", time.Hour, now)
	})
}

func (s *Service) ResetPassword(ctx context.Context, token, password string) error {
	if len(password) < 12 || len([]byte(password)) > 72 {
		return apperr.Unprocessable("weak_password", "La contraseña debe tener al menos 12 caracteres y no más de 72 bytes.")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.bcryptCost)
	if err != nil {
		return apperr.Internal(err)
	}
	return s.store.Within(ctx, func(q data.Querier) error {
		hashValue := secure.Hash(token)
		now := s.now().UTC()
		initial, err := data.EmailActionByTokenRead(ctx, q, hashValue, "reset_password", now)
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.Gone("invalid_reset_token", "El enlace no es válido o expiró.")
		}
		if err != nil {
			return apperr.Internal(err)
		}
		locked, err := data.UserForUpdate(ctx, q, initial.UserID)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && locked.DisabledAt != nil) {
			return apperr.Gone("invalid_reset_token", "El enlace no es válido o expiró.")
		}
		if err != nil {
			return apperr.Internal(err)
		}
		now = s.now().UTC()
		action, err := data.EmailActionByToken(ctx, q, hashValue, "reset_password", now)
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.Gone("invalid_reset_token", "El enlace no es válido o expiró.")
		}
		if err != nil {
			return apperr.Internal(err)
		}
		if action.ID != initial.ID || action.UserID != initial.UserID {
			return apperr.Gone("invalid_reset_token", "El enlace no es válido o expiró.")
		}
		if err := data.ConsumeEmailAction(ctx, q, action.ID, now); err != nil {
			return apperr.Internal(err)
		}
		if err := data.SetPassword(ctx, q, action.UserID, string(hash), now); err != nil {
			return apperr.Internal(err)
		}
		if err := data.RevokeAllSessions(ctx, q, action.UserID, now); err != nil {
			return apperr.Internal(err)
		}
		if err := data.InvalidateEmailActions(ctx, q, action.UserID, "reset_password", now); err != nil {
			return apperr.Internal(err)
		}
		return data.InsertActivity(ctx, q, data.Activity{
			ID: uuid.Must(uuid.NewV7()), ScopeType: "user", ActorID: &action.UserID,
			Action: "account.password_reset", Payload: map[string]any{}, CreatedAt: now,
		})
	})
}

func (s *Service) ChangePassword(ctx context.Context, userID uuid.UUID, current, next string) error {
	if len(next) < 12 || len([]byte(next)) > 72 {
		return apperr.Unprocessable("weak_password", "La contraseña debe tener al menos 12 caracteres y no más de 72 bytes.")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(next), s.bcryptCost)
	if err != nil {
		return apperr.Internal(err)
	}
	return s.store.Within(ctx, func(q data.Querier) error {
		locked, err := data.UserForUpdate(ctx, q, userID)
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.Unauthorized("invalid_session", "Tu sesión no es válida.")
		}
		if err != nil {
			return apperr.Internal(err)
		}
		if locked.DisabledAt != nil {
			return apperr.Unauthorized("invalid_session", "Tu sesión no es válida.")
		}
		if bcrypt.CompareHashAndPassword([]byte(locked.PasswordHash), []byte(current)) != nil {
			return apperr.Unprocessable("invalid_current_password", "La contraseña actual no es correcta.")
		}
		now := s.now().UTC()
		if err := data.SetPassword(ctx, q, userID, string(hash), now); err != nil {
			return apperr.Internal(err)
		}
		if err := data.RevokeAllSessions(ctx, q, userID, now); err != nil {
			return apperr.Internal(err)
		}
		if err := data.InvalidateEmailActions(ctx, q, userID, "reset_password", now); err != nil {
			return apperr.Internal(err)
		}
		return data.InsertActivity(ctx, q, data.Activity{
			ID: uuid.Must(uuid.NewV7()), ScopeType: "user", ActorID: &userID,
			Action: "account.password_changed", Payload: map[string]any{}, CreatedAt: now,
		})
	})
}

func (s *Service) UpdateProfile(ctx context.Context, userID uuid.UUID, name, locale, timezone string) (data.User, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > 120 {
		return data.User{}, apperr.Unprocessable("invalid_name", "El nombre debe tener entre 1 y 120 caracteres.")
	}
	if locale != "es" {
		locale = "es"
	}
	if timezone == "" {
		timezone = "America/Mexico_City"
	}
	if _, err := time.LoadLocation(timezone); err != nil {
		return data.User{}, apperr.Unprocessable("invalid_timezone", "La zona horaria no es válida.")
	}
	user, err := data.UpdateProfile(ctx, s.store, userID, name, locale, timezone, s.now().UTC())
	if err != nil {
		return data.User{}, apperr.Internal(err)
	}
	return user, nil
}

func (s *Service) Deactivate(ctx context.Context, userID uuid.UUID, password string) error {
	return s.store.Within(ctx, func(q data.Querier) error {
		locked, err := data.UserForUpdate(ctx, q, userID)
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.Unauthorized("invalid_session", "Tu sesión no es válida.")
		}
		if err != nil {
			return apperr.Internal(err)
		}
		if locked.DisabledAt != nil {
			return apperr.Conflict("already_disabled", "La cuenta ya está desactivada.")
		}
		if bcrypt.CompareHashAndPassword([]byte(locked.PasswordHash), []byte(password)) != nil {
			return apperr.Unprocessable("invalid_current_password", "La contraseña actual no es correcta.")
		}
		owned, err := data.CountOwnedGroups(ctx, q, userID)
		if err != nil {
			return apperr.Internal(err)
		}
		if owned > 0 {
			return apperr.Conflict("owner_must_transfer", "Transfiere tus grupos antes de desactivar la cuenta.")
		}
		now := s.now().UTC()
		if err := data.DeactivateUser(ctx, q, userID, now); err != nil {
			return apperr.Internal(err)
		}
		if err := data.InvalidateEmailActions(ctx, q, userID, "reset_password", now); err != nil {
			return apperr.Internal(err)
		}
		return data.RevokeAllSessions(ctx, q, userID, now)
	})
}

func (s *Service) Sessions(ctx context.Context, userID uuid.UUID) ([]data.Session, error) {
	sessions, err := data.ListSessions(ctx, s.store, userID, s.now().UTC())
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return sessions, nil
}

func (s *Service) RevokeSession(ctx context.Context, userID, sessionID uuid.UUID) error {
	ok, err := data.RevokeSession(ctx, s.store, userID, sessionID, s.now().UTC())
	if err != nil {
		return apperr.Internal(err)
	}
	if !ok {
		return apperr.NotFound("La sesión no existe.")
	}
	return nil
}

func (s *Service) queueEmailAction(ctx context.Context, q data.Querier, user data.User, purpose string, ttl time.Duration, now time.Time) error {
	locked, err := data.UserForUpdate(ctx, q, user.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return apperr.Internal(err)
	}
	if locked.DisabledAt != nil {
		return nil
	}
	if purpose == "verify_email" && locked.Verified() {
		return nil
	}
	if err := data.InvalidateEmailActions(ctx, q, locked.ID, purpose, now); err != nil {
		return apperr.Internal(err)
	}
	actionID := uuid.Must(uuid.NewV7())
	token := secure.Derive(s.tokenSecret, purpose, actionID.String(), 1)
	action := data.EmailAction{
		ID: actionID, UserID: locked.ID, Purpose: purpose, TokenHash: secure.Hash(token),
		Generation: 1, ExpiresAt: now.Add(ttl),
	}
	if err := data.CreateEmailAction(ctx, q, &action, now); err != nil {
		return apperr.Internal(err)
	}
	if err := data.CreateEmailDeliveryForAction(ctx, q, uuid.Must(uuid.NewV7()), actionID, locked.Email, 1, now); err != nil {
		return apperr.Internal(err)
	}
	return nil
}

func normalizeEmail(raw string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	address, err := mail.ParseAddress(value)
	if err != nil || address.Address != value || !strings.Contains(value, "@") {
		return "", errors.New("invalid email")
	}
	return value, nil
}

func secureSubject(value string) string {
	return hex.EncodeToString(secure.Hash(value))
}
