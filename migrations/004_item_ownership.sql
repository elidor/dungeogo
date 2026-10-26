-- Keep the legacy owner_id API, but make owner kind explicit and enforce real owners.
ALTER TABLE item_instances ADD COLUMN owner_kind TEXT;

DO $$
DECLARE invalid_item RECORD;
BEGIN
    SELECT i.id, i.owner_id INTO invalid_item
    FROM item_instances i
    LEFT JOIN room_definitions r ON r.room_id = i.owner_id
    LEFT JOIN characters c ON c.id::text = i.owner_id
    WHERE (r.room_id IS NULL) = (c.id IS NULL)
    ORDER BY i.id LIMIT 1;
    IF FOUND THEN
        RAISE EXCEPTION 'Item ownership migration: item % owner % must identify exactly one existing room or character. Repair missing or ambiguous ownership before starting; no items were deleted.', invalid_item.id, invalid_item.owner_id;
    END IF;
    UPDATE item_instances i SET owner_kind = CASE
        WHEN EXISTS (SELECT 1 FROM room_definitions r WHERE r.room_id = i.owner_id)
        THEN 'room' ELSE 'character' END;
END $$;

ALTER TABLE item_instances
    ALTER COLUMN owner_kind SET NOT NULL,
    ADD CONSTRAINT item_owner_kind CHECK (owner_kind IN ('room', 'character')),
    ADD COLUMN owner_room_id VARCHAR(100) GENERATED ALWAYS AS
        (CASE WHEN owner_kind = 'room' THEN owner_id END) STORED,
    ADD COLUMN owner_character_id UUID GENERATED ALWAYS AS
        (CASE WHEN owner_kind = 'character' THEN owner_id::uuid END) STORED;

ALTER TABLE item_instances
    ADD CONSTRAINT item_owner_room FOREIGN KEY (owner_room_id)
        REFERENCES room_definitions(room_id) ON DELETE RESTRICT,
    ADD CONSTRAINT item_owner_character FOREIGN KEY (owner_character_id)
        REFERENCES characters(id) ON DELETE RESTRICT;

CREATE INDEX idx_item_owner_kind_id ON item_instances(owner_kind, owner_id);
