-- name: GetGroupForUser :one
SELECT g.id, g.name, g.description, g.owner_user_id, g.created_by, g.archived_at,
       g.version, g.created_at, g.updated_at,
       COALESCE(CASE WHEN g.owner_user_id = sqlc.arg(user_id)::uuid THEN 'owner' ELSE gm.role END, '')::text AS role,
       (SELECT count(*) FROM group_members m WHERE m.group_id = g.id AND m.removed_at IS NULL) AS member_count,
       (SELECT count(*) FROM shopping_lists l WHERE l.group_id = g.id AND l.archived_at IS NULL) AS list_count
FROM groups g
LEFT JOIN group_members gm
  ON gm.group_id = g.id
 AND gm.user_id = sqlc.arg(user_id)::uuid
 AND gm.removed_at IS NULL
WHERE g.id = sqlc.arg(id)::uuid
  AND (g.owner_user_id = sqlc.arg(user_id)::uuid OR gm.user_id IS NOT NULL);

-- name: GetGroupForUserForUpdate :one
SELECT g.id, g.name, g.description, g.owner_user_id, g.created_by, g.archived_at,
       g.version, g.created_at, g.updated_at,
       COALESCE(CASE WHEN g.owner_user_id = sqlc.arg(user_id)::uuid THEN 'owner' ELSE gm.role END, '')::text AS role,
       (SELECT count(*) FROM group_members m WHERE m.group_id = g.id AND m.removed_at IS NULL) AS member_count,
       (SELECT count(*) FROM shopping_lists l WHERE l.group_id = g.id AND l.archived_at IS NULL) AS list_count
FROM groups g
LEFT JOIN group_members gm
  ON gm.group_id = g.id
 AND gm.user_id = sqlc.arg(user_id)::uuid
 AND gm.removed_at IS NULL
WHERE g.id = sqlc.arg(id)::uuid
  AND (g.owner_user_id = sqlc.arg(user_id)::uuid OR gm.user_id IS NOT NULL)
FOR UPDATE OF g;

-- name: LockGroupForShare :one
SELECT id FROM groups WHERE id = sqlc.arg(id)::uuid FOR SHARE;

-- name: GetGroupForUserForShare :one
SELECT g.id, g.name, g.description, g.owner_user_id, g.created_by, g.archived_at,
       g.version, g.created_at, g.updated_at,
       COALESCE(CASE WHEN g.owner_user_id = sqlc.arg(user_id)::uuid THEN 'owner' ELSE gm.role END, '')::text AS role,
       (SELECT count(*) FROM group_members m WHERE m.group_id = g.id AND m.removed_at IS NULL) AS member_count,
       (SELECT count(*) FROM shopping_lists l WHERE l.group_id = g.id AND l.archived_at IS NULL) AS list_count
FROM groups g
LEFT JOIN group_members gm
  ON gm.group_id = g.id
 AND gm.user_id = sqlc.arg(user_id)::uuid
 AND gm.removed_at IS NULL
WHERE g.id = sqlc.arg(id)::uuid
  AND (g.owner_user_id = sqlc.arg(user_id)::uuid OR gm.user_id IS NOT NULL)
FOR SHARE OF g;

-- name: GetGroupByIDForUpdate :one
SELECT g.id, g.name, g.description, g.owner_user_id, g.created_by, g.archived_at,
       g.version, g.created_at, g.updated_at, 'owner'::text AS role,
       (SELECT count(*) FROM group_members m WHERE m.group_id = g.id AND m.removed_at IS NULL) AS member_count,
       (SELECT count(*) FROM shopping_lists l WHERE l.group_id = g.id AND l.archived_at IS NULL) AS list_count
FROM groups g
WHERE g.id = sqlc.arg(id)::uuid
FOR UPDATE OF g;

-- name: ListGroupsForUser :many
SELECT g.id, g.name, g.description, g.owner_user_id, g.created_by, g.archived_at,
       g.version, g.created_at, g.updated_at,
       COALESCE(CASE WHEN g.owner_user_id = sqlc.arg(user_id)::uuid THEN 'owner' ELSE gm.role END, '')::text AS role,
       (SELECT count(*) FROM group_members m WHERE m.group_id = g.id AND m.removed_at IS NULL) AS member_count,
       (SELECT count(*) FROM shopping_lists l WHERE l.group_id = g.id AND l.archived_at IS NULL) AS list_count
FROM groups g
LEFT JOIN group_members gm
  ON gm.group_id = g.id
 AND gm.user_id = sqlc.arg(user_id)::uuid
 AND gm.removed_at IS NULL
WHERE g.owner_user_id = sqlc.arg(user_id)::uuid OR gm.user_id IS NOT NULL
ORDER BY g.archived_at NULLS FIRST, lower(g.name);

