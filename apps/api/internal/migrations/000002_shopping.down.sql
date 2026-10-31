DROP TRIGGER IF EXISTS group_members_owner_active ON group_members;
DROP TRIGGER IF EXISTS groups_owner_active ON groups;
DROP FUNCTION IF EXISTS assert_group_owner_active();
ALTER TABLE email_deliveries DROP CONSTRAINT IF EXISTS email_deliveries_invitation_fk;
ALTER TABLE activity_events
  DROP CONSTRAINT IF EXISTS activity_events_purchase_fk,
  DROP CONSTRAINT IF EXISTS activity_events_item_fk,
  DROP CONSTRAINT IF EXISTS activity_events_list_fk,
  DROP CONSTRAINT IF EXISTS activity_events_group_fk;
DROP TABLE IF EXISTS purchases;
DROP TABLE IF EXISTS shopping_list_items;
DROP TABLE IF EXISTS shopping_lists;
DROP TABLE IF EXISTS group_invitations;
DROP TABLE IF EXISTS group_members;
DROP TABLE IF EXISTS groups;
