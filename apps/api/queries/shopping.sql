-- name: GetListForUser :one
SELECT l.id, l.name, l.description, l.personal_owner_id, l.group_id, l.created_by,
       l.archived_at, l.version, l.created_at, l.updated_at,
       COALESCE(CASE
         WHEN l.personal_owner_id = sqlc.arg(user_id)::uuid THEN 'owner'
         WHEN g.owner_user_id = sqlc.arg(user_id)::uuid THEN 'owner'
         ELSE gm.role
       END, '')::text AS role,
       (SELECT count(*)
        FROM shopping_list_items i
        LEFT JOIN purchases p ON p.item_id = i.id AND p.reversed_at IS NULL
        WHERE i.list_id = l.id AND i.deleted_at IS NULL AND p.id IS NULL) AS pending_count,
       (SELECT count(*)
        FROM shopping_list_items i
        JOIN purchases p ON p.item_id = i.id AND p.reversed_at IS NULL
        WHERE i.list_id = l.id AND i.deleted_at IS NULL) AS purchased_count
FROM shopping_lists l
LEFT JOIN groups g ON g.id = l.group_id
LEFT JOIN group_members gm
  ON gm.group_id = l.group_id
 AND gm.user_id = sqlc.arg(user_id)::uuid
 AND gm.removed_at IS NULL
WHERE l.id = sqlc.arg(id)::uuid
  AND (l.personal_owner_id = sqlc.arg(user_id)::uuid
    OR g.owner_user_id = sqlc.arg(user_id)::uuid
    OR gm.user_id IS NOT NULL);

-- name: GetListForUserForUpdate :one
SELECT l.id, l.name, l.description, l.personal_owner_id, l.group_id, l.created_by,
       l.archived_at, l.version, l.created_at, l.updated_at,
       COALESCE(CASE
         WHEN l.personal_owner_id = sqlc.arg(user_id)::uuid THEN 'owner'
         WHEN g.owner_user_id = sqlc.arg(user_id)::uuid THEN 'owner'
         ELSE gm.role
       END, '')::text AS role,
       (SELECT count(*)
        FROM shopping_list_items i
        LEFT JOIN purchases p ON p.item_id = i.id AND p.reversed_at IS NULL
        WHERE i.list_id = l.id AND i.deleted_at IS NULL AND p.id IS NULL) AS pending_count,
       (SELECT count(*)
        FROM shopping_list_items i
        JOIN purchases p ON p.item_id = i.id AND p.reversed_at IS NULL
        WHERE i.list_id = l.id AND i.deleted_at IS NULL) AS purchased_count
FROM shopping_lists l
LEFT JOIN groups g ON g.id = l.group_id
LEFT JOIN group_members gm
  ON gm.group_id = l.group_id
 AND gm.user_id = sqlc.arg(user_id)::uuid
 AND gm.removed_at IS NULL
WHERE l.id = sqlc.arg(id)::uuid
  AND (l.personal_owner_id = sqlc.arg(user_id)::uuid
    OR g.owner_user_id = sqlc.arg(user_id)::uuid
    OR gm.user_id IS NOT NULL)
FOR UPDATE OF l;

-- name: GetListForUserForShare :one
SELECT l.id, l.name, l.description, l.personal_owner_id, l.group_id, l.created_by,
       l.archived_at, l.version, l.created_at, l.updated_at,
       COALESCE(CASE
         WHEN l.personal_owner_id = sqlc.arg(user_id)::uuid THEN 'owner'
         WHEN g.owner_user_id = sqlc.arg(user_id)::uuid THEN 'owner'
         ELSE gm.role
       END, '')::text AS role,
       (SELECT count(*)
        FROM shopping_list_items i
        LEFT JOIN purchases p ON p.item_id = i.id AND p.reversed_at IS NULL
        WHERE i.list_id = l.id AND i.deleted_at IS NULL AND p.id IS NULL) AS pending_count,
       (SELECT count(*)
        FROM shopping_list_items i
        JOIN purchases p ON p.item_id = i.id AND p.reversed_at IS NULL
        WHERE i.list_id = l.id AND i.deleted_at IS NULL) AS purchased_count
