package groups

import (
	"context"
	"encoding/hex"
	"errors"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/zchelalo/cestia/apps/api/internal/apperr"
	"github.com/zchelalo/cestia/apps/api/internal/data"
	"github.com/zchelalo/cestia/apps/api/internal/secure"
)

type Service struct {
	store       *data.Store
	tokenSecret string
	now         func() time.Time
}

func New(store *data.Store, tokenSecret string) *Service {
	return &Service{store: store, tokenSecret: tokenSecret, now: time.Now}
}

func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]data.Group, error) {
	items, err := data.ListGroups(ctx, s.store, userID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return items, nil
}

func (s *Service) Get(ctx context.Context, userID, groupID uuid.UUID) (data.Group, error) {
	group, err := data.GroupForUser(ctx, s.store, groupID, userID, false)
	if errors.Is(err, pgx.ErrNoRows) {
		return data.Group{}, apperr.NotFound("El grupo no existe.")
	}
	if err != nil {
		return data.Group{}, apperr.Internal(err)
	}
	return group, nil
}

func (s *Service) Create(ctx context.Context, userID uuid.UUID, name, description string) (data.Group, error) {
	name, description, err := validateGroup(name, description)
	if err != nil {
		return data.Group{}, err
	}
	now := s.now().UTC()
	group := data.Group{
		ID: uuid.Must(uuid.NewV7()), Name: name, Description: description, OwnerUserID: userID,
		CreatedBy: userID, CreatedAt: now, UpdatedAt: now,
	}
	err = s.store.Within(ctx, func(q data.Querier) error {
		actor, err := data.UserForShare(ctx, q, userID)
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.Unauthorized("invalid_session", "Tu sesión no es válida.")
		}
		if err != nil {
			return err
		}
		if actor.DisabledAt != nil {
			return apperr.Conflict("owner_unavailable", "La cuenta no puede ser propietaria de un grupo.")
		}
		if err := data.CreateGroup(ctx, q, group); err != nil {
			return err
		}
		if err := data.AddGroupMember(ctx, q, group.ID, userID, "admin", now); err != nil {
			return err
		}
		return data.InsertActivity(ctx, q, data.Activity{
			ID: uuid.Must(uuid.NewV7()), ScopeType: "group", ActorID: &userID, GroupID: &group.ID,
			Action: "group.created", Payload: map[string]any{"name": name}, CreatedAt: now,
		})
	})
	if err != nil {
		return data.Group{}, apperr.From(err)
	}
	return s.Get(ctx, userID, group.ID)
}

func (s *Service) Update(ctx context.Context, userID, groupID uuid.UUID, name, description string, version int64) (data.Group, error) {
	name, description, err := validateGroup(name, description)
	if err != nil {
		return data.Group{}, err
	}
	now := s.now().UTC()
	err = s.store.Within(ctx, func(q data.Querier) error {
		group, err := data.GroupForUserLocked(ctx, q, groupID, userID)
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.NotFound("El grupo no existe.")
		}
		if err != nil {
			return err
		}
		if group.Role != "owner" {
			return apperr.Forbidden("Solo el propietario puede editar el grupo.")
		}
		if group.ArchivedAt != nil {
			return apperr.Conflict("group_archived", "Restaura el grupo antes de editarlo.")
		}
		ok, err := data.UpdateGroup(ctx, q, groupID, name, description, version, now)
		if err != nil {
			return err
		}
		if !ok {
			return apperr.Conflict("version_conflict", "El grupo cambió. Actualiza e intenta de nuevo.")
		}
		return data.InsertActivity(ctx, q, data.Activity{
			ID: uuid.Must(uuid.NewV7()), ScopeType: "group", ActorID: &userID, GroupID: &groupID,
			Action: "group.updated", Payload: map[string]any{"name": name}, CreatedAt: now,
		})
	})
	if err != nil {
		return data.Group{}, apperr.From(err)
	}
	return s.Get(ctx, userID, groupID)
}

