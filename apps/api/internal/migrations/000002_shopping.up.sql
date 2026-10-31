CREATE TABLE groups (
  id uuid PRIMARY KEY,
  name text NOT NULL,
  description text NOT NULL DEFAULT '',
  owner_user_id uuid NOT NULL REFERENCES users (id),
  created_by uuid NOT NULL REFERENCES users (id),
  archived_at timestamptz,
  version bigint NOT NULL DEFAULT 1,
  created_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL
);

CREATE INDEX groups_owner ON groups (owner_user_id);

CREATE TABLE group_members (
  group_id uuid NOT NULL REFERENCES groups (id),
  user_id uuid NOT NULL REFERENCES users (id),
  role text NOT NULL CHECK (role IN ('admin', 'member')),
  joined_at timestamptz NOT NULL,
  removed_at timestamptz,
  version bigint NOT NULL DEFAULT 1,
  PRIMARY KEY (group_id, user_id)
);

CREATE INDEX group_members_user_active ON group_members (user_id) WHERE removed_at IS NULL;
CREATE INDEX group_members_group_active ON group_members (group_id) WHERE removed_at IS NULL;

CREATE TABLE group_invitations (
  id uuid PRIMARY KEY,
  group_id uuid NOT NULL REFERENCES groups (id),
  email text NOT NULL,
  invited_by uuid NOT NULL REFERENCES users (id),
  status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'accepted', 'rejected', 'canceled', 'expired')),
  token_hash bytea NOT NULL,
  generation integer NOT NULL DEFAULT 1,
  expires_at timestamptz NOT NULL,
  resolved_at timestamptz,
  resolved_by uuid REFERENCES users (id),
  version bigint NOT NULL DEFAULT 1,
  created_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL
);

CREATE UNIQUE INDEX group_invitations_pending_email
  ON group_invitations (group_id, lower(email))
  WHERE status = 'pending';
CREATE INDEX group_invitations_token ON group_invitations (token_hash);
CREATE INDEX group_invitations_group ON group_invitations (group_id, created_at DESC);

CREATE TABLE shopping_lists (
  id uuid PRIMARY KEY,
  name text NOT NULL,
  description text NOT NULL DEFAULT '',
  personal_owner_id uuid REFERENCES users (id),
  group_id uuid REFERENCES groups (id),
  created_by uuid NOT NULL REFERENCES users (id),
  archived_at timestamptz,
  version bigint NOT NULL DEFAULT 1,
  created_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL,
  CHECK ((personal_owner_id IS NOT NULL) <> (group_id IS NOT NULL))
);

CREATE INDEX shopping_lists_personal ON shopping_lists (personal_owner_id, archived_at);
CREATE INDEX shopping_lists_group ON shopping_lists (group_id, archived_at);

CREATE TABLE shopping_list_items (
  id uuid PRIMARY KEY,
  list_id uuid NOT NULL REFERENCES shopping_lists (id),
  name text NOT NULL,
  description text NOT NULL DEFAULT '',
  quantity numeric(12,3) NOT NULL CHECK (quantity > 0 AND quantity <= 999999.999),
  unit text NOT NULL CHECK (unit IN ('unit', 'kg', 'g', 'l', 'ml', 'package')),
  estimated_unit_price numeric(14,4) CHECK (estimated_unit_price >= 0 AND estimated_unit_price <= 99999999.9999),
  created_by uuid NOT NULL REFERENCES users (id),
  assigned_to uuid REFERENCES users (id),
  notes text NOT NULL DEFAULT '',
  deleted_at timestamptz,
  version bigint NOT NULL DEFAULT 1,
  created_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL
);

CREATE INDEX shopping_items_list ON shopping_list_items (list_id, created_at, id) WHERE deleted_at IS NULL;

