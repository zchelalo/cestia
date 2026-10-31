package shopping

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/zchelalo/cestia/apps/api/internal/apperr"
	"github.com/zchelalo/cestia/apps/api/internal/data"
)

type ExpenseFilter struct {
	GroupID         *uuid.UUID
	From            *time.Time
	To              *time.Time
	IncludeArchived bool
}

type HistoryFilter struct {
	GroupID     *uuid.UUID
	From        *time.Time
	To          *time.Time
	ListID      *uuid.UUID
	PaidBy      *uuid.UUID
	PurchasedBy *uuid.UUID
	Cursor      string
	Limit       int
}

func (s *Service) Summary(ctx context.Context, userID uuid.UUID, filter ExpenseFilter) (data.ExpenseSummary, error) {
	var summary data.ExpenseSummary
	err := s.store.WithinReadSnapshot(ctx, func(q data.Querier) error {
		scope, err := s.scopeWith(ctx, q, userID, filter.GroupID)
		if err != nil {
			return err
		}
		summary, err = data.Summary(ctx, q, scope, filter.From, filter.To, filter.IncludeArchived)
		return err
	})
	if err != nil {
		return data.ExpenseSummary{}, apperr.From(err)
	}
	return summary, nil
}

func (s *Service) ExpensesByList(ctx context.Context, userID uuid.UUID, filter ExpenseFilter) ([]data.ListExpense, error) {
	scope, err := s.scope(ctx, userID, filter.GroupID)
	if err != nil {
		return nil, err
	}
	items, err := data.ExpensesByList(ctx, s.store, scope, filter.From, filter.To, filter.IncludeArchived)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return items, nil
}

func (s *Service) PurchaseHistory(ctx context.Context, userID uuid.UUID, filter HistoryFilter) ([]data.Purchase, string, error) {
	scope, err := s.scope(ctx, userID, filter.GroupID)
	if err != nil {
		return nil, "", err
	}
	purchases, next, err := data.Purchases(ctx, s.store, scope, data.PurchaseFilter{
		From: filter.From, To: filter.To, ListID: filter.ListID, PaidBy: filter.PaidBy,
		PurchasedBy: filter.PurchasedBy, Cursor: filter.Cursor, Limit: filter.Limit,
	})
	if err != nil {
		return nil, "", apperr.Internal(err)
	}
	return purchases, next, nil
}

func (s *Service) scope(ctx context.Context, userID uuid.UUID, groupID *uuid.UUID) (data.ScopeFilter, error) {
	return s.scopeWith(ctx, s.store, userID, groupID)
}

func (s *Service) scopeWith(ctx context.Context, q data.Querier, userID uuid.UUID, groupID *uuid.UUID) (data.ScopeFilter, error) {
	if groupID == nil {
		return data.ScopeFilter{PersonalOwnerID: &userID}, nil
	}
	if _, err := s.groupAccessWith(ctx, q, userID, *groupID); err != nil {
		return data.ScopeFilter{}, err
	}
	return data.ScopeFilter{GroupID: groupID}, nil
}
