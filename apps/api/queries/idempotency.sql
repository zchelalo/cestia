-- name: AcquireIdempotencyLock :exec
SELECT pg_advisory_xact_lock(hashtextextended(sqlc.arg(lock_key)::text, 0));

-- name: GetIdempotencyRequest :one
SELECT actor_id, operation, key, request_hash, response_status, response_body,
       resource_id, expires_at, created_at
FROM idempotency_requests
WHERE actor_id = sqlc.arg(actor_id)::uuid
  AND operation = sqlc.arg(operation)::text
  AND key = sqlc.arg(key)::text
  AND expires_at > sqlc.arg(now)::timestamptz
FOR UPDATE;

-- name: UpsertIdempotencyRequest :exec
INSERT INTO idempotency_requests (
  actor_id, operation, key, request_hash, response_status, response_body,
  resource_id, expires_at, created_at
) VALUES (
  sqlc.arg(actor_id)::uuid,
  sqlc.arg(operation)::text,
  sqlc.arg(key)::text,
  sqlc.arg(request_hash)::text,
  sqlc.arg(response_status)::integer,
  sqlc.arg(response_body)::jsonb,
  sqlc.narg(resource_id)::uuid,
  sqlc.arg(expires_at)::timestamptz,
  sqlc.arg(created_at)::timestamptz
)
ON CONFLICT (actor_id, operation, key) DO UPDATE
SET request_hash = EXCLUDED.request_hash,
    response_body = EXCLUDED.response_body,
    resource_id = EXCLUDED.resource_id,
    expires_at = EXCLUDED.expires_at,
    created_at = EXCLUDED.created_at;

-- name: UpgradeLegacyIdempotencyRequest :exec
UPDATE idempotency_requests
SET resource_id = sqlc.arg(resource_id)::uuid,
    request_hash = sqlc.arg(request_hash)::text
WHERE actor_id = sqlc.arg(actor_id)::uuid
  AND operation = sqlc.arg(operation)::text
  AND key = sqlc.arg(key)::text
  AND resource_id IS NULL;
