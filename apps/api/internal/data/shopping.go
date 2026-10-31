package data

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	cestia "github.com/zchelalo/cestia/apps/api/internal/db/sqlc/cestia"
)

func listFromParts(id uuid.UUID, name, description string, personalOwnerID, groupID *uuid.UUID,
	createdBy uuid.UUID, archivedAt *time.Time, version int64, createdAt, updatedAt time.Time,
	role string, pendingCount, purchasedCount int64) List {
	return List{
		ID: id, Name: name, Description: description, PersonalOwnerID: personalOwnerID,
		GroupID: groupID, CreatedBy: createdBy, ArchivedAt: archivedAt, Version: version,
		CreatedAt: createdAt, UpdatedAt: updatedAt, Role: role,
		PendingCount: int(pendingCount), PurchasedCount: int(purchasedCount),
	}
}

func ListForUser(ctx context.Context, q Querier, id, userID uuid.UUID, lock bool) (List, error) {
	if lock {
		row, err := q.Queries().GetListForUserForUpdate(ctx, cestia.GetListForUserForUpdateParams{ID: id, UserID: userID})
		if err != nil {
			return List{}, err
		}
		return listFromParts(row.ID, row.Name, row.Description, row.PersonalOwnerID, row.GroupID,
			row.CreatedBy, row.ArchivedAt, row.Version, row.CreatedAt, row.UpdatedAt, row.Role,
			row.PendingCount, row.PurchasedCount), nil
	}
	row, err := q.Queries().GetListForUser(ctx, cestia.GetListForUserParams{ID: id, UserID: userID})
	if err != nil {
		return List{}, err
	}
	return listFromParts(row.ID, row.Name, row.Description, row.PersonalOwnerID, row.GroupID,
		row.CreatedBy, row.ArchivedAt, row.Version, row.CreatedAt, row.UpdatedAt, row.Role,
		row.PendingCount, row.PurchasedCount), nil
}

func ListForUserShare(ctx context.Context, q Querier, id, userID uuid.UUID) (List, error) {
	row, err := q.Queries().GetListForUserForShare(ctx, cestia.GetListForUserForShareParams{ID: id, UserID: userID})
	if err != nil {
		return List{}, err
	}
	return listFromParts(row.ID, row.Name, row.Description, row.PersonalOwnerID, row.GroupID,
		row.CreatedBy, row.ArchivedAt, row.Version, row.CreatedAt, row.UpdatedAt, row.Role,
		row.PendingCount, row.PurchasedCount), nil
}

func ListsForScope(ctx context.Context, q Querier, userID uuid.UUID, groupID *uuid.UUID, includeArchived bool) ([]List, error) {
	if groupID == nil {
		rows, err := q.Queries().ListPersonalLists(ctx, cestia.ListPersonalListsParams{
			UserID: userID, IncludeArchived: includeArchived,
		})
		if err != nil {
			return nil, err
		}
		out := make([]List, 0, len(rows))
		for _, row := range rows {
			out = append(out, listFromParts(row.ID, row.Name, row.Description, row.PersonalOwnerID,
				row.GroupID, row.CreatedBy, row.ArchivedAt, row.Version, row.CreatedAt, row.UpdatedAt,
				row.Role, row.PendingCount, row.PurchasedCount))
		}
		return out, nil
	}
	rows, err := q.Queries().ListGroupLists(ctx, cestia.ListGroupListsParams{
		UserID: userID, GroupID: *groupID, IncludeArchived: includeArchived,
	})
	if err != nil {
		return nil, err
	}
	out := make([]List, 0, len(rows))
	for _, row := range rows {
		out = append(out, listFromParts(row.ID, row.Name, row.Description, row.PersonalOwnerID,
			row.GroupID, row.CreatedBy, row.ArchivedAt, row.Version, row.CreatedAt, row.UpdatedAt,
			row.Role, row.PendingCount, row.PurchasedCount))
	}
	return out, nil
}

func CreateList(ctx context.Context, q Querier, l List) error {
	return q.Queries().CreateList(ctx, cestia.CreateListParams{
		ID: l.ID, Name: l.Name, Description: l.Description, PersonalOwnerID: l.PersonalOwnerID,
		GroupID: l.GroupID, CreatedBy: l.CreatedBy, CreatedAt: l.CreatedAt,
	})
}