func (s *Service) Archive(ctx context.Context, userID, groupID uuid.UUID, archived bool, version int64) (data.Group, error) {
	now := s.now().UTC()
	err := s.store.Within(ctx, func(q data.Querier) error {
		group, err := data.GroupForUserLocked(ctx, q, groupID, userID)
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.NotFound("El grupo no existe.")
		}
		if err != nil {
			return err
		}
		if group.Role != "owner" {
			return apperr.Forbidden("Solo el propietario puede archivar el grupo.")
		}
		ok, err := data.SetGroupArchived(ctx, q, groupID, archived, version, now)
		if err != nil {
			return err
		}
		if !ok {
			return apperr.Conflict("version_conflict", "El grupo cambió. Actualiza e intenta de nuevo.")
		}
		action := "group.restored"
		if archived {
			action = "group.archived"
		}
		return data.InsertActivity(ctx, q, data.Activity{
			ID: uuid.Must(uuid.NewV7()), ScopeType: "group", ActorID: &userID, GroupID: &groupID,
			Action: action, Payload: map[string]any{}, CreatedAt: now,
		})
	})
	if err != nil {
		return data.Group{}, apperr.From(err)
	}
	return s.Get(ctx, userID, groupID)
}

func (s *Service) Members(ctx context.Context, userID, groupID uuid.UUID) ([]data.GroupMember, error) {
	if _, err := s.Get(ctx, userID, groupID); err != nil {
		return nil, err
	}
	members, err := data.ListGroupMembers(ctx, s.store, groupID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return members, nil
}

func (s *Service) SetRole(ctx context.Context, actorID, groupID, targetID uuid.UUID, role string) error {
	if role != "admin" && role != "member" {
		return apperr.Unprocessable("invalid_role", "El rol debe ser admin o member.")
	}
	now := s.now().UTC()
	return s.store.Within(ctx, func(q data.Querier) error {
		group, err := data.GroupForUserLocked(ctx, q, groupID, actorID)
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.NotFound("El grupo no existe.")
		}
		if err != nil {
			return err
		}
		if group.Role != "owner" {
			return apperr.Forbidden("Solo el propietario puede administrar roles.")
		}
		if group.ArchivedAt != nil {
			return apperr.Conflict("group_archived", "El grupo está archivado.")
		}
		if targetID == group.OwnerUserID {
			return apperr.Conflict("owner_role_fixed", "La propiedad no se cambia desde los roles.")
		}
		targetRole, err := data.GroupMemberRole(ctx, q, groupID, targetID)
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.NotFound("El integrante no existe.")
		}
		if err != nil {
			return err
		}
		if targetRole == "owner" {
			return apperr.Conflict("owner_role_fixed", "La propiedad no se cambia desde los roles.")
		}
		if err := data.SetMemberRole(ctx, q, groupID, targetID, role); err != nil {
			return err
		}
		return data.InsertActivity(ctx, q, data.Activity{
			ID: uuid.Must(uuid.NewV7()), ScopeType: "group", ActorID: &actorID, GroupID: &groupID,
			Action: "member.role_changed", Payload: map[string]any{"user_id": targetID, "role": role}, CreatedAt: now,
		})
	})
}

func (s *Service) RemoveMember(ctx context.Context, actorID, groupID, targetID uuid.UUID) error {
	now := s.now().UTC()
	return s.store.Within(ctx, func(q data.Querier) error {
		group, err := data.GroupForUserLocked(ctx, q, groupID, actorID)
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.NotFound("El grupo no existe.")
		}
		if err != nil {
			return err
		}
		if group.Role != "owner" && group.Role != "admin" {
			return apperr.Forbidden("No puedes eliminar integrantes.")
		}
		if group.ArchivedAt != nil {
			return apperr.Conflict("group_archived", "El grupo está archivado.")
		}
		if targetID == actorID {
			return apperr.Conflict("use_leave", "Usa la opción Abandonar grupo.")
		}
		targetRole, err := data.GroupMemberRole(ctx, q, groupID, targetID)
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.NotFound("El integrante no existe.")
		}
		if err != nil {
			return err
		}
		if targetRole == "owner" || (group.Role == "admin" && targetRole == "admin") {
			return apperr.Forbidden("No puedes eliminar a este integrante.")
		}
		if err := data.RemoveGroupMember(ctx, q, groupID, targetID, now); err != nil {
			return err
		}
		return data.InsertActivity(ctx, q, data.Activity{
			ID: uuid.Must(uuid.NewV7()), ScopeType: "group", ActorID: &actorID, GroupID: &groupID,
			Action: "member.removed", Payload: map[string]any{"user_id": targetID}, CreatedAt: now,
		})
	})
}

