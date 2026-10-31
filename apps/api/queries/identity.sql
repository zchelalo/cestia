-- name: CreateUser :exec
INSERT INTO users (
  id, email, display_name, password_hash, email_verified_at, locale, timezone, created_at, updated_at
) VALUES (
  sqlc.arg(id)::uuid,
  sqlc.arg(email)::text,
  sqlc.arg(display_name)::text,
  sqlc.arg(password_hash)::text,
  sqlc.narg(email_verified_at)::timestamptz,
  sqlc.arg(locale)::text,
  sqlc.arg(timezone)::text,
  sqlc.arg(created_at)::timestamptz,
  sqlc.arg(created_at)::timestamptz
);

-- name: GetUserByEmail :one
SELECT id, email, display_name, password_hash, email_verified_at, locale, timezone,
       disabled_at, created_at, updated_at
FROM users
WHERE lower(email) = lower(sqlc.arg(email)::text);

-- name: GetUserByID :one
SELECT id, email, display_name, password_hash, email_verified_at, locale, timezone,
       disabled_at, created_at, updated_at
FROM users
WHERE id = sqlc.arg(id)::uuid;

-- name: GetUserByIDForUpdate :one
SELECT id, email, display_name, password_hash, email_verified_at, locale, timezone,
       disabled_at, created_at, updated_at
FROM users
WHERE id = sqlc.arg(id)::uuid
FOR UPDATE;

-- name: GetUserByIDForShare :one
SELECT id, email, display_name, password_hash, email_verified_at, locale, timezone,
       disabled_at, created_at, updated_at
FROM users
WHERE id = sqlc.arg(id)::uuid
FOR SHARE;

-- name: UpdateUserProfile :one
UPDATE users
SET display_name = sqlc.arg(display_name)::text,
    locale = sqlc.arg(locale)::text,
    timezone = sqlc.arg(timezone)::text,
    updated_at = sqlc.arg(updated_at)::timestamptz
WHERE id = sqlc.arg(id)::uuid
RETURNING id, email, display_name, password_hash, email_verified_at, locale, timezone,
          disabled_at, created_at, updated_at;

-- name: MarkUserEmailVerified :exec
UPDATE users
SET email_verified_at = COALESCE(email_verified_at, sqlc.arg(at)::timestamptz),
    updated_at = sqlc.arg(at)::timestamptz
WHERE id = sqlc.arg(id)::uuid;

-- name: SetUserPassword :exec
UPDATE users
SET password_hash = sqlc.arg(password_hash)::text,
    updated_at = sqlc.arg(at)::timestamptz
WHERE id = sqlc.arg(id)::uuid;

-- name: DeactivateUser :exec
UPDATE users
SET disabled_at = sqlc.arg(at)::timestamptz,
    updated_at = sqlc.arg(at)::timestamptz
WHERE id = sqlc.arg(id)::uuid;

-- name: UserExists :one
SELECT EXISTS (SELECT 1 FROM users WHERE id = sqlc.arg(id)::uuid);

-- name: CountOwnedGroups :one
SELECT count(*) FROM groups WHERE owner_user_id = sqlc.arg(user_id)::uuid;

-- name: CreateSession :exec
INSERT INTO sessions (id, user_id, token_hash, csrf_hash, expires_at, created_at)
VALUES (
  sqlc.arg(id)::uuid,
  sqlc.arg(user_id)::uuid,
  sqlc.arg(token_hash)::bytea,
  sqlc.arg(csrf_hash)::bytea,
  sqlc.arg(expires_at)::timestamptz,
  sqlc.arg(created_at)::timestamptz
);

-- name: GetSessionByToken :one
SELECT id, user_id, token_hash, csrf_hash, expires_at, revoked_at, created_at
FROM sessions
WHERE token_hash = sqlc.arg(token_hash)::bytea
  AND revoked_at IS NULL
  AND expires_at > sqlc.arg(now)::timestamptz;

-- name: RevokeSession :execrows
UPDATE sessions
SET revoked_at = sqlc.arg(at)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND user_id = sqlc.arg(user_id)::uuid
  AND revoked_at IS NULL;

-- name: RevokeAllSessions :exec
UPDATE sessions
SET revoked_at = sqlc.arg(at)::timestamptz
WHERE user_id = sqlc.arg(user_id)::uuid
  AND revoked_at IS NULL;