FROM shopping_lists l
LEFT JOIN groups g ON g.id = l.group_id
LEFT JOIN group_members gm
  ON gm.group_id = l.group_id
 AND gm.user_id = sqlc.arg(user_id)::uuid
 AND gm.removed_at IS NULL
WHERE l.id = sqlc.arg(id)::uuid
  AND (l.personal_owner_id = sqlc.arg(user_id)::uuid
    OR g.owner_user_id = sqlc.arg(user_id)::uuid
    OR gm.user_id IS NOT NULL)
FOR SHARE OF l;

-- name: ListPersonalLists :many
SELECT l.id, l.name, l.description, l.personal_owner_id, l.group_id, l.created_by,
       l.archived_at, l.version, l.created_at, l.updated_at,
       'owner'::text AS role,
       (SELECT count(*)
        FROM shopping_list_items i
        LEFT JOIN purchases p ON p.item_id = i.id AND p.reversed_at IS NULL
        WHERE i.list_id = l.id AND i.deleted_at IS NULL AND p.id IS NULL) AS pending_count,
       (SELECT count(*)
        FROM shopping_list_items i
        JOIN purchases p ON p.item_id = i.id AND p.reversed_at IS NULL
        WHERE i.list_id = l.id AND i.deleted_at IS NULL) AS purchased_count
FROM shopping_lists l
WHERE l.personal_owner_id = sqlc.arg(user_id)::uuid
  AND (sqlc.arg(include_archived)::boolean = true OR l.archived_at IS NULL)
ORDER BY l.archived_at NULLS FIRST, l.updated_at DESC;

-- name: ListGroupLists :many
SELECT l.id, l.name, l.description, l.personal_owner_id, l.group_id, l.created_by,
       l.archived_at, l.version, l.created_at, l.updated_at,
       COALESCE(CASE WHEN g.owner_user_id = sqlc.arg(user_id)::uuid THEN 'owner' ELSE gm.role END, '')::text AS role,
       (SELECT count(*)
        FROM shopping_list_items i
        LEFT JOIN purchases p ON p.item_id = i.id AND p.reversed_at IS NULL
        WHERE i.list_id = l.id AND i.deleted_at IS NULL AND p.id IS NULL) AS pending_count,
       (SELECT count(*)
        FROM shopping_list_items i
        JOIN purchases p ON p.item_id = i.id AND p.reversed_at IS NULL
        WHERE i.list_id = l.id AND i.deleted_at IS NULL) AS purchased_count
FROM shopping_lists l
JOIN groups g ON g.id = l.group_id
LEFT JOIN group_members gm
  ON gm.group_id = l.group_id
 AND gm.user_id = sqlc.arg(user_id)::uuid
 AND gm.removed_at IS NULL
WHERE l.group_id = sqlc.arg(group_id)::uuid
  AND (g.owner_user_id = sqlc.arg(user_id)::uuid OR gm.user_id IS NOT NULL)
  AND (sqlc.arg(include_archived)::boolean = true OR l.archived_at IS NULL)
ORDER BY l.archived_at NULLS FIRST, l.updated_at DESC;

-- name: CreateList :exec
INSERT INTO shopping_lists (
  id, name, description, personal_owner_id, group_id, created_by,
  version, created_at, updated_at
) VALUES (
  sqlc.arg(id)::uuid,
  sqlc.arg(name)::text,
  sqlc.arg(description)::text,
  sqlc.narg(personal_owner_id)::uuid,
  sqlc.narg(group_id)::uuid,
  sqlc.arg(created_by)::uuid,
  1,
  sqlc.arg(created_at)::timestamptz,
  sqlc.arg(created_at)::timestamptz
);

-- name: UpdateList :execrows
UPDATE shopping_lists
SET name = sqlc.arg(name)::text,
    description = sqlc.arg(description)::text,
    version = version + 1,
    updated_at = sqlc.arg(at)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND version = sqlc.arg(version)::bigint
  AND archived_at IS NULL;

