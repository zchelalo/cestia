package shopping

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"github.com/zchelalo/cestia/apps/api/internal/apperr"
	"github.com/zchelalo/cestia/apps/api/internal/data"
	"github.com/zchelalo/cestia/apps/api/internal/domain"
)

type Service struct {
	store *data.Store
	now   func() time.Time
}

func New(store *data.Store) *Service {
	return &Service{store: store, now: time.Now}
}

type ListInput struct {
	Name        string
	Description string
	GroupID     *uuid.UUID
}

type ItemInput struct {
	Name               string
	Description        string
	Quantity           string
	Unit               string
	EstimatedUnitPrice string
	AssignedTo         *uuid.UUID
	Notes              string
	Version            int64
}

func (s *Service) Lists(ctx context.Context, userID uuid.UUID, groupID *uuid.UUID, includeArchived bool) ([]data.List, error) {
	if groupID != nil {
		if _, err := s.groupAccess(ctx, userID, *groupID); err != nil {
			return nil, err
		}
	}
	lists, err := data.ListsForScope(ctx, s.store, userID, groupID, includeArchived)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return lists, nil
}

func (s *Service) GetList(ctx context.Context, userID, listID uuid.UUID) (data.List, error) {
	list, err := data.ListForUser(ctx, s.store, listID, userID, false)
	if errors.Is(err, pgx.ErrNoRows) {
		return data.List{}, apperr.NotFound("La lista no existe.")
	}
	if err != nil {
		return data.List{}, apperr.Internal(err)
	}
	return list, nil
}

func (s *Service) CreateList(ctx context.Context, userID uuid.UUID, in ListInput) (data.List, error) {
	name, description, err := validateList(in.Name, in.Description)
	if err != nil {
		return data.List{}, err
	}
	now := s.now().UTC()
	list := data.List{
		ID: uuid.Must(uuid.NewV7()), Name: name, Description: description, GroupID: in.GroupID,
		CreatedBy: userID, CreatedAt: now, UpdatedAt: now,
	}
	if in.GroupID == nil {
		list.PersonalOwnerID = &userID
	}
	err = s.store.Within(ctx, func(q data.Querier) error {
		if in.GroupID != nil {
			group, err := data.GroupForUserLocked(ctx, q, *in.GroupID, userID)
			if errors.Is(err, pgx.ErrNoRows) {
				return apperr.NotFound("El grupo no existe.")
			}
			if err != nil {
				return err
			}
			if group.Role != "owner" && group.Role != "admin" {
				return apperr.Forbidden("Solo propietarios y administradores pueden crear listas.")
			}
			if group.ArchivedAt != nil {
				return apperr.Conflict("group_archived", "El grupo está archivado.")
			}
		}
		if err := data.CreateList(ctx, q, list); err != nil {
			return err
		}
		return data.InsertActivity(ctx, q, data.Activity{
			ID: uuid.Must(uuid.NewV7()), ScopeType: scopeType(list), ActorID: &userID,
			GroupID: list.GroupID, ListID: &list.ID, Action: "list.created",
			Payload: map[string]any{"name": name}, CreatedAt: now,
		})
	})
	if err != nil {
		return data.List{}, apperr.From(err)
	}
	return s.GetList(ctx, userID, list.ID)
}

