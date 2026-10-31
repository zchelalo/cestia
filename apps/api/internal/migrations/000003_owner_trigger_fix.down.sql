-- Forward-only correctness fix; the previous trigger body was invalid for
-- group inserts because it referenced NEW.group_id on the groups relation.
SELECT 1;