-- name: CreateGroup :exec
INSERT INTO groups (id, name, description, owner_user_id, created_by, version, created_at, updated_at)
VALUES (
  sqlc.arg(id)::uuid,
  sqlc.arg(name)::text,
  sqlc.arg(description)::text,
  sqlc.arg(owner_user_id)::uuid,
  sqlc.arg(owner_user_id)::uuid,
  1,
  sqlc.arg(created_at)::timestamptz,
  sqlc.arg(created_at)::timestamptz
);

-- name: UpsertGroupMember :exec
INSERT INTO group_members (group_id, user_id, role, joined_at, version)
VALUES (
  sqlc.arg(group_id)::uuid,
  sqlc.arg(user_id)::uuid,
  sqlc.arg(role)::text,
  sqlc.arg(joined_at)::timestamptz,
  1
)
ON CONFLICT (group_id, user_id) DO UPDATE
SET role = EXCLUDED.role,
    removed_at = NULL,
    joined_at = EXCLUDED.joined_at,
    version = group_members.version + 1;

-- name: ListGroupMembers :many
SELECT u.id AS user_id, u.display_name, u.email,
       COALESCE(CASE WHEN g.owner_user_id = u.id THEN 'owner' ELSE gm.role END, '')::text AS role,
       gm.joined_at, gm.removed_at
FROM group_members gm
JOIN users u ON u.id = gm.user_id
JOIN groups g ON g.id = gm.group_id
WHERE gm.group_id = sqlc.arg(group_id)::uuid
  AND gm.removed_at IS NULL
ORDER BY CASE WHEN g.owner_user_id = u.id THEN 0 WHEN gm.role = 'admin' THEN 1 ELSE 2 END,
         lower(u.display_name);

-- name: GetGroupMemberRole :one
SELECT COALESCE(CASE WHEN g.owner_user_id = sqlc.arg(user_id)::uuid THEN 'owner' ELSE gm.role END, '')::text AS role
FROM groups g
JOIN group_members gm
  ON gm.group_id = g.id
 AND gm.user_id = sqlc.arg(user_id)::uuid
 AND gm.removed_at IS NULL
WHERE g.id = sqlc.arg(group_id)::uuid;

-- name: ActiveGroupMemberEmailExists :one
SELECT EXISTS (
  SELECT 1
  FROM users u
  JOIN group_members gm ON gm.user_id = u.id
  WHERE lower(u.email) = lower(sqlc.arg(email)::text)
    AND gm.group_id = sqlc.arg(group_id)::uuid
    AND gm.removed_at IS NULL
);

-- name: RemoveGroupMember :exec
UPDATE group_members
SET removed_at = sqlc.arg(at)::timestamptz, version = version + 1
WHERE group_id = sqlc.arg(group_id)::uuid
  AND user_id = sqlc.arg(user_id)::uuid
  AND removed_at IS NULL;

-- name: SetGroupMemberRole :exec
UPDATE group_members
SET role = sqlc.arg(role)::text, version = version + 1
WHERE group_id = sqlc.arg(group_id)::uuid
  AND user_id = sqlc.arg(user_id)::uuid
  AND removed_at IS NULL;

-- name: UpdateGroup :execrows
UPDATE groups
SET name = sqlc.arg(name)::text,
    description = sqlc.arg(description)::text,
    version = version + 1,
    updated_at = sqlc.arg(at)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND version = sqlc.arg(version)::bigint
  AND archived_at IS NULL;

-- name: ArchiveGroup :execrows
UPDATE groups
SET archived_at = sqlc.arg(at)::timestamptz,
    version = version + 1,
    updated_at = sqlc.arg(at)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND version = sqlc.arg(version)::bigint;

-- name: RestoreGroup :execrows
UPDATE groups
SET archived_at = NULL,
    version = version + 1,
    updated_at = sqlc.arg(at)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND version = sqlc.arg(version)::bigint;

-- name: SetGroupOwner :exec
UPDATE groups
SET owner_user_id = sqlc.arg(new_owner_user_id)::uuid,
    version = version + 1,
    updated_at = sqlc.arg(at)::timestamptz
WHERE id = sqlc.arg(id)::uuid;

-- name: SetGroupMemberAdmin :exec
UPDATE group_members
SET role = 'admin', version = version + 1
WHERE group_id = sqlc.arg(group_id)::uuid
  AND user_id = sqlc.arg(user_id)::uuid;

-- name: CreateGroupInvitation :exec
INSERT INTO group_invitations (
  id, group_id, email, invited_by, status, token_hash, generation,
  expires_at, version, created_at, updated_at
) VALUES (
  sqlc.arg(id)::uuid,
  sqlc.arg(group_id)::uuid,
  sqlc.arg(email)::text,
  sqlc.arg(invited_by)::uuid,
  'pending',
  sqlc.arg(token_hash)::bytea,
  sqlc.arg(generation)::integer,
  sqlc.arg(expires_at)::timestamptz,
  1,
  sqlc.arg(created_at)::timestamptz,
  sqlc.arg(created_at)::timestamptz
);