func (s *Service) Leave(ctx context.Context, actorID, groupID uuid.UUID) error {
	now := s.now().UTC()
	return s.store.Within(ctx, func(q data.Querier) error {
		group, err := data.GroupForUserLocked(ctx, q, groupID, actorID)
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.NotFound("El grupo no existe.")
		}
		if err != nil {
			return err
		}
		if group.Role == "owner" {
			return apperr.Conflict("owner_must_transfer", "Transfiere la propiedad antes de abandonar el grupo.")
		}
		if err := data.RemoveGroupMember(ctx, q, groupID, actorID, now); err != nil {
			return err
		}
		return data.InsertActivity(ctx, q, data.Activity{
			ID: uuid.Must(uuid.NewV7()), ScopeType: "group", ActorID: &actorID, GroupID: &groupID,
			Action: "member.left", Payload: map[string]any{"user_id": actorID}, CreatedAt: now,
		})
	})
}

func (s *Service) TransferOwner(ctx context.Context, actorID, groupID, targetID uuid.UUID) (data.Group, error) {
	now := s.now().UTC()
	err := s.store.Within(ctx, func(q data.Querier) error {
		target, err := data.UserForShare(ctx, q, targetID)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && target.DisabledAt != nil) {
			return apperr.Conflict("owner_unavailable", "El nuevo propietario debe ser una cuenta activa.")
		}
		if err != nil {
			return err
		}
		group, err := data.GroupForUserLocked(ctx, q, groupID, actorID)
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.NotFound("El grupo no existe.")
		}
		if err != nil {
			return err
		}
		if group.Role != "owner" {
			return apperr.Forbidden("Solo el propietario puede transferir el grupo.")
		}
		targetRole, err := data.GroupMemberRole(ctx, q, groupID, targetID)
		if errors.Is(err, pgx.ErrNoRows) || targetRole == "owner" {
			return apperr.NotFound("El nuevo propietario debe ser un integrante activo.")
		}
		if err != nil {
			return err
		}
		if err := data.TransferOwner(ctx, q, groupID, actorID, targetID, now); err != nil {
			return err
		}
		return data.InsertActivity(ctx, q, data.Activity{
			ID: uuid.Must(uuid.NewV7()), ScopeType: "group", ActorID: &actorID, GroupID: &groupID,
			Action: "group.owner_transferred", Payload: map[string]any{"new_owner_id": targetID}, CreatedAt: now,
		})
	})
	if err != nil {
		return data.Group{}, apperr.From(err)
	}
	return s.Get(ctx, actorID, groupID)
}

