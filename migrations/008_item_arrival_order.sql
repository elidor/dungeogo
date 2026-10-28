-- Arrival order: when each Item came to its current Owner. look, inventory
-- and numbered Item targets (2.sword) list Items oldest first by this, with
-- the ID breaking ties between Items that arrived together.
ALTER TABLE item_instances ADD COLUMN held_since TIMESTAMP WITH TIME ZONE;
UPDATE item_instances SET held_since = COALESCE(created_at, now());
ALTER TABLE item_instances
    ALTER COLUMN held_since SET NOT NULL,
    ALTER COLUMN held_since SET DEFAULT now();

-- Supersedes idx_item_owner_kind_id: same leading columns, plus the order.
CREATE INDEX idx_item_owner_arrival ON item_instances(owner_kind, owner_id, held_since, id);
DROP INDEX idx_item_owner_kind_id;
