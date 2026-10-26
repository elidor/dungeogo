-- A 3x3 starter grid. starting_room is the center and player spawn point.
INSERT INTO room_definitions (room_id, zone_id, name, description, exits)
VALUES
    ('newbie_northwest', 'newbie_zone', 'Northwest Corner', 'The northwest edge of the newbie zone is quiet and open.', '{"east": "newbie_north", "south": "newbie_west"}'),
    ('newbie_north', 'newbie_zone', 'Northern Walk', 'A worn path runs along the northern edge of the newbie zone.', '{"east": "newbie_northeast", "south": "starting_room", "west": "newbie_northwest"}'),
    ('newbie_northeast', 'newbie_zone', 'Northeast Corner', 'The northeastern corner offers a clear view across the training grounds.', '{"south": "newbie_east", "west": "newbie_north"}'),
    ('newbie_west', 'newbie_zone', 'Western Walk', 'A short trail leads between the western rooms of the newbie zone.', '{"north": "newbie_northwest", "east": "starting_room", "south": "newbie_southwest"}'),
    ('starting_room', 'newbie_zone', 'A Simple Room', 'You are in a basic room with stone walls and a dirt floor.', '{"north": "newbie_north", "south": "newbie_south", "east": "newbie_east", "west": "newbie_west"}'),
    ('newbie_east', 'newbie_zone', 'Eastern Walk', 'A short trail leads between the eastern rooms of the newbie zone.', '{"north": "newbie_northeast", "south": "newbie_southeast", "west": "starting_room"}'),
    ('newbie_southwest', 'newbie_zone', 'Southwest Corner', 'The southwest edge of the newbie zone is sheltered by low stone walls.', '{"north": "newbie_west", "east": "newbie_south"}'),
    ('newbie_south', 'newbie_zone', 'Southern Walk', 'A worn path runs along the southern edge of the newbie zone.', '{"north": "starting_room", "east": "newbie_southeast", "west": "newbie_southwest"}'),
    ('newbie_southeast', 'newbie_zone', 'Southeast Corner', 'The southeast corner is bright and exposed to the morning sun.', '{"north": "newbie_east", "west": "newbie_south"}')
ON CONFLICT (room_id) DO UPDATE SET
    zone_id = EXCLUDED.zone_id,
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    exits = EXCLUDED.exits,
    updated_at = NOW();