-- name: ListActiveSessions :many
SELECT id, user_id, token_hash, csrf_hash, expires_at, revoked_at, created_at
FROM sessions
WHERE user_id = sqlc.arg(user_id)::uuid
  AND revoked_at IS NULL
  AND expires_at > sqlc.arg(now)::timestamptz
ORDER BY created_at DESC;

-- name: CreateEmailAction :exec
INSERT INTO email_actions (id, user_id, purpose, token_hash, generation, expires_at, created_at)
VALUES (
  sqlc.arg(id)::uuid,
  sqlc.arg(user_id)::uuid,
  sqlc.arg(purpose)::text,
  sqlc.arg(token_hash)::bytea,
  sqlc.arg(generation)::integer,
  sqlc.arg(expires_at)::timestamptz,
  sqlc.arg(created_at)::timestamptz
);

-- name: GetEmailActionByToken :one
SELECT id, user_id, purpose, token_hash, generation, expires_at, consumed_at
FROM email_actions
WHERE token_hash = sqlc.arg(token_hash)::bytea
  AND purpose = sqlc.arg(purpose)::text
  AND consumed_at IS NULL
  AND expires_at > sqlc.arg(now)::timestamptz;

-- name: GetEmailActionByTokenForUpdate :one
SELECT id, user_id, purpose, token_hash, generation, expires_at, consumed_at
FROM email_actions
WHERE token_hash = sqlc.arg(token_hash)::bytea
  AND purpose = sqlc.arg(purpose)::text
  AND consumed_at IS NULL
  AND expires_at > sqlc.arg(now)::timestamptz
FOR UPDATE;

-- name: GetEmailActionByID :one
SELECT id, user_id, purpose, token_hash, generation, expires_at, consumed_at
FROM email_actions
WHERE id = sqlc.arg(id)::uuid;

-- name: ConsumeEmailAction :exec
UPDATE email_actions
SET consumed_at = sqlc.arg(at)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND consumed_at IS NULL;

-- name: CancelEmailDeliveriesForUserPurpose :exec
UPDATE email_deliveries
SET status = 'canceled', updated_at = sqlc.arg(at)::timestamptz
WHERE action_id IN (
  SELECT id FROM email_actions
  WHERE user_id = sqlc.arg(user_id)::uuid
    AND purpose = sqlc.arg(purpose)::text
    AND consumed_at IS NULL
);

-- name: ConsumeEmailActionsForUserPurpose :exec
UPDATE email_actions
SET consumed_at = sqlc.arg(at)::timestamptz
WHERE user_id = sqlc.arg(user_id)::uuid
  AND purpose = sqlc.arg(purpose)::text
  AND consumed_at IS NULL;

-- name: CreateEmailDeliveryForAction :exec
INSERT INTO email_deliveries (
  id, action_id, generation, recipient, next_attempt_at, created_at, updated_at
) VALUES (
  sqlc.arg(id)::uuid,
  sqlc.arg(action_id)::uuid,
  sqlc.arg(generation)::integer,
  sqlc.arg(recipient)::text,
  sqlc.arg(at)::timestamptz,
  sqlc.arg(at)::timestamptz,
  sqlc.arg(at)::timestamptz
);

-- name: InsertActivity :exec
INSERT INTO activity_events (
  id, scope_type, actor_id, group_id, list_id, item_id, purchase_id,
  action, payload_version, payload, created_at
) VALUES (
  sqlc.arg(id)::uuid,
  sqlc.arg(scope_type)::text,
  sqlc.narg(actor_id)::uuid,
  sqlc.narg(group_id)::uuid,
  sqlc.narg(list_id)::uuid,
  sqlc.narg(item_id)::uuid,
  sqlc.narg(purchase_id)::uuid,
  sqlc.arg(action)::text,
  1,
  sqlc.arg(payload)::jsonb,
  sqlc.arg(created_at)::timestamptz
);

-- name: IncrementRateLimit :one
INSERT INTO rate_limit_buckets (purpose, subject_hash, window_start, counter)
VALUES (
  sqlc.arg(purpose)::text,
  sqlc.arg(subject_hash)::text,
  sqlc.arg(window_start)::timestamptz,
  1
)
ON CONFLICT (purpose, subject_hash, window_start)
DO UPDATE SET counter = rate_limit_buckets.counter + 1
RETURNING counter;

-- name: DeleteExpiredIdempotency :exec
DELETE FROM idempotency_requests WHERE expires_at < sqlc.arg(before)::timestamptz;

-- name: DeleteOldRateLimits :exec
DELETE FROM rate_limit_buckets WHERE window_start < sqlc.arg(before)::timestamptz;
