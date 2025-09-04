-- Initial database schema for DungeoGo

-- Players table (account level)
CREATE TABLE players (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_login TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    account_status INTEGER DEFAULT 0, -- 0=Active, 1=Suspended, 2=Banned
    subscription JSONB,
    preferences JSONB NOT NULL DEFAULT '{}',
    max_characters INTEGER DEFAULT 5,
    current_character_id UUID
);

-- Characters table (game avatars)
CREATE TABLE characters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    name VARCHAR(50) UNIQUE NOT NULL,
    race_id VARCHAR(50) NOT NULL,
    class_id VARCHAR(50) NOT NULL,
    stats JSONB NOT NULL DEFAULT '{}',
    skills JSONB NOT NULL DEFAULT '{}',
    location JSONB NOT NULL DEFAULT '{}',
    state INTEGER DEFAULT 0, -- 0=Alive, 1=Dead, etc.
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_played TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    play_time INTERVAL DEFAULT '0 seconds',
    level INTEGER DEFAULT 1,
    experience INTEGER DEFAULT 0,
    death_count INTEGER DEFAULT 0,
    kill_count INTEGER DEFAULT 0,
    description TEXT DEFAULT '',
    appearance JSONB NOT NULL DEFAULT '{}'
);

-- Item instances table (actual items owned by characters/rooms)
CREATE TABLE item_instances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_id VARCHAR(100) NOT NULL,
    owner_id UUID NOT NULL, -- Can reference characters.id or room IDs
    quantity INTEGER DEFAULT 1,
    durability INTEGER DEFAULT 100,
    enchantments JSONB NOT NULL DEFAULT '[]',
    custom_name VARCHAR(255),
    modifications JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_used TIMESTAMP WITH TIME ZONE
);

-- Room states table (dynamic room data)
CREATE TABLE room_states (
    room_id VARCHAR(100) PRIMARY KEY,
    items JSONB NOT NULL DEFAULT '[]',
    npcs JSONB NOT NULL DEFAULT '[]',
    players JSONB NOT NULL DEFAULT '[]',
    flags JSONB NOT NULL DEFAULT '{}',
    last_update TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- NPC states table (dynamic NPC data)
CREATE TABLE npc_states (
    npc_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_id VARCHAR(100) NOT NULL,
    health INTEGER NOT NULL DEFAULT 100,
    location JSONB NOT NULL DEFAULT '{}',
    inventory JSONB NOT NULL DEFAULT '[]',
    state VARCHAR(50) DEFAULT 'idle',
    last_update TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- World events table (global game events)
CREATE TABLE world_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type VARCHAR(100) NOT NULL,
    description TEXT,
    start_time TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    end_time TIMESTAMP WITH TIME ZONE,
    data JSONB NOT NULL DEFAULT '{}'
);

-- Create indexes for performance
CREATE INDEX idx_characters_player_id ON characters(player_id);
CREATE INDEX idx_characters_name ON characters(name);
CREATE INDEX idx_item_instances_owner ON item_instances(owner_id);
CREATE INDEX idx_item_instances_template ON item_instances(template_id);
CREATE INDEX idx_world_events_active ON world_events(start_time, end_time) WHERE end_time IS NULL OR end_time > NOW();

-- Create a sample admin player for testing
INSERT INTO players (username, email, password_hash, account_status) VALUES 
('admin', 'admin@dungeogo.com', 'admin', 0);