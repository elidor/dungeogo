-- ADR-0001: deleting a Character destroys the Items it holds. Replace the
-- RESTRICT foreign key from 004 so the Character and its Items are removed
-- by one atomic DELETE. Room ownership stays RESTRICT.
ALTER TABLE item_instances
    DROP CONSTRAINT item_owner_character,
    ADD CONSTRAINT item_owner_character FOREIGN KEY (owner_character_id)
        REFERENCES characters(id) ON DELETE CASCADE;