func UpdateList(ctx context.Context, q Querier, id uuid.UUID, name, description string, version int64, at time.Time) (bool, error) {
	rows, err := q.Queries().UpdateList(ctx, cestia.UpdateListParams{
		ID: id, Name: name, Description: description, Version: version, At: at,
	})
	return rows == 1, err
}

func SetListArchived(ctx context.Context, q Querier, id uuid.UUID, archived bool, version int64, at time.Time) (bool, error) {
	if archived {
		rows, err := q.Queries().ArchiveList(ctx, cestia.ArchiveListParams{ID: id, Version: version, At: at})
		return rows == 1, err
	}
	rows, err := q.Queries().RestoreList(ctx, cestia.RestoreListParams{ID: id, Version: version, At: at})
	return rows == 1, err
}

func itemFromRow(row cestia.GetItemRow) Item {
	return Item{
		ID: row.ID, ListID: row.ListID, Name: row.Name, Description: row.Description,
		Quantity: row.Quantity, Unit: row.Unit, EstimatedUnitPrice: row.EstimatedUnitPrice,
		CreatedBy: row.CreatedBy, AssignedTo: row.AssignedTo, Notes: row.Notes,
		DeletedAt: row.DeletedAt, Version: row.Version, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		AssignedToName: row.AssignedToName,
	}
}

func itemFromListRow(row cestia.ListItemsRow) Item {
	return Item{
		ID: row.ID, ListID: row.ListID, Name: row.Name, Description: row.Description,
		Quantity: row.Quantity, Unit: row.Unit, EstimatedUnitPrice: row.EstimatedUnitPrice,
		CreatedBy: row.CreatedBy, AssignedTo: row.AssignedTo, Notes: row.Notes,
		DeletedAt: row.DeletedAt, Version: row.Version, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		AssignedToName: row.AssignedToName,
	}
}

func itemFromUpdateRow(row cestia.GetItemForUpdateRow) Item {
	return Item{
		ID: row.ID, ListID: row.ListID, Name: row.Name, Description: row.Description,
		Quantity: row.Quantity, Unit: row.Unit, EstimatedUnitPrice: row.EstimatedUnitPrice,
		CreatedBy: row.CreatedBy, AssignedTo: row.AssignedTo, Notes: row.Notes,
		DeletedAt: row.DeletedAt, Version: row.Version, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		AssignedToName: row.AssignedToName,
	}
}

func itemFromUpdateAnyRow(row cestia.GetItemForUpdateAnyRow) Item {
	return Item{
		ID: row.ID, ListID: row.ListID, Name: row.Name, Description: row.Description,
		Quantity: row.Quantity, Unit: row.Unit, EstimatedUnitPrice: row.EstimatedUnitPrice,
		CreatedBy: row.CreatedBy, AssignedTo: row.AssignedTo, Notes: row.Notes,
		DeletedAt: row.DeletedAt, Version: row.Version, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		AssignedToName: row.AssignedToName,
	}
}

func ListItems(ctx context.Context, q Querier, listID uuid.UUID) ([]Item, error) {
	rows, err := q.Queries().ListItems(ctx, listID)
	if err != nil {
		return nil, err
	}
	out := make([]Item, 0, len(rows))
	for _, row := range rows {
		out = append(out, itemFromListRow(row))
	}
	return out, nil
}

func ItemForUser(ctx context.Context, q Querier, itemID, userID uuid.UUID, lock bool) (Item, List, error) {
	row, err := q.Queries().GetItem(ctx, itemID)
	if err != nil {
		return Item{}, List{}, err
	}
	item := itemFromRow(row)
	list, err := ListForUser(ctx, q, item.ListID, userID, lock)
	return item, list, err
}

func ItemForUpdate(ctx context.Context, q Querier, itemID uuid.UUID) (Item, error) {
	row, err := q.Queries().GetItemForUpdate(ctx, itemID)
	if err != nil {
		return Item{}, err
	}
	return itemFromUpdateRow(row), nil
}

// ItemForUpdateAny locks an item even when it has been logically deleted, so
// purchase history corrections and reversals keep working after item deletion.
func ItemForUpdateAny(ctx context.Context, q Querier, itemID uuid.UUID) (Item, error) {
	row, err := q.Queries().GetItemForUpdateAny(ctx, itemID)
	if err != nil {
		return Item{}, err
	}
	return itemFromUpdateAnyRow(row), nil
}