-- name: ArchiveList :execrows
UPDATE shopping_lists
SET archived_at = sqlc.arg(at)::timestamptz,
    version = version + 1,
    updated_at = sqlc.arg(at)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND version = sqlc.arg(version)::bigint;

-- name: RestoreList :execrows
UPDATE shopping_lists
SET archived_at = NULL,
    version = version + 1,
    updated_at = sqlc.arg(at)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND version = sqlc.arg(version)::bigint;

-- name: TouchList :exec
UPDATE shopping_lists
SET version = version + 1, updated_at = sqlc.arg(at)::timestamptz
WHERE id = sqlc.arg(id)::uuid;

-- name: ListItems :many
SELECT i.id, i.list_id, i.name, i.description, i.quantity AS quantity, i.unit,
       i.estimated_unit_price AS estimated_unit_price,
       i.created_by, i.assigned_to, i.notes, i.deleted_at, i.version, i.created_at, i.updated_at,
       u.display_name AS assigned_to_name
FROM shopping_list_items i
LEFT JOIN users u ON u.id = i.assigned_to
WHERE i.list_id = sqlc.arg(list_id)::uuid
  AND i.deleted_at IS NULL
ORDER BY i.created_at, i.id;

-- name: GetItem :one
SELECT i.id, i.list_id, i.name, i.description, i.quantity AS quantity, i.unit,
       i.estimated_unit_price AS estimated_unit_price,
       i.created_by, i.assigned_to, i.notes, i.deleted_at, i.version, i.created_at, i.updated_at,
       u.display_name AS assigned_to_name
FROM shopping_list_items i
LEFT JOIN users u ON u.id = i.assigned_to
WHERE i.id = sqlc.arg(id)::uuid
  AND i.deleted_at IS NULL;

-- name: GetItemForUpdate :one
SELECT i.id, i.list_id, i.name, i.description, i.quantity AS quantity, i.unit,
       i.estimated_unit_price AS estimated_unit_price,
       i.created_by, i.assigned_to, i.notes, i.deleted_at, i.version, i.created_at, i.updated_at,
       u.display_name AS assigned_to_name
FROM shopping_list_items i
LEFT JOIN users u ON u.id = i.assigned_to
WHERE i.id = sqlc.arg(id)::uuid
  AND i.deleted_at IS NULL
FOR UPDATE OF i;

-- name: GetItemForUpdateAny :one
SELECT i.id, i.list_id, i.name, i.description, i.quantity AS quantity, i.unit,
       i.estimated_unit_price AS estimated_unit_price,
       i.created_by, i.assigned_to, i.notes, i.deleted_at, i.version, i.created_at, i.updated_at,
       u.display_name AS assigned_to_name
FROM shopping_list_items i
LEFT JOIN users u ON u.id = i.assigned_to
WHERE i.id = sqlc.arg(id)::uuid
FOR UPDATE OF i;

-- name: CreateItem :exec
INSERT INTO shopping_list_items (
  id, list_id, name, description, quantity, unit, estimated_unit_price,
  created_by, assigned_to, notes, version, created_at, updated_at
) VALUES (
  sqlc.arg(id)::uuid,
  sqlc.arg(list_id)::uuid,
  sqlc.arg(name)::text,
  sqlc.arg(description)::text,
  (sqlc.arg(quantity)::text)::numeric,
  sqlc.arg(unit)::text,
  (sqlc.narg(estimated_unit_price)::text)::numeric,
  sqlc.arg(created_by)::uuid,
  sqlc.narg(assigned_to)::uuid,
  sqlc.arg(notes)::text,
  1,
  sqlc.arg(created_at)::timestamptz,
  sqlc.arg(created_at)::timestamptz
);

-- name: UpdateItem :execrows
UPDATE shopping_list_items
SET name = sqlc.arg(name)::text,
    description = sqlc.arg(description)::text,
    quantity = (sqlc.arg(quantity)::text)::numeric,
    unit = sqlc.arg(unit)::text,
    estimated_unit_price = (sqlc.narg(estimated_unit_price)::text)::numeric,
    assigned_to = sqlc.narg(assigned_to)::uuid,
    notes = sqlc.arg(notes)::text,
    version = version + 1,
    updated_at = sqlc.arg(at)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND version = sqlc.arg(version)::bigint
  AND deleted_at IS NULL;

