package shopping

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/zchelalo/cestia/apps/api/internal/apperr"
	"github.com/zchelalo/cestia/apps/api/internal/data"
	"github.com/zchelalo/cestia/apps/api/internal/domain"
)

type PurchaseInput struct {
	ExpectedVersion int64
	ActualUnitPrice string
	PurchasedBy     uuid.UUID
	PaidBy          *uuid.UUID
	PurchasedAt     time.Time
	Notes           string
	IdempotencyKey  string
}

type CorrectionInput struct {
	ExpectedVersion int64
	ActualUnitPrice string
	PurchasedBy     uuid.UUID
	PaidBy          *uuid.UUID
	PurchasedAt     time.Time
	Notes           string
	Reason          string
	IdempotencyKey  string
}

type ReverseInput struct {
	ExpectedVersion int64
	Reason          string
	IdempotencyKey  string
}

func (s *Service) RecordPurchase(ctx context.Context, userID, itemID uuid.UUID, in PurchaseInput) (data.Purchase, error) {
	if in.PurchasedAt.IsZero() || in.PurchasedAt.After(s.now().UTC().Add(5*time.Minute)) {
		return data.Purchase{}, apperr.Unprocessable("invalid_purchase_date", "La fecha de compra no puede estar en el futuro.")
	}
	if utf8.RuneCountInString(in.Notes) > 4000 {
		return data.Purchase{}, apperr.Unprocessable("invalid_notes", "Las notas son demasiado largas.")
	}
	price, err := domain.ParsePrice(in.ActualUnitPrice)
	if err != nil {
		return data.Purchase{}, apperr.Unprocessable("invalid_price", "El precio real no es válido.")
	}
	if in.PurchasedBy == uuid.Nil {
		return data.Purchase{}, apperr.Unprocessable("purchaser_required", "Selecciona quién realizó la compra.")
	}
	now := s.now().UTC()
	return executeIdempotent(ctx, s.store, userID, itemID, "purchase.record", in.IdempotencyKey, in,
		func(q data.Querier) error {
			item, _, err := data.ItemForUser(ctx, q, itemID, userID, false)
			if errors.Is(err, pgx.ErrNoRows) {
				return apperr.NotFound("El artículo no existe.")
			}
			if err != nil {
				return err
			}
			_, err = s.writableList(ctx, q, userID, item.ListID)
			return err
		},
		func(q data.Querier) (data.Purchase, error) {
			initial, _, err := data.ItemForUser(ctx, q, itemID, userID, false)
			if errors.Is(err, pgx.ErrNoRows) {
				return data.Purchase{}, apperr.NotFound("El artículo no existe.")
			}
			if err != nil {
				return data.Purchase{}, err
			}
			list, err := s.writableList(ctx, q, userID, initial.ListID)
			if err != nil {
				return data.Purchase{}, err
			}
			item, err := data.ItemForUpdate(ctx, q, itemID)
			if errors.Is(err, pgx.ErrNoRows) {
				return data.Purchase{}, apperr.NotFound("El artículo no existe.")
			}
			if err != nil {
				return data.Purchase{}, err
			}
			if item.Version != in.ExpectedVersion {
				return data.Purchase{}, apperr.Conflict("version_conflict", "El artículo cambió. Actualiza e intenta de nuevo.")
			}
			_, err = data.ActivePurchaseForItem(ctx, q, item.ID, false)
			if err == nil {
				return data.Purchase{}, apperr.Conflict("already_purchased", "El artículo ya tiene una compra activa.")
			}
			if !errors.Is(err, pgx.ErrNoRows) {
				return data.Purchase{}, err
			}
			if err := s.validPurchasePrincipals(ctx, q, list, in.PurchasedBy, in.PaidBy, uuid.Nil, uuid.Nil); err != nil {
				return data.Purchase{}, err
			}
			var priceText *string
			if price != nil {
				value := price.StringFixed(4)
				priceText = &value
			}
			purchase := data.Purchase{
				ID: uuid.Must(uuid.NewV7()), ItemID: item.ID, ItemName: item.Name, Quantity: item.Quantity,
				Unit: item.Unit, ActualUnitPrice: priceText, RecordedBy: userID, PurchasedBy: in.PurchasedBy,
				PaidBy: in.PaidBy, PurchasedAt: in.PurchasedAt.UTC(), Notes: strings.TrimSpace(in.Notes),
				CreatedAt: now, UpdatedAt: now,
			}
			if err := data.CreatePurchase(ctx, q, purchase); err != nil {
				return data.Purchase{}, err
			}
			if err := data.TouchItem(ctx, q, item.ID, now); err != nil {
				return data.Purchase{}, err
			}
			if err := data.InsertActivity(ctx, q, data.Activity{
				ID: uuid.Must(uuid.NewV7()), ScopeType: scopeType(list), ActorID: &userID,
				GroupID: list.GroupID, ListID: &list.ID, ItemID: &item.ID, PurchaseID: &purchase.ID,
				Action: "purchase.recorded",
				Payload: map[string]any{
					"item_name": item.Name, "quantity": item.Quantity, "unit": item.Unit,
					"actual_unit_price": priceText, "purchased_by": in.PurchasedBy, "paid_by": in.PaidBy,
				}, CreatedAt: now,
			}); err != nil {
				return data.Purchase{}, err
			}
			return data.ActivePurchaseForItem(ctx, q, item.ID, false)
		},
		func(purchase data.Purchase) uuid.UUID { return purchase.ItemID })
}

