package data

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	cestia "github.com/zchelalo/cestia/apps/api/internal/db/sqlc/cestia"
)

func groupFromParts(
	id uuid.UUID,
	name, description string,
	ownerUserID, createdBy uuid.UUID,
	archivedAt *time.Time,
	version int64,
	createdAt, updatedAt time.Time,
	role string,
	memberCount, listCount int64,
) Group {
	return Group{
		ID: id, Name: name, Description: description, OwnerUserID: ownerUserID,
		CreatedBy: createdBy, ArchivedAt: archivedAt, Version: version,
		CreatedAt: createdAt, UpdatedAt: updatedAt, Role: role,
		MemberCount: int(memberCount), ListCount: int(listCount),
	}
}

func GroupForUser(ctx context.Context, q Querier, id, userID uuid.UUID, lock bool) (Group, error) {
	params := cestia.GetGroupForUserParams{ID: id, UserID: userID}
	if lock {
		row, err := q.Queries().GetGroupForUserForUpdate(ctx, cestia.GetGroupForUserForUpdateParams(params))
		if err != nil {
			return Group{}, err
		}
		return groupFromParts(row.ID, row.Name, row.Description, row.OwnerUserID, row.CreatedBy,
			row.ArchivedAt, row.Version, row.CreatedAt, row.UpdatedAt, row.Role, row.MemberCount, row.ListCount), nil
	}
	row, err := q.Queries().GetGroupForUser(ctx, params)
	if err != nil {
		return Group{}, err
	}
	return groupFromParts(row.ID, row.Name, row.Description, row.OwnerUserID, row.CreatedBy,
		row.ArchivedAt, row.Version, row.CreatedAt, row.UpdatedAt, row.Role, row.MemberCount, row.ListCount), nil
}

// GroupForUserLocked locks the group row first and then reads the actor's
// membership. The second statement sees the membership as of after the lock
// wait, so a concurrent removal cannot leave stale permissions behind.
func GroupForUserLocked(ctx context.Context, q Querier, id, userID uuid.UUID) (Group, error) {
	if _, err := GroupByIDForUpdate(ctx, q, id); err != nil {
		return Group{}, err
	}
	return GroupForUser(ctx, q, id, userID, false)
}

// GroupForUserSharedLocked is the shared-lock variant used by operations that
// only need to block group lifecycle and membership mutations.
func GroupForUserSharedLocked(ctx context.Context, q Querier, id, userID uuid.UUID) (Group, error) {
	if _, err := q.Queries().LockGroupForShare(ctx, id); err != nil {
		return Group{}, err
	}
	return GroupForUser(ctx, q, id, userID, false)
}

func GroupForShare(ctx context.Context, q Querier, id, userID uuid.UUID) (Group, error) {
	row, err := q.Queries().GetGroupForUserForShare(ctx, cestia.GetGroupForUserForShareParams{ID: id, UserID: userID})
	if err != nil {
		return Group{}, err
	}
	return groupFromParts(row.ID, row.Name, row.Description, row.OwnerUserID, row.CreatedBy,
		row.ArchivedAt, row.Version, row.CreatedAt, row.UpdatedAt, row.Role, row.MemberCount, row.ListCount), nil
}

func GroupByIDForUpdate(ctx context.Context, q Querier, id uuid.UUID) (Group, error) {
	row, err := q.Queries().GetGroupByIDForUpdate(ctx, id)
	if err != nil {
		return Group{}, err
	}
	return groupFromParts(row.ID, row.Name, row.Description, row.OwnerUserID, row.CreatedBy,
		row.ArchivedAt, row.Version, row.CreatedAt, row.UpdatedAt, row.Role, row.MemberCount, row.ListCount), nil
}

