package data

import (
	"context"
	"time"

	cestia "github.com/zchelalo/cestia/apps/api/internal/db/sqlc/cestia"
)

type ListExpense struct {
	ListID           string     `json:"list_id"`
	Name             string     `json:"name"`
	ArchivedAt       *time.Time `json:"archived_at"`
	ActualKnown      string     `json:"actual_known"`
	EstimatedKnown   string     `json:"estimated_known"`
	PendingEstimated string     `json:"pending_estimated"`
	PurchaseCount    int        `json:"purchase_count"`
	PendingCount     int        `json:"pending_count"`
}

func ExpensesByList(ctx context.Context, q Querier, scope ScopeFilter, from, to *time.Time, includeArchived bool) ([]ListExpense, error) {
	rows, err := q.Queries().ExpensesByList(ctx, cestia.ExpensesByListParams{
		FromAt: from, ToAt: to, PersonalOwnerID: scope.PersonalOwnerID,
		GroupID: scope.GroupID, IncludeArchived: includeArchived,
	})
	if err != nil {
		return nil, err
	}
	out := make([]ListExpense, 0, len(rows))
	for _, row := range rows {
		out = append(out, ListExpense{
			ListID: row.ListID.String(), Name: row.Name, ArchivedAt: row.ArchivedAt,
			ActualKnown: row.ActualKnown, EstimatedKnown: row.EstimatedKnown,
			PendingEstimated: row.PendingEstimated, PurchaseCount: int(row.PurchaseCount),
			PendingCount: int(row.PendingCount),
		})
	}
	return out, nil
}