func CreateItem(ctx context.Context, q Querier, item Item) error {
	return q.Queries().CreateItem(ctx, cestia.CreateItemParams{
		ID: item.ID, ListID: item.ListID, Name: item.Name, Description: item.Description,
		Quantity: item.Quantity, Unit: item.Unit, EstimatedUnitPrice: item.EstimatedUnitPrice,
		CreatedBy: item.CreatedBy, AssignedTo: item.AssignedTo, Notes: item.Notes, CreatedAt: item.CreatedAt,
	})
}

func UpdateItem(ctx context.Context, q Querier, item Item) (bool, error) {
	rows, err := q.Queries().UpdateItem(ctx, cestia.UpdateItemParams{
		ID: item.ID, Name: item.Name, Description: item.Description, Quantity: item.Quantity,
		Unit: item.Unit, EstimatedUnitPrice: item.EstimatedUnitPrice, AssignedTo: item.AssignedTo,
		Notes: item.Notes, At: item.UpdatedAt, Version: item.Version,
	})
	return rows == 1, err
}

func DeleteItem(ctx context.Context, q Querier, id uuid.UUID, version int64, at time.Time) (bool, error) {
	rows, err := q.Queries().DeleteItem(ctx, cestia.DeleteItemParams{ID: id, Version: version, At: at})
	return rows == 1, err
}

func purchaseFromParts(
	id, itemID, listID uuid.UUID, groupID *uuid.UUID, itemName, quantity, unit string,
	actualUnitPrice, totalAmount *string, currency string, recordedBy, purchasedBy uuid.UUID,
	paidBy *uuid.UUID, purchasedAt time.Time, notes string, reversedBy *uuid.UUID,
	reversedAt *time.Time, reversalReason *string, version int64, createdAt, updatedAt time.Time,
	recordedByName, purchasedByName, paidByName string,
) Purchase {
	var paidName *string
	if paidBy != nil {
		paidName = &paidByName
	}
	return Purchase{
		ID: id, ItemID: itemID, ListID: listID, GroupID: groupID, ItemName: itemName,
		Quantity: quantity, Unit: unit, ActualUnitPrice: actualUnitPrice, TotalAmount: totalAmount,
		Currency: currency, RecordedBy: recordedBy, PurchasedBy: purchasedBy, PaidBy: paidBy,
		PurchasedAt: purchasedAt, Notes: notes, ReversedBy: reversedBy, ReversedAt: reversedAt,
		ReversalReason: reversalReason, Version: version, CreatedAt: createdAt, UpdatedAt: updatedAt,
		PurchasedByName: purchasedByName, PaidByName: paidName, RecordedByName: recordedByName,
	}
}

func ActivePurchaseForItem(ctx context.Context, q Querier, itemID uuid.UUID, _ bool) (Purchase, error) {
	row, err := q.Queries().GetActivePurchaseForItem(ctx, itemID)
	if err != nil {
		return Purchase{}, err
	}
	return purchaseFromParts(row.ID, row.ItemID, row.ListID, row.GroupID, row.ItemName,
		row.Quantity, row.Unit, row.ActualUnitPrice, row.TotalAmount, row.Currency,
		row.RecordedBy, row.PurchasedBy, row.PaidBy, row.PurchasedAt, row.Notes,
		row.ReversedBy, row.ReversedAt, row.ReversalReason, row.Version, row.CreatedAt, row.UpdatedAt,
		row.RecordedByName, row.PurchasedByName, row.PaidByName), nil
}

func PurchaseForUser(ctx context.Context, q Querier, purchaseID, userID uuid.UUID, lock bool) (Purchase, error) {
	if lock {
		row, err := q.Queries().GetPurchaseForUserForUpdate(ctx, cestia.GetPurchaseForUserForUpdateParams{
			ID: purchaseID, UserID: userID,
		})
		if err != nil {
			return Purchase{}, err
		}
		return purchaseFromParts(row.ID, row.ItemID, row.ListID, row.GroupID, row.ItemName,
			row.Quantity, row.Unit, row.ActualUnitPrice, row.TotalAmount, row.Currency,
			row.RecordedBy, row.PurchasedBy, row.PaidBy, row.PurchasedAt, row.Notes,
			row.ReversedBy, row.ReversedAt, row.ReversalReason, row.Version, row.CreatedAt, row.UpdatedAt,
			row.RecordedByName, row.PurchasedByName, row.PaidByName), nil
	}
	row, err := q.Queries().GetPurchaseForUser(ctx, cestia.GetPurchaseForUserParams{
		ID: purchaseID, UserID: userID,
	})
	if err != nil {
		return Purchase{}, err
	}
	return purchaseFromParts(row.ID, row.ItemID, row.ListID, row.GroupID, row.ItemName,
		row.Quantity, row.Unit, row.ActualUnitPrice, row.TotalAmount, row.Currency,
		row.RecordedBy, row.PurchasedBy, row.PaidBy, row.PurchasedAt, row.Notes,
		row.ReversedBy, row.ReversedAt, row.ReversalReason, row.Version, row.CreatedAt, row.UpdatedAt,
		row.RecordedByName, row.PurchasedByName, row.PaidByName), nil
}