func (s *Service) UpdateList(ctx context.Context, userID, listID uuid.UUID, name, description string, version int64) (data.List, error) {
	name, description, err := validateList(name, description)
	if err != nil {
		return data.List{}, err
	}
	now := s.now().UTC()
	err = s.store.Within(ctx, func(q data.Querier) error {
		list, err := s.lockedList(ctx, q, userID, listID)
		if err != nil {
			return err
		}
		if err := ensureListWritable(list); err != nil {
			return err
		}
		if list.Role != "owner" && list.Role != "admin" {
			return apperr.Forbidden("No puedes editar esta lista.")
		}
		if list.GroupID != nil {
			group, err := data.GroupForUserSharedLocked(ctx, q, *list.GroupID, userID)
			if errors.Is(err, pgx.ErrNoRows) {
				return apperr.NotFound("La lista no existe.")
			}
			if err != nil {
				return apperr.Internal(err)
			}
			if group.ArchivedAt != nil {
				return apperr.Conflict("group_archived", "El grupo está archivado.")
			}
		}
		ok, err := data.UpdateList(ctx, q, listID, name, description, version, now)
		if err != nil {
			return err
		}
		if !ok {
			return apperr.Conflict("version_conflict", "La lista cambió. Actualiza e intenta de nuevo.")
		}
		return data.InsertActivity(ctx, q, data.Activity{
			ID: uuid.Must(uuid.NewV7()), ScopeType: scopeType(list), ActorID: &userID,
			GroupID: list.GroupID, ListID: &list.ID, Action: "list.updated",
			Payload: map[string]any{"name": name}, CreatedAt: now,
		})
	})
	if err != nil {
		return data.List{}, apperr.From(err)
	}
	return s.GetList(ctx, userID, listID)
}

func (s *Service) ArchiveList(ctx context.Context, userID, listID uuid.UUID, archived bool, version int64) (data.List, error) {
	now := s.now().UTC()
	err := s.store.Within(ctx, func(q data.Querier) error {
		list, err := s.lockedList(ctx, q, userID, listID)
		if err != nil {
			return err
		}
		if list.Role != "owner" && list.Role != "admin" {
			return apperr.Forbidden("No puedes archivar esta lista.")
		}
		if list.GroupID != nil {
			group, err := data.GroupForUserSharedLocked(ctx, q, *list.GroupID, userID)
			if errors.Is(err, pgx.ErrNoRows) {
				return apperr.NotFound("La lista no existe.")
			}
			if err != nil {
				return apperr.Internal(err)
			}
			if group.ArchivedAt != nil && !archived {
				return apperr.Conflict("group_archived", "Restaura el grupo primero.")
			}
		}
		ok, err := data.SetListArchived(ctx, q, listID, archived, version, now)
		if err != nil {
			return err
		}
		if !ok {
			return apperr.Conflict("version_conflict", "La lista cambió. Actualiza e intenta de nuevo.")
		}
		action := "list.restored"
		if archived {
			action = "list.archived"
		}
		return data.InsertActivity(ctx, q, data.Activity{
			ID: uuid.Must(uuid.NewV7()), ScopeType: scopeType(list), ActorID: &userID,
			GroupID: list.GroupID, ListID: &list.ID, Action: action,
			Payload: map[string]any{}, CreatedAt: now,
		})
	})
	if err != nil {
		return data.List{}, apperr.From(err)
	}
	return s.GetList(ctx, userID, listID)
}