func (s *Service) Invitations(ctx context.Context, actorID, groupID uuid.UUID) ([]data.Invitation, error) {
	group, err := s.Get(ctx, actorID, groupID)
	if err != nil {
		return nil, err
	}
	if group.Role != "owner" && group.Role != "admin" {
		return nil, apperr.Forbidden("No puedes consultar las invitaciones.")
	}
	items, err := data.ListInvitations(ctx, s.store, groupID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return items, nil
}

func (s *Service) Invite(ctx context.Context, actorID, groupID uuid.UUID, email string) (data.Invitation, error) {
	email, err := normalizeEmail(email)
	if err != nil {
		return data.Invitation{}, apperr.Unprocessable("invalid_email", "Ingresa un correo válido.")
	}
	now := s.now().UTC()
	allowed, _, err := data.RateLimit(ctx, s.store, "invite_actor", hex.EncodeToString(secure.Hash(actorID.String())), now.Truncate(time.Hour), 20)
	if err != nil {
		return data.Invitation{}, apperr.Internal(err)
	}
	if !allowed {
		return data.Invitation{}, apperr.TooMany("Alcanzaste el límite de invitaciones por hora.")
	}
	allowed, _, err = data.RateLimit(ctx, s.store, "invite_group", hex.EncodeToString(secure.Hash(groupID.String())), now.Truncate(24*time.Hour), 100)
	if err != nil {
		return data.Invitation{}, apperr.Internal(err)
	}
	if !allowed {
		return data.Invitation{}, apperr.TooMany("El grupo alcanzó el límite diario de invitaciones.")
	}

	invitation := data.Invitation{
		ID: uuid.Must(uuid.NewV7()), GroupID: groupID, Email: email, InvitedBy: actorID,
		Status: "pending", Generation: 1, Version: 1, ExpiresAt: now.Add(7 * 24 * time.Hour), CreatedAt: now, UpdatedAt: now,
	}
	err = s.store.Within(ctx, func(q data.Querier) error {
		group, err := data.GroupForUserLocked(ctx, q, groupID, actorID)
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.NotFound("El grupo no existe.")
		}
		if err != nil {
			return err
		}
		if group.Role != "owner" && group.Role != "admin" {
			return apperr.Forbidden("No puedes invitar integrantes.")
		}
		if group.ArchivedAt != nil {
			return apperr.Conflict("group_archived", "Restaura el grupo antes de invitar.")
		}
		existing, err := data.ActiveGroupMemberEmail(ctx, q, groupID, email)
		if err != nil {
			return err
		}
		if existing {
			return apperr.Conflict("already_member", "Ese correo ya pertenece al grupo.")
		}
		if err := data.ExpirePendingInvitation(ctx, q, groupID, email, now); err != nil {
			return err
		}
		token := secure.Derive(s.tokenSecret, "invitation", invitation.ID.String(), 1)
		invitation.TokenHash = secure.Hash(token)
		invitation.RawToken = token
		if err := data.CreateInvitation(ctx, q, &invitation); err != nil {
			return err
		}
		if err := data.CreateInvitationDelivery(ctx, q, uuid.Must(uuid.NewV7()), invitation.ID, email, 1, now); err != nil {
			return err
		}
		return data.InsertActivity(ctx, q, data.Activity{
			ID: uuid.Must(uuid.NewV7()), ScopeType: "group", ActorID: &actorID, GroupID: &groupID,
			Action: "invitation.created", Payload: map[string]any{"email": email, "invitation_id": invitation.ID}, CreatedAt: now,
		})
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return data.Invitation{}, apperr.Conflict("invitation_exists", "Ya existe una invitación pendiente para ese correo.")
		}
		return data.Invitation{}, apperr.From(err)
	}
	return invitation, nil
}

