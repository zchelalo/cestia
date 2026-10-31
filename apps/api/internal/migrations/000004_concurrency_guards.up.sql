-- Keep at most one live email action per user and purpose. Older duplicates are
-- consumed instead of deleted so delivery history and audit remain intact.
WITH ranked AS (
  SELECT id,
         row_number() OVER (PARTITION BY user_id, purpose ORDER BY created_at DESC, id DESC) AS position
  FROM email_actions
  WHERE consumed_at IS NULL
), stale AS (
  SELECT id FROM ranked WHERE position > 1
)
UPDATE email_deliveries
SET status = 'canceled', updated_at = now()
WHERE action_id IN (SELECT id FROM stale)
  AND status IN ('queued', 'processing');

WITH ranked AS (
  SELECT id,
         row_number() OVER (PARTITION BY user_id, purpose ORDER BY created_at DESC, id DESC) AS position
  FROM email_actions
  WHERE consumed_at IS NULL
), stale AS (
  SELECT id FROM ranked WHERE position > 1
)
UPDATE email_actions
SET consumed_at = now()
WHERE id IN (SELECT id FROM stale);

CREATE UNIQUE INDEX email_actions_one_active_per_purpose
  ON email_actions (user_id, purpose)
  WHERE consumed_at IS NULL;

-- Revalidate the owner under a shared lock so a concurrent deactivation cannot
-- commit between the group check and this trigger.
CREATE OR REPLACE FUNCTION assert_group_owner_active() RETURNS trigger AS $$
DECLARE
  target_group uuid;
  owner_id uuid;
BEGIN
  IF TG_TABLE_NAME = 'groups' THEN
    target_group := NEW.id;
  ELSIF TG_OP = 'DELETE' THEN
    target_group := OLD.group_id;
  ELSE
    target_group := NEW.group_id;
  END IF;

  SELECT owner_user_id INTO owner_id FROM groups WHERE id = target_group;
  IF owner_id IS NULL THEN
    IF TG_OP = 'DELETE' THEN
      RETURN OLD;
    END IF;
    RETURN NEW;
  END IF;
  PERFORM 1
  FROM group_members gm
  JOIN users u ON u.id = gm.user_id
  WHERE gm.group_id = target_group
    AND gm.user_id = owner_id
    AND gm.removed_at IS NULL
    AND u.disabled_at IS NULL
  FOR SHARE OF u;
  IF NOT FOUND THEN
    RAISE EXCEPTION 'group % must retain its active owner', target_group
      USING ERRCODE = '23514';
  END IF;
  IF TG_OP = 'DELETE' THEN
    RETURN OLD;
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION assert_user_without_owned_groups() RETURNS trigger AS $$
BEGIN
  IF NEW.disabled_at IS NOT NULL AND OLD.disabled_at IS NULL
     AND EXISTS (SELECT 1 FROM groups WHERE owner_user_id = NEW.id) THEN
    RAISE EXCEPTION 'user % still owns groups', NEW.id
      USING ERRCODE = '23514';
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE CONSTRAINT TRIGGER users_owner_active
  AFTER UPDATE OF disabled_at ON users
  DEFERRABLE INITIALLY DEFERRED
  FOR EACH ROW EXECUTE FUNCTION assert_user_without_owned_groups();
