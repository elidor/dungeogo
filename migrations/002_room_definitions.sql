-- Static room metadata. Dynamic occupants, items, and NPCs remain in room_states.
CREATE TABLE room_definitions (
    room_id VARCHAR(100) PRIMARY KEY,
    zone_id VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    exits JSONB NOT NULL DEFAULT '{}',
    flags JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_room_definitions_zone_id ON room_definitions(zone_id);

INSERT INTO room_definitions (room_id, zone_id, name, description)
VALUES (
    'starting_room',
    'newbie_zone',
    'A Simple Room',
    'You are in a basic room with stone walls and a dirt floor.'
);