func (s *Service) ResendInvitation(ctx context.Context, actorID, groupID, invitationID uuid.UUID, version int64) (data.Invitation, error) {
	now := s.now().UTC()
	allowed, _, err := data.RateLimit(ctx, s.store, "invitation_resend", hex.EncodeToString(secure.Hash(actorID.String())), now.Truncate(time.Hour), 30)
	if err != nil {
		return data.Invitation{}, apperr.Internal(err)
	}
	if !allowed {
		return data.Invitation{}, apperr.TooMany("Alcanzaste el límite de reenvíos por hora.")
	}
	var result data.Invitation
	err = s.store.Within(ctx, func(q data.Querier) error {
		group, err := data.GroupForUserSharedLocked(ctx, q, groupID, actorID)
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.NotFound("El grupo no existe.")
		}
		if err != nil {
			return err
		}
		if group.Role != "owner" && group.Role != "admin" {
			return apperr.Forbidden("No puedes reenviar invitaciones.")
		}
		if group.ArchivedAt != nil {
			return apperr.Conflict("group_archived", "El grupo está archivado.")
		}
		inv, err := data.InvitationByID(ctx, q, invitationID, true)
		if errors.Is(err, pgx.ErrNoRows) || inv.GroupID != groupID {
			return apperr.NotFound("La invitación no existe.")
		}
		if err != nil {
			return err
		}
		if inv.Status != "pending" && inv.Status != "expired" {
			return apperr.Conflict("invitation_resolved", "La invitación ya fue resuelta.")
		}
		if now.Sub(inv.UpdatedAt) < 60*time.Second {
			return apperr.TooMany("Espera un minuto antes de reenviar.")
		}
		generation := inv.Generation + 1
		token := secure.Derive(s.tokenSecret, "invitation", inv.ID.String(), generation)
		expires := now.Add(7 * 24 * time.Hour)
		if err := data.CancelInvitationDeliveries(ctx, q, inv.ID, now); err != nil {
			return err
		}
		ok, err := data.UpdateInvitationToken(ctx, q, inv.ID, secure.Hash(token), generation, expires, version, now)
		if err != nil {
			return err
		}
		if !ok {
			return apperr.Conflict("version_conflict", "La invitación cambió. Actualiza e intenta de nuevo.")
		}
		if err := data.CreateInvitationDelivery(ctx, q, uuid.Must(uuid.NewV7()), inv.ID, inv.Email, generation, now); err != nil {
			return err
		}
		result = inv
		result.Status = "pending"
		result.ResolvedAt = nil
		result.ResolvedBy = nil
		result.Generation = generation
		result.ExpiresAt = expires
		result.RawToken = token
		result.Version = version + 1
		result.UpdatedAt = now
		return data.InsertActivity(ctx, q, data.Activity{
			ID: uuid.Must(uuid.NewV7()), ScopeType: "group", ActorID: &actorID, GroupID: &groupID,
			Action: "invitation.resent", Payload: map[string]any{"invitation_id": inv.ID}, CreatedAt: now,
		})
	})
	if err != nil {
		return data.Invitation{}, apperr.From(err)
	}
	return result, nil
}

func (s *Service) CancelInvitation(ctx context.Context, actorID, groupID, invitationID uuid.UUID) error {
	now := s.now().UTC()
	return s.store.Within(ctx, func(q data.Querier) error {
		group, err := data.GroupForUserSharedLocked(ctx, q, groupID, actorID)
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.NotFound("El grupo no existe.")
		}
		if err != nil {
			return err
		}
		if group.Role != "owner" && group.Role != "admin" {
			return apperr.Forbidden("No puedes cancelar invitaciones.")
		}
		if group.ArchivedAt != nil {
			return apperr.Conflict("group_archived", "El grupo está archivado.")
		}
		inv, err := data.InvitationByID(ctx, q, invitationID, true)
		if errors.Is(err, pgx.ErrNoRows) || inv.GroupID != groupID {
			return apperr.NotFound("La invitación no existe.")
		}
		if err != nil {
			return err
		}
		if inv.Status != "pending" {
			return apperr.Conflict("invitation_resolved", "La invitación ya fue resuelta.")
		}
		if err := data.CancelInvitationDeliveries(ctx, q, inv.ID, now); err != nil {
			return err
		}
		if err := data.CancelInvitation(ctx, q, inv.ID, now); err != nil {
			return err
		}
		return data.InsertActivity(ctx, q, data.Activity{
			ID: uuid.Must(uuid.NewV7()), ScopeType: "group", ActorID: &actorID, GroupID: &groupID,
			Action: "invitation.canceled", Payload: map[string]any{"invitation_id": inv.ID}, CreatedAt: now,
		})
	})
}