-- name: DeleteItem :execrows
UPDATE shopping_list_items
SET deleted_at = sqlc.arg(at)::timestamptz,
    version = version + 1,
    updated_at = sqlc.arg(at)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND version = sqlc.arg(version)::bigint
  AND deleted_at IS NULL;

-- name: TouchItem :exec
UPDATE shopping_list_items
SET version = version + 1, updated_at = sqlc.arg(at)::timestamptz
WHERE id = sqlc.arg(id)::uuid;

-- name: GetActivePurchaseForItem :one
SELECT p.id, p.item_id, l.id AS list_id, l.group_id, p.item_name,
       p.quantity AS quantity, p.unit,
       p.actual_unit_price AS actual_unit_price,
       p.total_amount AS total_amount,
       p.currency, p.recorded_by, p.purchased_by, p.paid_by, p.purchased_at,
       p.notes, p.reversed_by, p.reversed_at, p.reversal_reason, p.version,
       p.created_at, p.updated_at,
       u1.display_name AS recorded_by_name,
       u2.display_name AS purchased_by_name,
       COALESCE(u3.display_name, '')::text AS paid_by_name
FROM purchases p
JOIN shopping_list_items i ON i.id = p.item_id
JOIN shopping_lists l ON l.id = i.list_id
JOIN users u1 ON u1.id = p.recorded_by
JOIN users u2 ON u2.id = p.purchased_by
LEFT JOIN users u3 ON u3.id = p.paid_by
WHERE p.item_id = sqlc.arg(item_id)::uuid
  AND p.reversed_at IS NULL;

-- name: GetPurchaseForUser :one
SELECT p.id, p.item_id, l.id AS list_id, l.group_id, p.item_name,
       p.quantity AS quantity, p.unit,
       p.actual_unit_price AS actual_unit_price,
       p.total_amount AS total_amount,
       p.currency, p.recorded_by, p.purchased_by, p.paid_by, p.purchased_at,
       p.notes, p.reversed_by, p.reversed_at, p.reversal_reason, p.version,
       p.created_at, p.updated_at,
       u1.display_name AS recorded_by_name,
       u2.display_name AS purchased_by_name,
       COALESCE(u3.display_name, '')::text AS paid_by_name
FROM purchases p
JOIN shopping_list_items i ON i.id = p.item_id
JOIN shopping_lists l ON l.id = i.list_id
JOIN users u1 ON u1.id = p.recorded_by
JOIN users u2 ON u2.id = p.purchased_by
LEFT JOIN users u3 ON u3.id = p.paid_by
WHERE p.id = sqlc.arg(id)::uuid
  AND (l.personal_owner_id = sqlc.arg(user_id)::uuid
    OR EXISTS (
      SELECT 1
      FROM groups g
      WHERE g.id = l.group_id
        AND (g.owner_user_id = sqlc.arg(user_id)::uuid
          OR EXISTS (
            SELECT 1 FROM group_members gm
            WHERE gm.group_id = g.id
              AND gm.user_id = sqlc.arg(user_id)::uuid
              AND gm.removed_at IS NULL
          ))
    ));

-- name: GetPurchaseForUserForUpdate :one
SELECT p.id, p.item_id, l.id AS list_id, l.group_id, p.item_name,
       p.quantity AS quantity, p.unit,
       p.actual_unit_price AS actual_unit_price,
       p.total_amount AS total_amount,
       p.currency, p.recorded_by, p.purchased_by, p.paid_by, p.purchased_at,
       p.notes, p.reversed_by, p.reversed_at, p.reversal_reason, p.version,
       p.created_at, p.updated_at,
       u1.display_name AS recorded_by_name,
       u2.display_name AS purchased_by_name,
       COALESCE(u3.display_name, '')::text AS paid_by_name
