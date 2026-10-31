CREATE TABLE users (
  id uuid PRIMARY KEY,
  email text NOT NULL,
  display_name text NOT NULL,
  password_hash text NOT NULL,
  email_verified_at timestamptz,
  locale text NOT NULL DEFAULT 'es',
  timezone text NOT NULL DEFAULT 'America/Mexico_City',
  disabled_at timestamptz,
  created_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL
);

CREATE UNIQUE INDEX users_email_unique ON users (lower(email));

CREATE TABLE sessions (
  id uuid PRIMARY KEY,
  user_id uuid NOT NULL REFERENCES users (id),
  token_hash bytea NOT NULL UNIQUE,
  csrf_hash bytea NOT NULL,
  expires_at timestamptz NOT NULL,
  revoked_at timestamptz,
  created_at timestamptz NOT NULL
);

CREATE INDEX sessions_user_active ON sessions (user_id, expires_at) WHERE revoked_at IS NULL;

CREATE TABLE email_actions (
  id uuid PRIMARY KEY,
  user_id uuid NOT NULL REFERENCES users (id),
  purpose text NOT NULL CHECK (purpose IN ('verify_email', 'reset_password')),
  token_hash bytea NOT NULL UNIQUE,
  generation integer NOT NULL DEFAULT 1,
  expires_at timestamptz NOT NULL,
  consumed_at timestamptz,
  created_at timestamptz NOT NULL
);

CREATE INDEX email_actions_user_purpose ON email_actions (user_id, purpose, consumed_at);

CREATE TABLE activity_events (
  id uuid PRIMARY KEY,
  scope_type text NOT NULL CHECK (scope_type IN ('user', 'personal', 'group')),
  actor_id uuid REFERENCES users (id),
  group_id uuid,
  list_id uuid,
  item_id uuid,
  purchase_id uuid,
  action text NOT NULL,
  payload_version integer NOT NULL DEFAULT 1,
  payload jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL
);

CREATE INDEX activity_scope_created ON activity_events (scope_type, created_at DESC, id DESC);
CREATE INDEX activity_group_created ON activity_events (group_id, created_at DESC, id DESC) WHERE group_id IS NOT NULL;
CREATE INDEX activity_list_created ON activity_events (list_id, created_at DESC, id DESC) WHERE list_id IS NOT NULL;
CREATE INDEX activity_actor_created ON activity_events (actor_id, created_at DESC, id DESC);

CREATE TABLE idempotency_requests (
  actor_id uuid NOT NULL REFERENCES users (id),
  operation text NOT NULL,
  key text NOT NULL,
  request_hash text NOT NULL,
  response_status integer NOT NULL,
  response_body jsonb NOT NULL,
  resource_id uuid,
  expires_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL,
  PRIMARY KEY (actor_id, operation, key)
);

CREATE INDEX idempotency_expiry ON idempotency_requests (expires_at);

CREATE TABLE rate_limit_buckets (
  purpose text NOT NULL,
  subject_hash text NOT NULL,
  window_start timestamptz NOT NULL,
  counter integer NOT NULL,
  PRIMARY KEY (purpose, subject_hash, window_start)
);

CREATE TABLE email_deliveries (
  id uuid PRIMARY KEY,
  invitation_id uuid,
  action_id uuid REFERENCES email_actions (id),
  generation integer NOT NULL DEFAULT 1,
  recipient text NOT NULL,
  status text NOT NULL DEFAULT 'queued' CHECK (status IN ('queued', 'processing', 'sent', 'failed', 'canceled')),
  attempt_count integer NOT NULL DEFAULT 0,
  next_attempt_at timestamptz NOT NULL,
  lease_token uuid,
  lease_until timestamptz,
  provider_message_id text,
  sent_at timestamptz,
  last_error text,
  created_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL,
  CHECK ((invitation_id IS NOT NULL) <> (action_id IS NOT NULL))
);

CREATE INDEX email_deliveries_claim ON email_deliveries (status, next_attempt_at);
CREATE INDEX email_deliveries_invitation ON email_deliveries (invitation_id) WHERE invitation_id IS NOT NULL;
