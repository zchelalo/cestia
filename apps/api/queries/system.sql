-- name: SetLocalApplicationName :exec
SELECT set_config('application_name', sqlc.arg(name)::text, true);

-- name: CountLockWaitsForApplication :one
SELECT count(*)
FROM pg_stat_activity
WHERE application_name = sqlc.arg(application_name)::text
  AND wait_event_type = 'Lock';

-- name: AcquireSchemaMigrationLock :exec
SELECT pg_advisory_xact_lock(hashtextextended('cestia:schema-migrations', 0));

-- name: SchemaMigrationExists :one
SELECT EXISTS (
  SELECT 1 FROM schema_migrations
  WHERE version = sqlc.arg(version)::text
);

-- name: RecordSchemaMigration :exec
INSERT INTO schema_migrations (version)
VALUES (sqlc.arg(version)::text)
ON CONFLICT (version) DO NOTHING;