FROM purchases p
JOIN shopping_list_items i ON i.id = p.item_id
JOIN shopping_lists l ON l.id = i.list_id
JOIN users u1 ON u1.id = p.recorded_by
JOIN users u2 ON u2.id = p.purchased_by
LEFT JOIN users u3 ON u3.id = p.paid_by
WHERE p.id = sqlc.arg(id)::uuid
  AND (l.personal_owner_id = sqlc.arg(user_id)::uuid
    OR EXISTS (
      SELECT 1
      FROM groups g
      WHERE g.id = l.group_id
        AND (g.owner_user_id = sqlc.arg(user_id)::uuid
          OR EXISTS (
            SELECT 1 FROM group_members gm
            WHERE gm.group_id = g.id
              AND gm.user_id = sqlc.arg(user_id)::uuid
              AND gm.removed_at IS NULL
          ))
    ))
FOR UPDATE OF p;

-- name: CreatePurchase :exec
INSERT INTO purchases (
  id, item_id, item_name, quantity, unit, actual_unit_price, currency,
  recorded_by, purchased_by, paid_by, purchased_at, notes, version, created_at, updated_at
) VALUES (
  sqlc.arg(id)::uuid,
  sqlc.arg(item_id)::uuid,
  sqlc.arg(item_name)::text,
  (sqlc.arg(quantity)::text)::numeric,
  sqlc.arg(unit)::text,
  (sqlc.narg(actual_unit_price)::text)::numeric,
  'MXN',
  sqlc.arg(recorded_by)::uuid,
  sqlc.arg(purchased_by)::uuid,
  sqlc.narg(paid_by)::uuid,
  sqlc.arg(purchased_at)::timestamptz,
  sqlc.arg(notes)::text,
  1,
  sqlc.arg(created_at)::timestamptz,
  sqlc.arg(created_at)::timestamptz
);

-- name: UpdatePurchase :execrows
UPDATE purchases
SET actual_unit_price = (sqlc.narg(actual_unit_price)::text)::numeric,
    purchased_by = sqlc.arg(purchased_by)::uuid,
    paid_by = sqlc.narg(paid_by)::uuid,
    purchased_at = sqlc.arg(purchased_at)::timestamptz,
    notes = sqlc.arg(notes)::text,
    version = version + 1,
    updated_at = sqlc.arg(at)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND version = sqlc.arg(version)::bigint
  AND reversed_at IS NULL;

-- name: ReversePurchase :execrows
UPDATE purchases
SET reversed_at = sqlc.arg(at)::timestamptz,
    reversed_by = sqlc.arg(actor_id)::uuid,
    reversal_reason = sqlc.arg(reason)::text,
    version = version + 1,
    updated_at = sqlc.arg(at)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND version = sqlc.arg(version)::bigint
  AND reversed_at IS NULL;

-- name: ListPurchaseHistory :many
SELECT p.id, p.item_id, l.id AS list_id, l.group_id, p.item_name,
       p.quantity AS quantity, p.unit,
       p.actual_unit_price AS actual_unit_price,
       p.total_amount AS total_amount,
       p.currency, p.recorded_by, p.purchased_by, p.paid_by, p.purchased_at,
       p.notes, p.reversed_by, p.reversed_at, p.reversal_reason, p.version,
       p.created_at, p.updated_at,
       u1.display_name AS recorded_by_name,
       u2.display_name AS purchased_by_name,
       COALESCE(u3.display_name, '')::text AS paid_by_name
FROM purchases p
JOIN shopping_list_items i ON i.id = p.item_id
JOIN shopping_lists l ON l.id = i.list_id
JOIN users u1 ON u1.id = p.recorded_by
JOIN users u2 ON u2.id = p.purchased_by
LEFT JOIN users u3 ON u3.id = p.paid_by
WHERE ((sqlc.narg(personal_owner_id)::uuid IS NOT NULL AND l.personal_owner_id = sqlc.narg(personal_owner_id)::uuid)
    OR (sqlc.narg(group_id)::uuid IS NOT NULL AND l.group_id = sqlc.narg(group_id)::uuid))
  AND (sqlc.narg(from_at)::timestamptz IS NULL OR p.purchased_at >= sqlc.narg(from_at)::timestamptz)
  AND (sqlc.narg(to_at)::timestamptz IS NULL OR p.purchased_at < sqlc.narg(to_at)::timestamptz)
  AND (sqlc.narg(list_id)::uuid IS NULL OR l.id = sqlc.narg(list_id)::uuid)
  AND (sqlc.narg(paid_by)::uuid IS NULL OR p.paid_by = sqlc.narg(paid_by)::uuid)
  AND (sqlc.narg(purchased_by)::uuid IS NULL OR p.purchased_by = sqlc.narg(purchased_by)::uuid)
  AND (sqlc.narg(cursor_at)::timestamptz IS NULL
    OR (p.purchased_at, p.id) < (sqlc.narg(cursor_at)::timestamptz, sqlc.narg(cursor_id)::uuid))