func (s *Service) Items(ctx context.Context, userID, listID uuid.UUID) ([]data.Item, error) {
	if _, err := s.GetList(ctx, userID, listID); err != nil {
		return nil, err
	}
	items, err := data.ListItems(ctx, s.store, listID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	for i := range items {
		purchase, err := data.ActivePurchaseForItem(ctx, s.store, items[i].ID, false)
		if err == nil {
			items[i].Purchase = &purchase
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.Internal(err)
		}
	}
	return items, nil
}

func (s *Service) CreateItem(ctx context.Context, userID, listID uuid.UUID, in ItemInput) (data.Item, error) {
	name, quantity, estimated, err := validateItem(in)
	if err != nil {
		return data.Item{}, err
	}
	now := s.now().UTC()
	item := data.Item{
		ID: uuid.Must(uuid.NewV7()), ListID: listID, Name: name, Description: strings.TrimSpace(in.Description),
		Quantity: quantity.StringFixed(3), Unit: in.Unit, EstimatedUnitPrice: estimated,
		CreatedBy: userID, AssignedTo: in.AssignedTo, Notes: strings.TrimSpace(in.Notes),
		CreatedAt: now, UpdatedAt: now,
	}
	err = s.store.Within(ctx, func(q data.Querier) error {
		list, err := s.writableListForMutation(ctx, q, userID, listID)
		if err != nil {
			return err
		}
		if err := s.validateAssignee(ctx, q, list, userID, in.AssignedTo); err != nil {
			return err
		}
		if err := data.CreateItem(ctx, q, item); err != nil {
			return err
		}
		if err := data.TouchList(ctx, q, listID, now); err != nil {
			return err
		}
		return data.InsertActivity(ctx, q, data.Activity{
			ID: uuid.Must(uuid.NewV7()), ScopeType: scopeType(list), ActorID: &userID,
			GroupID: list.GroupID, ListID: &listID, ItemID: &item.ID, Action: "item.created",
			Payload: map[string]any{"name": item.Name, "quantity": item.Quantity, "unit": item.Unit}, CreatedAt: now,
		})
	})
	if err != nil {
		return data.Item{}, apperr.From(err)
	}
	item.Version = 1
	item.Purchase = nil
	return item, nil
}

func (s *Service) UpdateItem(ctx context.Context, userID, itemID uuid.UUID, in ItemInput) (data.Item, error) {
	name, quantity, estimated, err := validateItem(in)
	if err != nil {
		return data.Item{}, err
	}
	now := s.now().UTC()
	var result data.Item
	err = s.store.Within(ctx, func(q data.Querier) error {
		initial, _, err := data.ItemForUser(ctx, q, itemID, userID, false)
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.NotFound("El artículo no existe.")
		}
		if err != nil {
			return err
		}
		list, err := s.writableListForMutation(ctx, q, userID, initial.ListID)
		if err != nil {
			return err
		}
		item, err := data.ItemForUpdate(ctx, q, itemID)
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.NotFound("El artículo no existe.")
		}
		if err != nil {
			return err
		}
		if item.Version != in.Version {
			return apperr.Conflict("version_conflict", "El artículo cambió. Actualiza e intenta de nuevo.")
		}
		purchase, err := data.ActivePurchaseForItem(ctx, q, itemID, false)
		if err == nil {
			if item.Name != name || item.Description != strings.TrimSpace(in.Description) ||
				item.Quantity != quantity.StringFixed(3) || item.Unit != in.Unit ||
				!samePrice(item.EstimatedUnitPrice, estimated) || !sameUUID(item.AssignedTo, in.AssignedTo) {
				return apperr.Conflict("item_purchased", "Solo puedes editar las notas de un artículo comprado.")
			}
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if err := s.validateAssignee(ctx, q, list, userID, in.AssignedTo); err != nil {
			return err
		}
		item.Name, item.Description, item.Quantity, item.Unit = name, strings.TrimSpace(in.Description), quantity.StringFixed(3), in.Unit
		item.EstimatedUnitPrice, item.AssignedTo, item.Notes, item.UpdatedAt, item.Version = estimated, in.AssignedTo, strings.TrimSpace(in.Notes), now, in.Version
		ok, err := data.UpdateItem(ctx, q, item)
		if err != nil {
			return err
		}
		if !ok {
			return apperr.Conflict("version_conflict", "El artículo cambió. Actualiza e intenta de nuevo.")
		}
		result = item
		result.Version = in.Version + 1
		if purchase.ID != uuid.Nil {
			result.Purchase = &purchase
		}
		if err := data.TouchList(ctx, q, list.ID, now); err != nil {
			return err
		}
		return data.InsertActivity(ctx, q, data.Activity{
			ID: uuid.Must(uuid.NewV7()), ScopeType: scopeType(list), ActorID: &userID,
			GroupID: list.GroupID, ListID: &list.ID, ItemID: &item.ID, Action: "item.updated",
			Payload: map[string]any{"name": item.Name}, CreatedAt: now,
		})
	})
	if err != nil {
		return data.Item{}, apperr.From(err)
	}
	return result, nil
}