func ListGroups(ctx context.Context, q Querier, userID uuid.UUID, cursor string, limit int) ([]Group, string, error) {
	page, fetch := pageLimit(limit)
	var payload nameCursor
	if err := decodePageCursor(cursor, "groups", &payload); err != nil {
		return nil, "", err
	}
	var cursorArchived *int32
	var cursorName *string
	var cursorID *uuid.UUID
	if cursor != "" {
		archived := int32(payload.Archived)
		cursorArchived, cursorName, cursorID = &archived, &payload.Name, &payload.ID
	}
	rows, err := q.Queries().ListGroupsForUser(ctx, cestia.ListGroupsForUserParams{
		UserID: userID, CursorArchived: cursorArchived, CursorName: cursorName,
		CursorID: cursorID, LimitCount: fetch,
	})
	if err != nil {
		return nil, "", err
	}
	out := make([]Group, 0, len(rows))
	for _, row := range rows {
		out = append(out, groupFromParts(row.ID, row.Name, row.Description, row.OwnerUserID, row.CreatedBy,
			row.ArchivedAt, row.Version, row.CreatedAt, row.UpdatedAt, row.Role, row.MemberCount, row.ListCount))
	}
	out, next := pageOf(out, page, func(g Group) string {
		archived := 0
		if g.ArchivedAt != nil {
			archived = 1
		}
		return encodePageCursor("groups", nameCursor{Archived: archived, Name: strings.ToLower(g.Name), ID: g.ID})
	})
	return out, next, nil
}

