package postgres

import (
	"testing"

	"github.com/elidor/dungeogo/pkg/persistence/interfaces"
)

func TestWorldRepository_SaveAndLoadRoomDefinition(t *testing.T) {
	repoManager := setupTestDB(t)
	if repoManager == nil {
		t.Skip("Database not available for testing")
	}

	definition := &interfaces.RoomDefinition{
		ID:          "training_hall",
		ZoneID:      "newbie_zone",
		Name:        "Training Hall",
		Description: "Wooden practice weapons line the walls.",
		Exits:       map[string]string{"south": "starting_room"},
		Flags:       map[string]interface{}{"safe": true},
	}

	if err := repoManager.World().SaveRoomDefinition(definition); err != nil {
		t.Fatalf("failed to save room definition: %v", err)
	}

	loaded, err := repoManager.World().LoadRoomDefinition(definition.ID)
	if err != nil {
		t.Fatalf("failed to load room definition: %v", err)
	}

	if loaded.ZoneID != definition.ZoneID || loaded.Name != definition.Name || loaded.Description != definition.Description {
		t.Fatalf("loaded definition does not match: %#v", loaded)
	}
	if loaded.Exits["south"] != "starting_room" {
		t.Errorf("expected south exit to starting_room, got %q", loaded.Exits["south"])
	}
	if safe, ok := loaded.Flags["safe"].(bool); !ok || !safe {
		t.Errorf("expected safe flag, got %#v", loaded.Flags["safe"])
	}
}
