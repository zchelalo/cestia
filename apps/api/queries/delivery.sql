-- name: ClaimEmailDeliveries :many
WITH candidates AS (
  SELECT id
  FROM email_deliveries
  WHERE ((status = 'queued' AND next_attempt_at <= sqlc.arg(now)::timestamptz)
      OR (status = 'processing' AND lease_until < sqlc.arg(now)::timestamptz))
  ORDER BY next_attempt_at, created_at
  LIMIT sqlc.arg(limit_count)::integer
  FOR UPDATE SKIP LOCKED
)
UPDATE email_deliveries d
SET status = 'processing',
    attempt_count = d.attempt_count + 1,
    lease_token = sqlc.arg(lease_token)::uuid,
    lease_until = sqlc.arg(lease_until)::timestamptz,
    updated_at = sqlc.arg(now)::timestamptz
FROM candidates c
WHERE d.id = c.id
RETURNING d.id, d.invitation_id, d.action_id, d.generation, d.recipient,
          d.attempt_count, d.lease_token;

-- name: MarkDeliverySent :execrows
UPDATE email_deliveries
SET status = 'sent', lease_token = NULL, lease_until = NULL, sent_at = sqlc.arg(at)::timestamptz,
    last_error = NULL, updated_at = sqlc.arg(at)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND lease_token = sqlc.arg(lease_token)::uuid
  AND status = 'processing';

-- name: MarkDeliveryRetry :execrows
UPDATE email_deliveries
SET status = 'queued', lease_token = NULL, lease_until = NULL,
    next_attempt_at = sqlc.arg(next_attempt_at)::timestamptz,
    last_error = sqlc.arg(last_error)::text,
    updated_at = sqlc.arg(at)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND lease_token = sqlc.arg(lease_token)::uuid
  AND status = 'processing';

-- name: MarkDeliveryCanceled :execrows
UPDATE email_deliveries
SET status = 'canceled', lease_token = NULL, lease_until = NULL,
    last_error = sqlc.arg(last_error)::text,
    updated_at = sqlc.arg(at)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND lease_token = sqlc.arg(lease_token)::uuid
  AND status = 'processing';

-- name: MarkDeliveryFailed :execrows
UPDATE email_deliveries
SET status = 'failed', lease_token = NULL, lease_until = NULL,
    last_error = sqlc.arg(last_error)::text,
    updated_at = sqlc.arg(at)::timestamptz
WHERE id = sqlc.arg(id)::uuid
  AND lease_token = sqlc.arg(lease_token)::uuid
  AND status = 'processing';