func (s *Service) DeleteItem(ctx context.Context, userID, itemID uuid.UUID, version int64) error {
	now := s.now().UTC()
	return s.store.Within(ctx, func(q data.Querier) error {
		initial, _, err := data.ItemForUser(ctx, q, itemID, userID, false)
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.NotFound("El artículo no existe.")
		}
		if err != nil {
			return err
		}
		list, err := s.writableListForMutation(ctx, q, userID, initial.ListID)
		if err != nil {
			return err
		}
		if list.Role != "owner" && list.Role != "admin" {
			return apperr.Forbidden("Solo propietarios y administradores pueden eliminar artículos.")
		}
		ok, err := data.DeleteItem(ctx, q, itemID, version, now)
		if err != nil {
			return err
		}
		if !ok {
			return apperr.Conflict("version_conflict", "El artículo cambió. Actualiza e intenta de nuevo.")
		}
		if err := data.TouchList(ctx, q, list.ID, now); err != nil {
			return err
		}
		return data.InsertActivity(ctx, q, data.Activity{
			ID: uuid.Must(uuid.NewV7()), ScopeType: scopeType(list), ActorID: &userID,
			GroupID: list.GroupID, ListID: &list.ID, ItemID: &itemID, Action: "item.deleted",
			Payload: map[string]any{"name": initial.Name}, CreatedAt: now,
		})
	})
}

func (s *Service) Activity(ctx context.Context, userID uuid.UUID, groupID *uuid.UUID, limit int) ([]data.Activity, error) {
	scope := "personal"
	if groupID != nil {
		scope = "group"
		if _, err := s.groupAccess(ctx, userID, *groupID); err != nil {
			return nil, err
		}
	}
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	events, err := data.LastActivity(ctx, s.store, scope, userID, groupID, limit)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return events, nil
}

func (s *Service) groupAccess(ctx context.Context, userID, groupID uuid.UUID) (data.Group, error) {
	return s.groupAccessWith(ctx, s.store, userID, groupID)
}

func (s *Service) groupAccessWith(ctx context.Context, q data.Querier, userID, groupID uuid.UUID) (data.Group, error) {
	group, err := data.GroupForUser(ctx, q, groupID, userID, false)
	if errors.Is(err, pgx.ErrNoRows) {
		return data.Group{}, apperr.NotFound("El grupo no existe.")
	}
	if err != nil {
		return data.Group{}, apperr.Internal(err)
	}
	return group, nil
}

func (s *Service) lockedList(ctx context.Context, q data.Querier, userID, listID uuid.UUID) (data.List, error) {
	initial, err := data.ListForUser(ctx, q, listID, userID, false)
	if errors.Is(err, pgx.ErrNoRows) {
		return data.List{}, apperr.NotFound("La lista no existe.")
	}
	if err != nil {
		return data.List{}, apperr.Internal(err)
	}
	if initial.GroupID != nil {
		if _, err := data.GroupForUserSharedLocked(ctx, q, *initial.GroupID, userID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return data.List{}, apperr.NotFound("La lista no existe.")
			}
			return data.List{}, apperr.From(err)
		}
	}
	list, err := data.ListForUser(ctx, q, listID, userID, true)
	if errors.Is(err, pgx.ErrNoRows) {
		return data.List{}, apperr.NotFound("La lista no existe.")
	}
	if err != nil {
		return data.List{}, apperr.Internal(err)
	}
	return list, nil
}

func (s *Service) writableList(ctx context.Context, q data.Querier, userID, listID uuid.UUID) (data.List, error) {
	return s.writableListWithLock(ctx, q, userID, listID, false)
}

// writableListForMutation takes the exclusive list lock required by operations
// that touch the list row (create, update or delete an item). Keeping group ->
// list -> item lock order avoids upgrading a shared list lock mid-transaction.
func (s *Service) writableListForMutation(ctx context.Context, q data.Querier, userID, listID uuid.UUID) (data.List, error) {
	return s.writableListWithLock(ctx, q, userID, listID, true)
}