func CreatePurchase(ctx context.Context, q Querier, p Purchase) error {
	return q.Queries().CreatePurchase(ctx, cestia.CreatePurchaseParams{
		ID: p.ID, ItemID: p.ItemID, ItemName: p.ItemName, Quantity: p.Quantity, Unit: p.Unit,
		ActualUnitPrice: p.ActualUnitPrice, RecordedBy: p.RecordedBy, PurchasedBy: p.PurchasedBy,
		PaidBy: p.PaidBy, PurchasedAt: p.PurchasedAt, Notes: p.Notes, CreatedAt: p.CreatedAt,
	})
}

func UpdatePurchase(ctx context.Context, q Querier, p Purchase) (bool, error) {
	rows, err := q.Queries().UpdatePurchase(ctx, cestia.UpdatePurchaseParams{
		ID: p.ID, ActualUnitPrice: p.ActualUnitPrice, PurchasedBy: p.PurchasedBy, PaidBy: p.PaidBy,
		PurchasedAt: p.PurchasedAt, Notes: p.Notes, At: p.UpdatedAt, Version: p.Version,
	})
	return rows == 1, err
}

func ReversePurchase(ctx context.Context, q Querier, id, actor uuid.UUID, reason string, version int64, at time.Time) (bool, error) {
	rows, err := q.Queries().ReversePurchase(ctx, cestia.ReversePurchaseParams{
		ID: id, ActorID: actor, Reason: reason, Version: version, At: at,
	})
	return rows == 1, err
}

func TouchList(ctx context.Context, q Querier, listID uuid.UUID, at time.Time) error {
	return q.Queries().TouchList(ctx, cestia.TouchListParams{ID: listID, At: at})
}

func TouchItem(ctx context.Context, q Querier, itemID uuid.UUID, at time.Time) error {
	return q.Queries().TouchItem(ctx, cestia.TouchItemParams{ID: itemID, At: at})
}

func activityFromParts(id uuid.UUID, scopeType string, actorID *uuid.UUID, actorName *string,
	groupID, listID, itemID, purchaseID *uuid.UUID, action string, payload json.RawMessage,
	createdAt time.Time) Activity {
	var decoded any
	_ = json.Unmarshal(payload, &decoded)
	return Activity{
		ID: id, ScopeType: scopeType, ActorID: actorID, ActorName: actorName, GroupID: groupID,
		ListID: listID, ItemID: itemID, PurchaseID: purchaseID, Action: action,
		Payload: decoded, CreatedAt: createdAt,
	}
}

func LastActivity(ctx context.Context, q Querier, scopeType string, userID uuid.UUID, groupID *uuid.UUID, limit int) ([]Activity, error) {
	if groupID != nil {
		rows, err := q.Queries().ListGroupActivity(ctx, cestia.ListGroupActivityParams{
			GroupID: *groupID, LimitCount: int32(limit),
		})
		if err != nil {
			return nil, err
		}
		out := make([]Activity, 0, len(rows))
		for _, row := range rows {
			out = append(out, activityFromParts(row.ID, row.ScopeType, row.ActorID, row.ActorName,
				row.GroupID, row.ListID, row.ItemID, row.PurchaseID, row.Action, row.Payload, row.CreatedAt))
		}
		return out, nil
	}
	rows, err := q.Queries().ListPersonalActivity(ctx, cestia.ListPersonalActivityParams{
		UserID: userID, LimitCount: int32(limit),
	})
	if err != nil {
		return nil, err
	}
	out := make([]Activity, 0, len(rows))
	for _, row := range rows {
		out = append(out, activityFromParts(row.ID, row.ScopeType, row.ActorID, row.ActorName,
			row.GroupID, row.ListID, row.ItemID, row.PurchaseID, row.Action, row.Payload, row.CreatedAt))
	}
	return out, nil
}
