-- name: ListGroupOptions :many
SELECT g.id, g.name, g.description
FROM groups g
LEFT JOIN group_members gm
  ON gm.group_id = g.id
 AND gm.user_id = sqlc.arg(user_id)::uuid
 AND gm.removed_at IS NULL
WHERE (g.owner_user_id = sqlc.arg(user_id)::uuid OR gm.user_id IS NOT NULL)
  AND (
    sqlc.arg(search)::text = ''
    OR g.name ILIKE ('%' || sqlc.arg(search)::text || '%')
  )
  AND (
    sqlc.narg(cursor_name)::text IS NULL
    OR (lower(g.name), g.id) > (lower(sqlc.narg(cursor_name)::text), sqlc.narg(cursor_id)::uuid)
  )
ORDER BY lower(g.name), g.id
LIMIT sqlc.arg(limit_count)::integer;

-- name: ListGroupMemberOptions :many
SELECT u.id AS user_id, u.display_name, u.email
FROM group_members gm
JOIN users u ON u.id = gm.user_id
WHERE gm.group_id = sqlc.arg(group_id)::uuid
  AND gm.removed_at IS NULL
  AND u.disabled_at IS NULL
  AND (
    sqlc.arg(search)::text = ''
    OR u.display_name ILIKE ('%' || sqlc.arg(search)::text || '%')
    OR u.email ILIKE ('%' || sqlc.arg(search)::text || '%')
  )
  AND (
    sqlc.narg(cursor_name)::text IS NULL
    OR (lower(u.display_name), u.id) > (lower(sqlc.narg(cursor_name)::text), sqlc.narg(cursor_id)::uuid)
  )
ORDER BY lower(u.display_name), u.id
LIMIT sqlc.arg(limit_count)::integer;