func (s *Service) CorrectPurchase(ctx context.Context, userID, purchaseID uuid.UUID, in CorrectionInput) (data.Purchase, error) {
	if in.PurchasedBy == uuid.Nil {
		return data.Purchase{}, apperr.Unprocessable("purchaser_required", "Selecciona quién realizó la compra.")
	}
	if strings.TrimSpace(in.Reason) == "" || utf8.RuneCountInString(in.Reason) > 500 {
		return data.Purchase{}, apperr.Unprocessable("reason_required", "Indica un motivo de entre 1 y 500 caracteres.")
	}
	if in.PurchasedAt.IsZero() || in.PurchasedAt.After(s.now().UTC().Add(5*time.Minute)) {
		return data.Purchase{}, apperr.Unprocessable("invalid_purchase_date", "La fecha de compra no puede estar en el futuro.")
	}
	if utf8.RuneCountInString(in.Notes) > 4000 {
		return data.Purchase{}, apperr.Unprocessable("invalid_notes", "Las notas son demasiado largas.")
	}
	price, err := domain.ParsePrice(in.ActualUnitPrice)
	if err != nil {
		return data.Purchase{}, apperr.Unprocessable("invalid_price", "El precio real no es válido.")
	}
	now := s.now().UTC()
	return executeIdempotent(ctx, s.store, userID, purchaseID, "purchase.correct", in.IdempotencyKey, in,
		func(q data.Querier) error {
			purchase, err := data.PurchaseForUser(ctx, q, purchaseID, userID, false)
			if errors.Is(err, pgx.ErrNoRows) {
				return apperr.NotFound("La compra no existe.")
			}
			if err != nil {
				return err
			}
			_, err = s.writableList(ctx, q, userID, purchase.ListID)
			return err
		},
		func(q data.Querier) (data.Purchase, error) {
			initial, err := data.PurchaseForUser(ctx, q, purchaseID, userID, false)
			if errors.Is(err, pgx.ErrNoRows) {
				return data.Purchase{}, apperr.NotFound("La compra no existe.")
			}
			if err != nil {
				return data.Purchase{}, err
			}
			list, err := s.writableList(ctx, q, userID, initial.ListID)
			if err != nil {
				return data.Purchase{}, err
			}
			if _, err := data.ItemForUpdateAny(ctx, q, initial.ItemID); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return data.Purchase{}, apperr.NotFound("El artículo no existe.")
				}
				return data.Purchase{}, err
			}
			purchase, err := data.PurchaseForUser(ctx, q, purchaseID, userID, true)
			if err != nil {
				return data.Purchase{}, err
			}
			if purchase.ReversedAt != nil {
				return data.Purchase{}, apperr.Conflict("purchase_reversed", "Una compra revertida no se corrige.")
			}
			if list.Role != "owner" && list.Role != "admin" && purchase.RecordedBy != userID {
				return data.Purchase{}, apperr.Forbidden("Solo puedes corregir las compras que registraste.")
			}
			if purchase.Version != in.ExpectedVersion {
				return data.Purchase{}, apperr.Conflict("version_conflict", "La compra cambió. Actualiza e intenta de nuevo.")
			}
			if err := s.validPurchasePrincipals(ctx, q, list, in.PurchasedBy, in.PaidBy, purchase.PurchasedBy, derefUUID(purchase.PaidBy)); err != nil {
				return data.Purchase{}, err
			}
			previous := map[string]any{
				"actual_unit_price": purchase.ActualUnitPrice, "purchased_by": purchase.PurchasedBy,
				"paid_by": purchase.PaidBy, "purchased_at": purchase.PurchasedAt, "notes": purchase.Notes,
			}
			var priceText *string
			if price != nil {
				value := price.StringFixed(4)
				priceText = &value
			}
			purchase.ActualUnitPrice = priceText
			purchase.PurchasedBy = in.PurchasedBy
			purchase.PaidBy = in.PaidBy
			purchase.PurchasedAt = in.PurchasedAt.UTC()
			purchase.Notes = strings.TrimSpace(in.Notes)
			purchase.UpdatedAt = now
			purchase.Version = in.ExpectedVersion
			ok, err := data.UpdatePurchase(ctx, q, purchase)
			if err != nil {
				return data.Purchase{}, err
			}
			if !ok {
				return data.Purchase{}, apperr.Conflict("version_conflict", "La compra cambió. Actualiza e intenta de nuevo.")
			}
			if err := data.TouchItem(ctx, q, purchase.ItemID, now); err != nil {
				return data.Purchase{}, err
			}
			if err := data.InsertActivity(ctx, q, data.Activity{
				ID: uuid.Must(uuid.NewV7()), ScopeType: scopeType(list), ActorID: &userID,
				GroupID: list.GroupID, ListID: &list.ID, ItemID: &purchase.ItemID, PurchaseID: &purchase.ID,
				Action: "purchase.corrected",
				Payload: map[string]any{
					"reason": strings.TrimSpace(in.Reason), "before": previous,
					"after": map[string]any{
						"actual_unit_price": priceText, "purchased_by": in.PurchasedBy,
						"paid_by": in.PaidBy, "purchased_at": in.PurchasedAt.UTC(), "notes": purchase.Notes,
					},
				}, CreatedAt: now,
			}); err != nil {
				return data.Purchase{}, err
			}
			return data.PurchaseForUser(ctx, q, purchase.ID, userID, false)
		},
		func(purchase data.Purchase) uuid.UUID { return purchase.ID })
}