func (s *Service) writableListWithLock(ctx context.Context, q data.Querier, userID, listID uuid.UUID, exclusive bool) (data.List, error) {
	initial, err := data.ListForUser(ctx, q, listID, userID, false)
	if errors.Is(err, pgx.ErrNoRows) {
		return data.List{}, apperr.NotFound("La lista no existe.")
	}
	if err != nil {
		return data.List{}, apperr.Internal(err)
	}
	if initial.GroupID != nil {
		group, err := data.GroupForUserSharedLocked(ctx, q, *initial.GroupID, userID)
		if errors.Is(err, pgx.ErrNoRows) {
			return data.List{}, apperr.NotFound("El grupo no existe.")
		}
		if err != nil {
			return data.List{}, apperr.Internal(err)
		}
		if group.ArchivedAt != nil {
			return data.List{}, apperr.Conflict("group_archived", "El grupo está archivado.")
		}
	}
	var list data.List
	if exclusive {
		list, err = data.ListForUser(ctx, q, listID, userID, true)
	} else {
		list, err = data.ListForUserShare(ctx, q, listID, userID)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return data.List{}, apperr.NotFound("La lista no existe.")
	}
	if err != nil {
		return data.List{}, apperr.Internal(err)
	}
	if err := ensureListWritable(list); err != nil {
		return data.List{}, err
	}
	return list, nil
}

func (s *Service) validateAssignee(ctx context.Context, q data.Querier, list data.List, actorID uuid.UUID, assignee *uuid.UUID) error {
	if assignee == nil {
		return nil
	}
	if list.GroupID == nil {
		if *assignee != actorID {
			return apperr.Unprocessable("invalid_assignee", "Solo puedes asignarte artículos a ti mismo.")
		}
		return nil
	}
	role, err := data.GroupMemberRole(ctx, q, *list.GroupID, *assignee)
	if errors.Is(err, pgx.ErrNoRows) || role == "" {
		return apperr.Unprocessable("invalid_assignee", "El usuario asignado debe ser un integrante activo.")
	}
	if err != nil {
		return err
	}
	return nil
}

func ensureListWritable(list data.List) error {
	if list.ArchivedAt != nil {
		return apperr.Conflict("list_archived", "La lista está archivada.")
	}
	return nil
}

func scopeType(list data.List) string {
	if list.GroupID != nil {
		return "group"
	}
	return "personal"
}

func validateList(name, description string) (string, string, error) {
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

func validateItem(in ItemInput) (string, decimal.Decimal, *string, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" || utf8.RuneCountInString(name) > 120 {
		return "", decimal.Zero, nil, apperr.Unprocessable("invalid_name", "El nombre debe tener entre 1 y 120 caracteres.")
	}
	if utf8.RuneCountInString(in.Description) > 2000 || utf8.RuneCountInString(in.Notes) > 4000 {
		return "", decimal.Zero, nil, apperr.Unprocessable("invalid_text", "La descripción o las notas son demasiado largas.")
	}
	switch in.Unit {
	case "unit", "kg", "g", "l", "ml", "package":
	default:
		return "", decimal.Zero, nil, apperr.Unprocessable("invalid_unit", "Selecciona una unidad válida.")
	}
	quantity, err := domain.ParseQuantity(in.Quantity)
	if err != nil {
		return "", decimal.Zero, nil, apperr.Unprocessable("invalid_quantity", "La cantidad debe ser mayor que cero y dentro del límite.")
	}
	price, err := domain.ParsePrice(in.EstimatedUnitPrice)
	if err != nil {
		return "", decimal.Zero, nil, apperr.Unprocessable("invalid_price", "El precio estimado no es válido.")
	}
	var estimated *string
	if price != nil {
		value := price.StringFixed(4)
		estimated = &value
	}
	return name, quantity, estimated, nil
}

func sameUUID(a, b *uuid.UUID) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func samePrice(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	left, errLeft := domain.ParsePrice(*a)
	right, errRight := domain.ParsePrice(*b)
	return errLeft == nil && errRight == nil && left != nil && right != nil && left.Equal(*right)
}
