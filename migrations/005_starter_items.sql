-- Fixed instance IDs and migration history seed once; never reset an item's owner.
INSERT INTO item_instances (id, template_id, owner_kind, owner_id, custom_name, quantity, durability)
VALUES
    ('90000000-0000-4000-8000-000000000001', 'rusty_sword', 'room', 'starting_room', 'Training Sword', 1, 50),
    ('90000000-0000-4000-8000-000000000002', 'travel_lantern', 'room', 'starting_room', 'Travel Lantern', 1, 100),
    ('90000000-0000-4000-8000-000000000003', 'smooth_stone', 'room', 'newbie_north', 'Smooth Stone', 1, 100)
ON CONFLICT (id) DO NOTHING;