ORDER BY p.purchased_at DESC, p.id DESC
LIMIT sqlc.arg(limit_count)::integer;

-- name: ListPersonalActivity :many
SELECT a.id, a.scope_type, a.actor_id, u.display_name AS actor_name, a.group_id,
       a.list_id, a.item_id, a.purchase_id, a.action, a.payload, a.created_at
FROM activity_events a
LEFT JOIN users u ON u.id = a.actor_id
WHERE a.scope_type = 'personal'
  AND (a.actor_id = sqlc.arg(user_id)::uuid
    OR a.list_id IN (SELECT id FROM shopping_lists WHERE personal_owner_id = sqlc.arg(user_id)::uuid))
ORDER BY a.created_at DESC, a.id DESC
LIMIT sqlc.arg(limit_count)::integer;

-- name: ListGroupActivity :many
SELECT a.id, a.scope_type, a.actor_id, u.display_name AS actor_name, a.group_id,
       a.list_id, a.item_id, a.purchase_id, a.action, a.payload, a.created_at
FROM activity_events a
LEFT JOIN users u ON u.id = a.actor_id
WHERE a.scope_type = 'group'
  AND a.group_id = sqlc.arg(group_id)::uuid
ORDER BY a.created_at DESC, a.id DESC
LIMIT sqlc.arg(limit_count)::integer;

-- name: SummaryItems :one
SELECT
  COALESCE(SUM(CASE WHEN i.estimated_unit_price IS NOT NULL THEN round(i.quantity * i.estimated_unit_price, 2) ELSE 0 END), 0)::text AS estimated_known,
  COUNT(*) FILTER (WHERE i.estimated_unit_price IS NULL) AS estimated_unknown_items,
  COALESCE(SUM(CASE WHEN p.id IS NULL AND i.estimated_unit_price IS NOT NULL THEN round(i.quantity * i.estimated_unit_price, 2) ELSE 0 END), 0)::text AS pending_estimated,
  COUNT(*) FILTER (WHERE p.id IS NULL) AS pending_count
FROM shopping_list_items i
JOIN shopping_lists l ON l.id = i.list_id
LEFT JOIN purchases p ON p.item_id = i.id AND p.reversed_at IS NULL
WHERE i.deleted_at IS NULL
  AND ((sqlc.narg(personal_owner_id)::uuid IS NOT NULL AND l.personal_owner_id = sqlc.narg(personal_owner_id)::uuid)
    OR (sqlc.narg(group_id)::uuid IS NOT NULL AND l.group_id = sqlc.narg(group_id)::uuid))
  AND (sqlc.arg(include_archived)::boolean = true OR l.archived_at IS NULL);

-- name: SummaryPurchases :one
SELECT
  COALESCE(SUM(COALESCE(p.total_amount, 0)), 0)::text AS actual_known,
  COUNT(*) FILTER (WHERE p.actual_unit_price IS NULL) AS purchases_without_price,
  COUNT(p.id) AS purchase_count,
  COUNT(*) FILTER (WHERE p.id IS NOT NULL) AS purchased_count