func (s *Service) ReversePurchase(ctx context.Context, userID, purchaseID uuid.UUID, in ReverseInput) (data.Purchase, error) {
	if strings.TrimSpace(in.Reason) == "" || utf8.RuneCountInString(in.Reason) > 500 {
		return data.Purchase{}, apperr.Unprocessable("reason_required", "Indica un motivo de entre 1 y 500 caracteres.")
	}
	now := s.now().UTC()
	return executeIdempotent(ctx, s.store, userID, purchaseID, "purchase.reverse", in.IdempotencyKey, in,
		func(q data.Querier) error {
			purchase, err := data.PurchaseForUser(ctx, q, purchaseID, userID, false)
			if errors.Is(err, pgx.ErrNoRows) {
				return apperr.NotFound("La compra no existe.")
			}
			if err != nil {
				return err
			}
			_, err = s.writableList(ctx, q, userID, purchase.ListID)
			return err
		},
		func(q data.Querier) (data.Purchase, error) {
			initial, err := data.PurchaseForUser(ctx, q, purchaseID, userID, false)
			if errors.Is(err, pgx.ErrNoRows) {
				return data.Purchase{}, apperr.NotFound("La compra no existe.")
			}
			if err != nil {
				return data.Purchase{}, err
			}
			list, err := s.writableList(ctx, q, userID, initial.ListID)
			if err != nil {
				return data.Purchase{}, err
			}
			if _, err := data.ItemForUpdateAny(ctx, q, initial.ItemID); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return data.Purchase{}, apperr.NotFound("El artículo no existe.")
				}
				return data.Purchase{}, err
			}
			purchase, err := data.PurchaseForUser(ctx, q, purchaseID, userID, true)
			if err != nil {
				return data.Purchase{}, err
			}
			if purchase.ReversedAt != nil {
				return data.Purchase{}, apperr.Conflict("purchase_reversed", "La compra ya fue revertida.")
			}
			if list.Role != "owner" && list.Role != "admin" && purchase.RecordedBy != userID {
				return data.Purchase{}, apperr.Forbidden("Solo puedes revertir las compras que registraste.")
			}
			if purchase.Version != in.ExpectedVersion {
				return data.Purchase{}, apperr.Conflict("version_conflict", "La compra cambió. Actualiza e intenta de nuevo.")
			}
			ok, err := data.ReversePurchase(ctx, q, purchase.ID, userID, strings.TrimSpace(in.Reason), in.ExpectedVersion, now)
			if err != nil {
				return data.Purchase{}, err
			}
			if !ok {
				return data.Purchase{}, apperr.Conflict("version_conflict", "La compra cambió. Actualiza e intenta de nuevo.")
			}
			if err := data.TouchItem(ctx, q, purchase.ItemID, now); err != nil {
				return data.Purchase{}, err
			}
			if err := data.InsertActivity(ctx, q, data.Activity{
				ID: uuid.Must(uuid.NewV7()), ScopeType: scopeType(list), ActorID: &userID,
				GroupID: list.GroupID, ListID: &list.ID, ItemID: &purchase.ItemID, PurchaseID: &purchase.ID,
				Action:  "purchase.reversed",
				Payload: map[string]any{"reason": strings.TrimSpace(in.Reason)}, CreatedAt: now,
			}); err != nil {
				return data.Purchase{}, err
			}
			purchase.ReversedAt = &now
			purchase.ReversedBy = &userID
			purchase.Version = in.ExpectedVersion + 1
			reason := strings.TrimSpace(in.Reason)
			purchase.ReversalReason = &reason
			return purchase, nil
		},
		func(purchase data.Purchase) uuid.UUID { return purchase.ID })
}