CREATE TABLE purchases (
  id uuid PRIMARY KEY,
  item_id uuid NOT NULL REFERENCES shopping_list_items (id),
  item_name text NOT NULL,
  quantity numeric(12,3) NOT NULL,
  unit text NOT NULL,
  actual_unit_price numeric(14,4) CHECK (actual_unit_price >= 0 AND actual_unit_price <= 99999999.9999),
  total_amount numeric(18,2) GENERATED ALWAYS AS (round(quantity * actual_unit_price, 2)) STORED,
  currency char(3) NOT NULL DEFAULT 'MXN' CHECK (currency = 'MXN'),
  recorded_by uuid NOT NULL REFERENCES users (id),
  purchased_by uuid NOT NULL REFERENCES users (id),
  paid_by uuid REFERENCES users (id),
  purchased_at timestamptz NOT NULL,
  notes text NOT NULL DEFAULT '',
  reversed_by uuid REFERENCES users (id),
  reversed_at timestamptz,
  reversal_reason text,
  version bigint NOT NULL DEFAULT 1,
  created_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL,
  CHECK ((reversed_at IS NULL AND reversed_by IS NULL AND reversal_reason IS NULL)
      OR (reversed_at IS NOT NULL AND reversed_by IS NOT NULL AND reversal_reason IS NOT NULL))
);

CREATE UNIQUE INDEX purchases_one_active_per_item ON purchases (item_id) WHERE reversed_at IS NULL;
CREATE INDEX purchases_item_history ON purchases (item_id, created_at DESC);
CREATE INDEX purchases_purchased_at ON purchases (purchased_at DESC, id DESC) WHERE reversed_at IS NULL;
CREATE INDEX purchases_paid_by ON purchases (paid_by, purchased_at DESC) WHERE reversed_at IS NULL;

ALTER TABLE activity_events
  ADD CONSTRAINT activity_events_group_fk FOREIGN KEY (group_id) REFERENCES groups (id),
  ADD CONSTRAINT activity_events_list_fk FOREIGN KEY (list_id) REFERENCES shopping_lists (id),
  ADD CONSTRAINT activity_events_item_fk FOREIGN KEY (item_id) REFERENCES shopping_list_items (id),
  ADD CONSTRAINT activity_events_purchase_fk FOREIGN KEY (purchase_id) REFERENCES purchases (id);

ALTER TABLE email_deliveries
  ADD CONSTRAINT email_deliveries_invitation_fk FOREIGN KEY (invitation_id) REFERENCES group_invitations (id);

-- The owner is represented once in groups and must always remain an active,
-- enabled member. Constraint triggers run at commit so group creation can
-- insert the group and its first membership in one transaction.
CREATE OR REPLACE FUNCTION assert_group_owner_active() RETURNS trigger AS $$
DECLARE
  target_group uuid;
  owner_id uuid;
BEGIN
  target_group := COALESCE(NEW.id, NEW.group_id);
  SELECT owner_user_id INTO owner_id FROM groups WHERE id = target_group;
  IF owner_id IS NULL THEN
    RETURN NEW;
  END IF;
  IF NOT EXISTS (
    SELECT 1
    FROM group_members gm
    JOIN users u ON u.id = gm.user_id
    WHERE gm.group_id = target_group
      AND gm.user_id = owner_id
      AND gm.removed_at IS NULL
      AND u.disabled_at IS NULL
  ) THEN
    RAISE EXCEPTION 'group % must retain its active owner', target_group
      USING ERRCODE = '23514';
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE CONSTRAINT TRIGGER groups_owner_active
  AFTER INSERT OR UPDATE ON groups
  DEFERRABLE INITIALLY DEFERRED
  FOR EACH ROW EXECUTE FUNCTION assert_group_owner_active();

CREATE CONSTRAINT TRIGGER group_members_owner_active
  AFTER INSERT OR UPDATE OR DELETE ON group_members
  DEFERRABLE INITIALLY DEFERRED
  FOR EACH ROW EXECUTE FUNCTION assert_group_owner_active();