-- name: UpdateInvitationToken :execrows
UPDATE group_invitations
SET token_hash = sqlc.arg(token_hash)::bytea,
    generation = sqlc.arg(generation)::integer,
    expires_at = sqlc.arg(expires_at)::timestamptz,
    status = 'pending',
    resolved_at = NULL,
    resolved_by = NULL,
    version = version + 1,
    updated_at = sqlc.arg(at)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND status IN ('pending', 'expired')
  AND version = sqlc.arg(version)::bigint;

-- name: ListInvitations :many
SELECT i.id, i.group_id, g.name AS group_name, i.email, i.invited_by, i.status,
       i.generation, i.expires_at, i.resolved_at, i.resolved_by, i.version,
       i.created_at, i.updated_at
FROM group_invitations i
JOIN groups g ON g.id = i.group_id
WHERE i.group_id = sqlc.arg(group_id)::uuid
ORDER BY i.created_at DESC;

-- name: GetInvitationByID :one
SELECT i.id, i.group_id, g.name AS group_name, i.email, i.invited_by, i.status,
       i.generation, i.expires_at, i.resolved_at, i.resolved_by, i.version,
       i.created_at, i.updated_at
FROM group_invitations i
JOIN groups g ON g.id = i.group_id
WHERE i.id = sqlc.arg(id)::uuid;

-- name: GetInvitationByIDForUpdate :one
SELECT i.id, i.group_id, g.name AS group_name, i.email, i.invited_by, i.status,
       i.generation, i.expires_at, i.resolved_at, i.resolved_by, i.version,
       i.created_at, i.updated_at
FROM group_invitations i
JOIN groups g ON g.id = i.group_id
WHERE i.id = sqlc.arg(id)::uuid
FOR UPDATE OF i;

-- name: GetInvitationByToken :one
SELECT i.id, i.group_id, g.name AS group_name, i.email, i.invited_by, i.status,
       i.generation, i.expires_at, i.resolved_at, i.resolved_by, i.version,
       i.created_at, i.updated_at
FROM group_invitations i
JOIN groups g ON g.id = i.group_id
WHERE i.token_hash = sqlc.arg(token_hash)::bytea;

-- name: GetInvitationByTokenForUpdate :one
SELECT i.id, i.group_id, g.name AS group_name, i.email, i.invited_by, i.status,
       i.generation, i.expires_at, i.resolved_at, i.resolved_by, i.version,
       i.created_at, i.updated_at
FROM group_invitations i
JOIN groups g ON g.id = i.group_id
WHERE i.token_hash = sqlc.arg(token_hash)::bytea
FOR UPDATE OF i;

-- name: ResolveInvitation :exec
UPDATE group_invitations
SET status = sqlc.arg(status)::text,
    resolved_at = sqlc.arg(at)::timestamptz,
    resolved_by = sqlc.arg(resolved_by)::uuid,
    version = version + 1,
    updated_at = sqlc.arg(at)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND status = 'pending';

-- name: CancelInvitation :exec
UPDATE group_invitations
SET status = 'canceled',
    resolved_at = sqlc.arg(at)::timestamptz,
    resolved_by = NULL,
    version = version + 1,
    updated_at = sqlc.arg(at)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND status = 'pending';

-- name: ExpirePendingInvitation :exec
UPDATE group_invitations
SET status = 'expired',
    resolved_at = sqlc.arg(at)::timestamptz,
    version = version + 1,
    updated_at = sqlc.arg(at)::timestamptz
WHERE group_id = sqlc.arg(group_id)::uuid
  AND lower(email) = lower(sqlc.arg(email)::text)
  AND status = 'pending'
  AND expires_at <= sqlc.arg(at)::timestamptz;

-- name: CreateInvitationDelivery :exec
INSERT INTO email_deliveries (
  id, invitation_id, generation, recipient, next_attempt_at, created_at, updated_at
) VALUES (
  sqlc.arg(id)::uuid,
  sqlc.arg(invitation_id)::uuid,
  sqlc.arg(generation)::integer,
  sqlc.arg(recipient)::text,
  sqlc.arg(at)::timestamptz,
  sqlc.arg(at)::timestamptz,
  sqlc.arg(at)::timestamptz
);

-- name: CancelInvitationDeliveries :exec
UPDATE email_deliveries
SET status = 'canceled', updated_at = sqlc.arg(at)::timestamptz
WHERE invitation_id = sqlc.arg(invitation_id)::uuid
  AND status IN ('queued', 'processing', 'failed');
