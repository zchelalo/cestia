package data

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	cestia "github.com/zchelalo/cestia/apps/api/internal/db/sqlc/cestia"
)

type ListExpense struct {
	ListID           string     `json:"list_id"`
	Name             string     `json:"name"`
	ArchivedAt       *time.Time `json:"archived_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	ActualKnown      string     `json:"actual_known"`
	EstimatedKnown   string     `json:"estimated_known"`
	PendingEstimated string     `json:"pending_estimated"`
	PurchaseCount    int        `json:"purchase_count"`
	PendingCount     int        `json:"pending_count"`
}

func ExpensesByList(ctx context.Context, q Querier, scope ScopeFilter, from, to *time.Time, includeArchived bool, cursor string, limit int) ([]ListExpense, string, error) {
	page, fetch := pageLimit(limit)
	var payload updatedCursor
	if err := decodePageCursor(cursor, "expenses-by-list", &payload); err != nil {
		return nil, "", err
	}
	var cursorArchived *int32
	var cursorUpdatedAt *time.Time
	var cursorID *uuid.UUID
	if cursor != "" {
		archived := int32(payload.Archived)
		cursorArchived, cursorUpdatedAt, cursorID = &archived, &payload.UpdatedAt, &payload.ID
	}
	rows, err := q.Queries().ExpensesByList(ctx, cestia.ExpensesByListParams{
		FromAt: from, ToAt: to, PersonalOwnerID: scope.PersonalOwnerID,
		GroupID: scope.GroupID, IncludeArchived: includeArchived,
		CursorArchived: cursorArchived, CursorUpdatedAt: cursorUpdatedAt, CursorID: cursorID,
		LimitCount: fetch,
	})
	if err != nil {
		return nil, "", err
	}
	out := make([]ListExpense, 0, len(rows))
	for _, row := range rows {
		out = append(out, ListExpense{
			ListID: row.ListID.String(), Name: row.Name, ArchivedAt: row.ArchivedAt,
			UpdatedAt: row.UpdatedAt, ActualKnown: row.ActualKnown,
			EstimatedKnown: row.EstimatedKnown, PendingEstimated: row.PendingEstimated,
			PurchaseCount: int(row.PurchaseCount), PendingCount: int(row.PendingCount),
		})
	}
	out, next := pageOf(out, page, func(item ListExpense) string {
		archived := 0
		if item.ArchivedAt != nil {
			archived = 1
		}
		return encodePageCursor("expenses-by-list", updatedCursor{
			Archived: archived, UpdatedAt: item.UpdatedAt, ID: uuid.MustParse(item.ListID),
		})
	})
	return out, next, nil
}

func PayerExpenses(ctx context.Context, q Querier, scope ScopeFilter, from, to *time.Time, includeArchived bool, cursor string, limit int) ([]PayerTotal, string, error) {
	page, fetch := pageLimit(limit)
	var payload payerCursor
	if err := decodePageCursor(cursor, "expenses-by-payer", &payload); err != nil {
		return nil, "", err
	}
	var cursorAmount pgtype.Numeric
	var cursorPayerID *uuid.UUID
	if cursor != "" {
		if err := cursorAmount.Scan(payload.Amount); err != nil {
			return nil, "", ErrInvalidCursor
		}
		cursorPayerID = &payload.PayerID
	}
	rows, err := q.Queries().ListPayerExpenses(ctx, cestia.ListPayerExpensesParams{
		FromAt: from, ToAt: to, PersonalOwnerID: scope.PersonalOwnerID,
		GroupID: scope.GroupID, IncludeArchived: includeArchived,
		CursorAmount: cursorAmount, CursorPayerID: cursorPayerID, LimitCount: fetch,
	})
	if err != nil {
		return nil, "", err
	}
	out := make([]PayerTotal, 0, len(rows))
	for _, row := range rows {
		out = append(out, PayerTotal{
			PaidBy: row.PaidBy, DisplayName: row.DisplayName, Amount: row.Amount,
			PurchaseCount: int(row.PurchaseCount),
		})
	}
	out, next := pageOf(out, page, func(payer PayerTotal) string {
		payerID := uuid.Nil
		if payer.PaidBy != nil {
			payerID = *payer.PaidBy
		}
		return encodePageCursor("expenses-by-payer", payerCursor{Amount: payer.Amount, PayerID: payerID})
	})
	return out, next, nil
}
