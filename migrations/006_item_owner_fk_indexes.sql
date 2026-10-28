-- Index the generated owner columns behind the item_owner_room and
-- item_owner_character foreign keys. Without them, deleting a room or a
-- character makes PostgreSQL scan every item to enforce ON DELETE.
-- Partial indexes skip the rows owned by the other kind, which are NULL here.
CREATE INDEX idx_item_owner_room_id ON item_instances(owner_room_id)
    WHERE owner_room_id IS NOT NULL;
CREATE INDEX idx_item_owner_character_id ON item_instances(owner_character_id)
    WHERE owner_character_id IS NOT NULL;
