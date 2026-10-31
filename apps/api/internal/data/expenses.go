package data

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	cestia "github.com/zchelalo/cestia/apps/api/internal/db/sqlc/cestia"
)

type ScopeFilter struct {
	PersonalOwnerID *uuid.UUID
	GroupID         *uuid.UUID
}

type PurchaseFilter struct {
	From        *time.Time
	To          *time.Time
	ListID      *uuid.UUID
	PaidBy      *uuid.UUID
	PurchasedBy *uuid.UUID
	Cursor      string
	Limit       int
}

func Summary(ctx context.Context, q Querier, scope ScopeFilter, from, to *time.Time, includeArchived bool) (ExpenseSummary, error) {
	var s ExpenseSummary
	if scope.GroupID != nil {
		s.Scope = scope.GroupID.String()
	} else if scope.PersonalOwnerID != nil {
		s.Scope = "personal"
	}
	s.PeriodFrom, s.PeriodTo, s.IncludeArchived = from, to, includeArchived
	s.UnidentifiedPaid = "0.00"
	s.ByPayer = []PayerTotal{}

	queries := q.Queries()
	items, err := queries.SummaryItems(ctx, cestia.SummaryItemsParams{
		PersonalOwnerID: scope.PersonalOwnerID, GroupID: scope.GroupID, IncludeArchived: includeArchived,
	})
	if err != nil {
		return s, err
	}
	s.EstimatedKnown = items.EstimatedKnown
	s.EstimatedUnknownItems = int(items.EstimatedUnknownItems)
	s.PendingEstimated = items.PendingEstimated
	s.PendingCount = int(items.PendingCount)

	purchases, err := queries.SummaryPurchases(ctx, cestia.SummaryPurchasesParams{
		PersonalOwnerID: scope.PersonalOwnerID, GroupID: scope.GroupID,
		IncludeArchived: includeArchived, FromAt: from, ToAt: to,
	})
	if err != nil {
		return s, err
	}
	s.ActualKnown = purchases.ActualKnown
	s.PurchasesWithoutPrice = int(purchases.PurchasesWithoutPrice)
	s.PurchaseCount = int(purchases.PurchaseCount)
	s.PurchasedCount = int(purchases.PurchasedCount)

	payers, err := queries.SummaryPayers(ctx, cestia.SummaryPayersParams{
		PersonalOwnerID: scope.PersonalOwnerID, GroupID: scope.GroupID,
		IncludeArchived: includeArchived, FromAt: from, ToAt: to,
	})
	if err != nil {
		return s, err
	}
	for _, row := range payers {
		payer := PayerTotal{
			PaidBy: row.PaidBy, DisplayName: row.DisplayName, Amount: row.Amount,
			PurchaseCount: int(row.PurchaseCount),
		}
		if payer.PaidBy == nil {
			s.UnidentifiedPaid = payer.Amount
		}
		s.ByPayer = append(s.ByPayer, payer)
	}
	return s, nil
}

func Purchases(ctx context.Context, q Querier, scope ScopeFilter, filter PurchaseFilter) ([]Purchase, string, error) {
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 30
	}
	var cursorAt *time.Time
	var cursorID *uuid.UUID
	if filter.Cursor != "" {
		raw, err := base64.RawURLEncoding.DecodeString(filter.Cursor)
		if err == nil {
			parts := strings.SplitN(string(raw), "|", 2)
			if len(parts) == 2 {
				parsed, parseErr := time.Parse(time.RFC3339Nano, parts[0])
				parsedID, idErr := uuid.Parse(parts[1])
				if parseErr == nil && idErr == nil {
					cursorAt = &parsed
					cursorID = &parsedID
				}
			}
		}
	}
	rows, err := q.Queries().ListPurchaseHistory(ctx, cestia.ListPurchaseHistoryParams{
		PersonalOwnerID: scope.PersonalOwnerID, GroupID: scope.GroupID,
		FromAt: filter.From, ToAt: filter.To, ListID: filter.ListID, PaidBy: filter.PaidBy,
		PurchasedBy: filter.PurchasedBy, CursorAt: cursorAt, CursorID: cursorID,
		LimitCount: int32(filter.Limit + 1),
	})
	if err != nil {
		return nil, "", err
	}
	out := make([]Purchase, 0, len(rows))
	for _, row := range rows {
		out = append(out, purchaseFromParts(row.ID, row.ItemID, row.ListID, row.GroupID, row.ItemName,
			row.Quantity, row.Unit, row.ActualUnitPrice, row.TotalAmount, row.Currency,
			row.RecordedBy, row.PurchasedBy, row.PaidBy, row.PurchasedAt, row.Notes,
			row.ReversedBy, row.ReversedAt, row.ReversalReason, row.Version, row.CreatedAt, row.UpdatedAt,
			row.RecordedByName, row.PurchasedByName, row.PaidByName))
	}
	next := ""
	if len(out) > filter.Limit {
		out = out[:filter.Limit]
		last := out[len(out)-1]
		next = base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf("%s|%s",
			last.PurchasedAt.UTC().Format(time.RFC3339Nano), last.ID.String())))
	}
	return out, next, nil
}
