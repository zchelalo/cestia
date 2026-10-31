DROP TRIGGER IF EXISTS users_owner_active ON users;
DROP FUNCTION IF EXISTS assert_user_without_owned_groups();
DROP INDEX IF EXISTS email_actions_one_active_per_purpose;

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
  IF TG_OP = 'DELETE' THEN
    RETURN OLD;
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
