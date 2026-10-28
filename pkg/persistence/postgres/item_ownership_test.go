package postgres

import (
	"context"
	"testing"

	"github.com/elidor/dungeogo/internal/testdb"
	"github.com/elidor/dungeogo/pkg/game/items"
	"github.com/elidor/dungeogo/pkg/persistence/interfaces"
)

const starterSword = "90000000-0000-4000-8000-000000000001"
const starterLantern = "90000000-0000-4000-8000-000000000002"

func TestStarterItemsPersistAcrossRepeatedBootAndRepositoryRestart(t *testing.T) {
	_, url := testdb.New(t, true)
	manager, err := NewPostgreSQLRepositoryManager(url)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	roomItems, err := manager.Items().GetRoomItems("starting_room")
	if err != nil || len(roomItems) != 2 {
		t.Fatalf("starter items: %v %v", roomItems, err)
	}
	for _, item := range roomItems {
		if item.Quantity != 1 || item.Owner.Kind != items.OwnerRoom || item.ID == "" || item.CustomName == "" {
			t.Fatalf("unidentifiable or stacking starter item: %+v", item)
		}
	}
	owner := existingItemCharacter(t, manager)
	if err := manager.Items().TransferItem(starterSword, owner); err != nil {
		t.Fatal(err)
	}
	if err := manager.Items().DeleteItemInstance(starterLantern); err != nil {
		t.Fatal(err)
	}
	if err := manager.Close(); err != nil {
		t.Fatal(err)
	}
	manager, err = NewPostgreSQLRepositoryManager(url)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	for i := 0; i < 2; i++ {
		if err := manager.Bootstrap(context.Background(), DefaultStartingRoom); err != nil {
			t.Fatal(err)
		}
	}
	item, err := manager.Items().GetItemInstance(starterSword)
	if err != nil || item.Owner.ID != owner || item.Owner.Kind != items.OwnerCharacter {
		t.Fatalf("boot reset sword: %+v %v", item, err)
	}
	if _, err := manager.Items().GetItemInstance(starterLantern); err == nil {
		t.Fatal("deleted seed respawned")
	}
	var count int
	if err := manager.GetDB().QueryRow(`SELECT count(*) FROM item_instances`).Scan(&count); err != nil || count != 2 {
		t.Fatalf("duplicate seeds: %d %v", count, err)
	}
}

func TestItemOwnershipConstraintsAndKindIsolation(t *testing.T) {
	manager := setupTestDB(t)
	id := existingItemCharacter(t, manager)
	// Explicit kinds disambiguate a valid room whose ID equals a character ID.
	if err := manager.World().SaveRoomDefinition(&interfaces.RoomDefinition{ID: id, ZoneID: "test", Name: "Shared ID", Description: "Explicit owners", Exits: map[string]string{}, Flags: map[string]interface{}{}}); err != nil {
		t.Fatal(err)
	}
	roomItem := createTestItemInstance()
	roomItem.Owner = items.Owner{ID: id}
	roomItem.Owner.Kind = items.OwnerRoom
	charItem := createTestItemInstance()
	charItem.Owner = items.Owner{ID: id}
	charItem.Owner.Kind = items.OwnerCharacter
	for _, item := range []*items.ItemInstance{roomItem, charItem} {
		if err := manager.Items().CreateItemInstance(item); err != nil {
			t.Fatal(err)
		}
	}
	rooms, err := manager.Items().GetRoomItems(id)
	if err != nil || len(rooms) != 1 || rooms[0].ID != roomItem.ID {
		t.Fatalf("room ownership mixed: %v %v", rooms, err)
	}
	inventory, err := manager.Items().GetPlayerItems(id)
	if err != nil || len(inventory) != 1 || inventory[0].ID != charItem.ID {
		t.Fatalf("character ownership mixed: %v %v", inventory, err)
	}
	ambiguous := createTestItemInstance()
	ambiguous.Owner = items.Owner{ID: id}
	if err := manager.Items().CreateItemInstance(ambiguous); err == nil {
		t.Fatal("legacy API guessed ambiguous ownership")
	}
	for _, tc := range []struct{ kind, owner string }{
		{"room", "missing_room"}, {"character", "00000000-0000-0000-0000-000000000099"}, {"character", "not-a-uuid"}, {"unknown", "starting_room"},
	} {
		if _, err := manager.GetDB().Exec(`INSERT INTO item_instances(template_id,owner_kind,owner_id) VALUES ('invalid',$1,$2)`, tc.kind, tc.owner); err == nil {
			t.Fatalf("accepted dangling/invalid owner %+v", tc)
		}
	}
	if _, err := manager.GetDB().Exec(`DELETE FROM characters WHERE id=$1`, id); err == nil {
		t.Fatal("deleted character while retaining owned item")
	}
	if _, err := manager.GetDB().Exec(`DELETE FROM room_definitions WHERE room_id=$1`, id); err == nil {
		t.Fatal("deleted room while retaining owned item")
	}
	if err := manager.Items().TransferItem(charItem.ID, "missing_room"); err == nil {
		t.Fatal("transferred to nonexistent owner")
	}
	unchanged, err := manager.Items().GetItemInstance(charItem.ID)
	if err != nil || unchanged.Owner.Kind != items.OwnerCharacter || unchanged.Owner.ID != id {
		t.Fatalf("failed transfer changed ownership: %+v %v", unchanged, err)
	}
}

func TestItemOwnerResolution(t *testing.T) {
	_, url := testdb.New(t, true)
	manager, err := NewPostgreSQLRepositoryManager(url)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	character := existingItemCharacter(t, manager)

	for _, tc := range []struct {
		name  string
		given items.Owner
		want  items.Owner
	}{
		{"room kind inferred from ID", items.Owner{ID: "newbie_west"}, items.RoomOwner("newbie_west")},
		{"character kind inferred from ID", items.Owner{ID: character}, items.CharacterOwner(character)},
		{"explicit room kept", items.RoomOwner("newbie_east"), items.RoomOwner("newbie_east")},
		{"explicit character kept", items.CharacterOwner(character), items.CharacterOwner(character)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			item := createTestItemInstance()
			item.Owner = tc.given
			if err := manager.Items().CreateItemInstance(item); err != nil {
				t.Fatal(err)
			}
			if item.Owner != tc.want {
				t.Fatalf("created owner = %v, want %v", item.Owner, tc.want)
			}
			stored, err := manager.Items().GetItemInstance(item.ID)
			if err != nil || stored.Owner != tc.want {
				t.Fatalf("stored owner = %+v %v, want %v", stored, err, tc.want)
			}
		})
	}

	invalid := createTestItemInstance()
	invalid.Owner = items.Owner{Kind: "npc", ID: "newbie_west"}
	if err := manager.Items().CreateItemInstance(invalid); err == nil {
		t.Fatal("created item with invalid owner kind")
	}
	if _, err := manager.Items().GetItemInstance(invalid.ID); err == nil {
		t.Fatal("invalid owner kind was written")
	}
}