FROM purchases p
JOIN shopping_list_items i ON i.id = p.item_id
JOIN shopping_lists l ON l.id = i.list_id
WHERE p.reversed_at IS NULL
  AND ((sqlc.narg(personal_owner_id)::uuid IS NOT NULL AND l.personal_owner_id = sqlc.narg(personal_owner_id)::uuid)
    OR (sqlc.narg(group_id)::uuid IS NOT NULL AND l.group_id = sqlc.narg(group_id)::uuid))
  AND (sqlc.arg(include_archived)::boolean = true OR l.archived_at IS NULL)
  AND (sqlc.narg(from_at)::timestamptz IS NULL OR p.purchased_at >= sqlc.narg(from_at)::timestamptz)
  AND (sqlc.narg(to_at)::timestamptz IS NULL OR p.purchased_at < sqlc.narg(to_at)::timestamptz);

-- name: SummaryPayers :many
SELECT p.paid_by, COALESCE(u.display_name, 'No identificado')::text AS display_name,
       COALESCE(SUM(p.total_amount), 0)::text AS amount, COUNT(p.id) AS purchase_count
FROM purchases p
JOIN shopping_list_items i ON i.id = p.item_id
JOIN shopping_lists l ON l.id = i.list_id
LEFT JOIN users u ON u.id = p.paid_by
WHERE p.reversed_at IS NULL
  AND ((sqlc.narg(personal_owner_id)::uuid IS NOT NULL AND l.personal_owner_id = sqlc.narg(personal_owner_id)::uuid)
    OR (sqlc.narg(group_id)::uuid IS NOT NULL AND l.group_id = sqlc.narg(group_id)::uuid))
  AND (sqlc.arg(include_archived)::boolean = true OR l.archived_at IS NULL)
  AND (sqlc.narg(from_at)::timestamptz IS NULL OR p.purchased_at >= sqlc.narg(from_at)::timestamptz)
  AND (sqlc.narg(to_at)::timestamptz IS NULL OR p.purchased_at < sqlc.narg(to_at)::timestamptz)
GROUP BY p.paid_by, u.display_name
ORDER BY SUM(p.total_amount) DESC NULLS LAST;

-- name: ExpensesByList :many
SELECT l.id AS list_id, l.name, l.archived_at,
       COALESCE(actual.amount, 0)::text AS actual_known,
       COALESCE(estimated.amount, 0)::text AS estimated_known,
       COALESCE(estimated.pending, 0)::text AS pending_estimated,
       COALESCE(actual.purchase_count, 0) AS purchase_count,
       COALESCE(estimated.pending_count, 0) AS pending_count
FROM shopping_lists l
LEFT JOIN LATERAL (
  SELECT SUM(COALESCE(p.total_amount, 0)) AS amount, COUNT(p.id) AS purchase_count
  FROM purchases p
  JOIN shopping_list_items i ON i.id = p.item_id
  WHERE i.list_id = l.id AND p.reversed_at IS NULL
    AND (sqlc.narg(from_at)::timestamptz IS NULL OR p.purchased_at >= sqlc.narg(from_at)::timestamptz)
    AND (sqlc.narg(to_at)::timestamptz IS NULL OR p.purchased_at < sqlc.narg(to_at)::timestamptz)
) actual ON TRUE
LEFT JOIN LATERAL (
  SELECT SUM(CASE WHEN i.estimated_unit_price IS NOT NULL THEN round(i.quantity * i.estimated_unit_price, 2) ELSE 0 END) AS amount,
         SUM(CASE WHEN p.id IS NULL AND i.estimated_unit_price IS NOT NULL THEN round(i.quantity * i.estimated_unit_price, 2) ELSE 0 END) AS pending,
         COUNT(*) FILTER (WHERE p.id IS NULL) AS pending_count
  FROM shopping_list_items i
  LEFT JOIN purchases p ON p.item_id = i.id AND p.reversed_at IS NULL
  WHERE i.list_id = l.id AND i.deleted_at IS NULL
) estimated ON TRUE
WHERE ((sqlc.narg(personal_owner_id)::uuid IS NOT NULL AND l.personal_owner_id = sqlc.narg(personal_owner_id)::uuid)
    OR (sqlc.narg(group_id)::uuid IS NOT NULL AND l.group_id = sqlc.narg(group_id)::uuid))
  AND (sqlc.arg(include_archived)::boolean = true OR l.archived_at IS NULL)
ORDER BY l.archived_at NULLS FIRST, l.updated_at DESC;
