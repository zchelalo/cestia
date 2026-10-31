-- SQLC-only catalog stub. The migration runner embeds only *.up.sql, so this
-- file is never executed; it lets the generator typecheck diagnostics used by
-- concurrency tests against PostgreSQL's pg_stat_activity view.
CREATE TABLE pg_stat_activity (
  application_name text,
  wait_event_type text
);