func CountGroups(ctx context.Context, q Querier, userID uuid.UUID) (int, error) {
	count, err := q.Queries().CountGroupsForUser(ctx, userID)
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

func CountGroupMembersTotal(ctx context.Context, q Querier, userID uuid.UUID) (int, error) {
	count, err := q.Queries().CountGroupMembersForUser(ctx, userID)
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

func CreateGroup(ctx context.Context, q Querier, g Group) error {
	return q.Queries().CreateGroup(ctx, cestia.CreateGroupParams{
		ID: g.ID, Name: g.Name, Description: g.Description,
		OwnerUserID: g.OwnerUserID, CreatedAt: g.CreatedAt,
	})
}

func AddGroupMember(ctx context.Context, q Querier, groupID, userID uuid.UUID, role string, at time.Time) error {
	return q.Queries().UpsertGroupMember(ctx, cestia.UpsertGroupMemberParams{
		GroupID: groupID, UserID: userID, Role: role, JoinedAt: at,
	})
}

func ListGroupMembers(ctx context.Context, q Querier, groupID uuid.UUID, includeOwner bool, cursor string, limit int) ([]GroupMember, string, error) {
	page, fetch := pageLimit(limit)
	var payload memberCursor
	if err := decodePageCursor(cursor, "members", &payload); err != nil {
		return nil, "", err
	}
	var cursorPriority *int32
	var cursorName *string
	var cursorID *uuid.UUID
	if cursor != "" {
		priority := int32(payload.Priority)
		cursorPriority, cursorName, cursorID = &priority, &payload.Name, &payload.ID
	}
	rows, err := q.Queries().ListGroupMembers(ctx, cestia.ListGroupMembersParams{
		GroupID: groupID, IncludeOwner: includeOwner, CursorPriority: cursorPriority,
		CursorName: cursorName, CursorID: cursorID, LimitCount: fetch,
	})
	if err != nil {
		return nil, "", err
	}
	out := make([]GroupMember, 0, len(rows))
	for _, row := range rows {
		out = append(out, GroupMember{
			UserID: row.UserID, DisplayName: row.DisplayName, Email: row.Email, Role: row.Role,
			JoinedAt: row.JoinedAt, RemovedAt: row.RemovedAt,
		})
	}
	out, next := pageOf(out, page, func(m GroupMember) string {
		priority := 2
		if m.Role == "owner" {
			priority = 0
		} else if m.Role == "admin" {
			priority = 1
		}
		return encodePageCursor("members", memberCursor{
			Priority: priority, Name: strings.ToLower(m.DisplayName), ID: m.UserID,
		})
	})
	return out, next, nil
}

func GroupMemberRole(ctx context.Context, q Querier, groupID, userID uuid.UUID) (string, error) {
	return q.Queries().GetGroupMemberRole(ctx, cestia.GetGroupMemberRoleParams{
		GroupID: groupID, UserID: userID,
	})
}

func ActiveGroupMemberEmail(ctx context.Context, q Querier, groupID uuid.UUID, email string) (bool, error) {
	return q.Queries().ActiveGroupMemberEmailExists(ctx, cestia.ActiveGroupMemberEmailExistsParams{
		GroupID: groupID, Email: email,
	})
}

func RemoveGroupMember(ctx context.Context, q Querier, groupID, userID uuid.UUID, at time.Time) error {
	return q.Queries().RemoveGroupMember(ctx, cestia.RemoveGroupMemberParams{
		GroupID: groupID, UserID: userID, At: at,
	})
}

func SetMemberRole(ctx context.Context, q Querier, groupID, userID uuid.UUID, role string) error {
	return q.Queries().SetGroupMemberRole(ctx, cestia.SetGroupMemberRoleParams{
		GroupID: groupID, UserID: userID, Role: role,
	})
}

func UpdateGroup(ctx context.Context, q Querier, id uuid.UUID, name, description string, version int64, at time.Time) (bool, error) {
	rows, err := q.Queries().UpdateGroup(ctx, cestia.UpdateGroupParams{
		ID: id, Name: name, Description: description, Version: version, At: at,
	})
	return rows == 1, err
}

func SetGroupArchived(ctx context.Context, q Querier, id uuid.UUID, archived bool, version int64, at time.Time) (bool, error) {
	if archived {
		rows, err := q.Queries().ArchiveGroup(ctx, cestia.ArchiveGroupParams{ID: id, Version: version, At: at})
		return rows == 1, err
	}
	rows, err := q.Queries().RestoreGroup(ctx, cestia.RestoreGroupParams{ID: id, Version: version, At: at})
	return rows == 1, err
}

func TransferOwner(ctx context.Context, q Querier, groupID, oldOwner, newOwner uuid.UUID, at time.Time) error {
	queries := q.Queries()
	if err := queries.SetGroupOwner(ctx, cestia.SetGroupOwnerParams{
		ID: groupID, NewOwnerUserID: newOwner, At: at,
	}); err != nil {
		return err
	}
	return queries.SetGroupMemberAdmin(ctx, cestia.SetGroupMemberAdminParams{
		GroupID: groupID, UserID: oldOwner,
	})
}

func CreateInvitation(ctx context.Context, q Querier, inv *Invitation) error {
	return q.Queries().CreateGroupInvitation(ctx, cestia.CreateGroupInvitationParams{
		ID: inv.ID, GroupID: inv.GroupID, Email: inv.Email, InvitedBy: inv.InvitedBy,
		TokenHash: inv.TokenHash, Generation: int32(inv.Generation),
		ExpiresAt: inv.ExpiresAt, CreatedAt: inv.CreatedAt,
	})
}

func UpdateInvitationToken(ctx context.Context, q Querier, id uuid.UUID, tokenHash []byte, generation int, expires time.Time, version int64, at time.Time) (bool, error) {
	rows, err := q.Queries().UpdateInvitationToken(ctx, cestia.UpdateInvitationTokenParams{
		ID: id, TokenHash: tokenHash, Generation: int32(generation), ExpiresAt: expires,
		At: at, Version: version,
	})
	return rows == 1, err
}

func invitationFromParts(id, groupID uuid.UUID, groupName, email string, invitedBy uuid.UUID,
	status string, generation int32, expiresAt time.Time, resolvedAt *time.Time, resolvedBy *uuid.UUID,
	version int64, createdAt, updatedAt time.Time) Invitation {
	return Invitation{
		ID: id, GroupID: groupID, GroupName: groupName, Email: email, InvitedBy: invitedBy,
		Status: status, Generation: int(generation), ExpiresAt: expiresAt, ResolvedAt: resolvedAt,
		ResolvedBy: resolvedBy, Version: version, CreatedAt: createdAt, UpdatedAt: updatedAt,
	}
}

func ListInvitations(ctx context.Context, q Querier, groupID uuid.UUID, status, cursor string, limit int) ([]Invitation, string, error) {
	page, fetch := pageLimit(limit)
	var payload createdCursor
	if err := decodePageCursor(cursor, "invitations", &payload); err != nil {
		return nil, "", err
	}
	var cursorCreatedAt *time.Time
	var cursorID *uuid.UUID
	if cursor != "" {
		cursorCreatedAt, cursorID = &payload.CreatedAt, &payload.ID
	}
	rows, err := q.Queries().ListInvitations(ctx, cestia.ListInvitationsParams{
		GroupID: groupID, StatusFilter: status, CursorCreatedAt: cursorCreatedAt,
		CursorID: cursorID, LimitCount: fetch,
	})
	if err != nil {
		return nil, "", err
	}
	out := make([]Invitation, 0, len(rows))
	for _, row := range rows {
		out = append(out, invitationFromParts(row.ID, row.GroupID, row.GroupName, row.Email, row.InvitedBy,
			row.Status, row.Generation, row.ExpiresAt, row.ResolvedAt, row.ResolvedBy, row.Version, row.CreatedAt, row.UpdatedAt))
	}
	out, next := pageOf(out, page, func(inv Invitation) string {
		return encodePageCursor("invitations", createdCursor{CreatedAt: inv.CreatedAt, ID: inv.ID})
	})
	return out, next, nil
}

func InvitationByID(ctx context.Context, q Querier, id uuid.UUID, lock bool) (Invitation, error) {
	if lock {
		row, err := q.Queries().GetInvitationByIDForUpdate(ctx, id)
		if err != nil {
			return Invitation{}, err
		}
		return invitationFromParts(row.ID, row.GroupID, row.GroupName, row.Email, row.InvitedBy,
			row.Status, row.Generation, row.ExpiresAt, row.ResolvedAt, row.ResolvedBy, row.Version, row.CreatedAt, row.UpdatedAt), nil
	}
	row, err := q.Queries().GetInvitationByID(ctx, id)
	if err != nil {
		return Invitation{}, err
	}
	return invitationFromParts(row.ID, row.GroupID, row.GroupName, row.Email, row.InvitedBy,
		row.Status, row.Generation, row.ExpiresAt, row.ResolvedAt, row.ResolvedBy, row.Version, row.CreatedAt, row.UpdatedAt), nil
}

func InvitationByToken(ctx context.Context, q Querier, hash []byte, lock bool) (Invitation, error) {
	if lock {
		row, err := q.Queries().GetInvitationByTokenForUpdate(ctx, hash)
		if err != nil {
			return Invitation{}, err
		}
		return invitationFromParts(row.ID, row.GroupID, row.GroupName, row.Email, row.InvitedBy,
			row.Status, row.Generation, row.ExpiresAt, row.ResolvedAt, row.ResolvedBy, row.Version, row.CreatedAt, row.UpdatedAt), nil
	}
	row, err := q.Queries().GetInvitationByToken(ctx, hash)
	if err != nil {
		return Invitation{}, err
	}
	return invitationFromParts(row.ID, row.GroupID, row.GroupName, row.Email, row.InvitedBy,
		row.Status, row.Generation, row.ExpiresAt, row.ResolvedAt, row.ResolvedBy, row.Version, row.CreatedAt, row.UpdatedAt), nil
}

func ResolveInvitation(ctx context.Context, q Querier, id uuid.UUID, status string, by uuid.UUID, at time.Time) error {
	return q.Queries().ResolveInvitation(ctx, cestia.ResolveInvitationParams{
		ID: id, Status: status, At: at, ResolvedBy: by,
	})
}

func CancelInvitation(ctx context.Context, q Querier, id uuid.UUID, at time.Time) error {
	return q.Queries().CancelInvitation(ctx, cestia.CancelInvitationParams{ID: id, At: at})
}

func ExpirePendingInvitation(ctx context.Context, q Querier, groupID uuid.UUID, email string, now time.Time) error {
	return q.Queries().ExpirePendingInvitation(ctx, cestia.ExpirePendingInvitationParams{
		GroupID: groupID, Email: email, At: now,
	})
}

func CreateInvitationDelivery(ctx context.Context, q Querier, id, invitationID uuid.UUID, recipient string, generation int, at time.Time) error {
	return q.Queries().CreateInvitationDelivery(ctx, cestia.CreateInvitationDeliveryParams{
		ID: id, InvitationID: invitationID, Generation: int32(generation), Recipient: recipient, At: at,
	})
}

func CancelInvitationDeliveries(ctx context.Context, q Querier, invitationID uuid.UUID, at time.Time) error {
	return q.Queries().CancelInvitationDeliveries(ctx, cestia.CancelInvitationDeliveriesParams{
		InvitationID: invitationID, At: at,
	})
}