func (s *Service) Purchase(ctx context.Context, userID, purchaseID uuid.UUID) (data.Purchase, error) {
	purchase, err := data.PurchaseForUser(ctx, s.store, purchaseID, userID, false)
	if errors.Is(err, pgx.ErrNoRows) {
		return data.Purchase{}, apperr.NotFound("La compra no existe.")
	}
	if err != nil {
		return data.Purchase{}, apperr.Internal(err)
	}
	return purchase, nil
}

func (s *Service) validPurchasePrincipals(
	ctx context.Context,
	q data.Querier,
	list data.List,
	purchasedBy uuid.UUID,
	paidBy *uuid.UUID,
	oldPurchasedBy uuid.UUID,
	oldPaidBy uuid.UUID,
) error {
	if err := s.validPrincipal(ctx, q, list, purchasedBy, oldPurchasedBy); err != nil {
		return apperr.Unprocessable("invalid_purchaser", "Quien compró debe ser un integrante activo.")
	}
	if paidBy != nil {
		if err := s.validPrincipal(ctx, q, list, *paidBy, oldPaidBy); err != nil {
			return apperr.Unprocessable("invalid_payer", "Quien pagó debe ser un integrante activo.")
		}
	}
	return nil
}

func (s *Service) validPrincipal(ctx context.Context, q data.Querier, list data.List, userID, existing uuid.UUID) error {
	if userID == uuid.Nil {
		return errors.New("nil principal")
	}
	if list.GroupID == nil {
		if list.PersonalOwnerID != nil && *list.PersonalOwnerID == userID {
			return nil
		}
		return errors.New("principal outside personal space")
	}
	role, err := data.GroupMemberRole(ctx, q, *list.GroupID, userID)
	if err == nil && role != "" {
		return nil
	}
	if !errors.Is(err, pgx.ErrNoRows) && err != nil {
		return err
	}
	if existing != uuid.Nil && existing == userID {
		exists, existsErr := data.UserExists(ctx, q, userID)
		if existsErr != nil {
			return existsErr
		}
		if exists {
			return nil
		}
	}
	return errors.New("inactive principal")
}

func derefUUID(value *uuid.UUID) uuid.UUID {
	if value == nil {
		return uuid.Nil
	}
	return *value
}