func (s *Service) PreviewInvitation(ctx context.Context, user data.User, token string) (data.Invitation, error) {
	now := s.now().UTC()
	inv, err := data.InvitationByToken(ctx, s.store, secure.Hash(token), false)
	if errors.Is(err, pgx.ErrNoRows) {
		return data.Invitation{}, apperr.Gone("invitation_invalid", "La invitación no es válida.")
	}
	if err != nil {
		return data.Invitation{}, apperr.Internal(err)
	}
	if inv.Status != "pending" {
		return data.Invitation{}, apperr.Conflict("invitation_resolved", "La invitación ya fue resuelta.")
	}
	if !inv.ExpiresAt.After(now) {
		return data.Invitation{}, apperr.Gone("invitation_expired", "La invitación expiró.")
	}
	if !strings.EqualFold(inv.Email, user.Email) {
		return data.Invitation{}, apperr.Forbidden("Esta invitación no corresponde a tu cuenta.")
	}
	inv.RawToken = ""
	return inv, nil
}

func (s *Service) ResolveInvitation(ctx context.Context, user data.User, token string, accept bool) (data.Invitation, error) {
	now := s.now().UTC()
	var result data.Invitation
	err := s.store.Within(ctx, func(q data.Querier) error {
		initial, err := data.InvitationByToken(ctx, q, secure.Hash(token), false)
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.Gone("invitation_invalid", "La invitación no es válida.")
		}
		if err != nil {
			return err
		}
		group, err := data.GroupByIDForUpdate(ctx, q, initial.GroupID)
		if err != nil {
			return err
		}
		inv, err := data.InvitationByToken(ctx, q, secure.Hash(token), true)
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.Gone("invitation_invalid", "La invitación no es válida.")
		}
		if err != nil {
			return err
		}
		if inv.Status != "pending" {
			return apperr.Conflict("invitation_resolved", "La invitación ya fue resuelta.")
		}
		if !inv.ExpiresAt.After(now) {
			return apperr.Gone("invitation_expired", "La invitación expiró.")
		}
		if !strings.EqualFold(inv.Email, user.Email) {
			return apperr.Forbidden("Esta invitación no corresponde a tu cuenta.")
		}
		if group.ArchivedAt != nil {
			return apperr.Conflict("group_archived", "El grupo está archivado.")
		}
		if !accept {
			if err := data.ResolveInvitation(ctx, q, inv.ID, "rejected", user.ID, now); err != nil {
				return err
			}
			result = inv
			result.Status = "rejected"
			result.ResolvedAt = &now
			result.ResolvedBy = &user.ID
			result.Version = inv.Version + 1
			return nil
		}
		if err := data.AddGroupMember(ctx, q, inv.GroupID, user.ID, "member", now); err != nil {
			return err
		}
		if err := data.ResolveInvitation(ctx, q, inv.ID, "accepted", user.ID, now); err != nil {
			return err
		}
		result = inv
		result.Status = "accepted"
		result.ResolvedAt = &now
		result.ResolvedBy = &user.ID
		result.Version = inv.Version + 1
		return data.InsertActivity(ctx, q, data.Activity{
			ID: uuid.Must(uuid.NewV7()), ScopeType: "group", ActorID: &user.ID, GroupID: &inv.GroupID,
			Action: "invitation.accepted", Payload: map[string]any{"user_id": user.ID}, CreatedAt: now,
		})
	})
	if err != nil {
		return data.Invitation{}, apperr.From(err)
	}
	return result, nil
}

func validateGroup(name, description string) (string, string, error) {
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)
	if name == "" || utf8.RuneCountInString(name) > 120 {
		return "", "", apperr.Unprocessable("invalid_name", "El nombre debe tener entre 1 y 120 caracteres.")
	}
	if utf8.RuneCountInString(description) > 2000 {
		return "", "", apperr.Unprocessable("invalid_description", "La descripción es demasiado larga.")
	}
	return name, description, nil
}

func normalizeEmail(raw string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	address, err := mail.ParseAddress(value)
	if err != nil || address.Address != value || !strings.Contains(value, "@") {
		return "", errors.New("invalid email")
	}
	return value, nil
}
