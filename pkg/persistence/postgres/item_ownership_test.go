package postgres

import (
	"context"
	"sync"
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
	if moved, err := manager.Items().TransferItem(starterSword, items.RoomOwner("starting_room"), items.CharacterOwner(owner)); err != nil || !moved {
		t.Fatalf("transfer starter sword: moved=%v err=%v", moved, err)
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
	roomItem.Owner = items.RoomOwner(id)
	charItem := createTestItemInstance()
	charItem.Owner = items.CharacterOwner(id)
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
		t.Fatal("created item whose owner kind was left to guesswork")
	}
	for _, tc := range []struct{ kind, owner string }{
		{"room", "missing_room"}, {"character", "00000000-0000-0000-0000-000000000099"}, {"character", "not-a-uuid"}, {"unknown", "starting_room"},
	} {
		if _, err := manager.GetDB().Exec(`INSERT INTO item_instances(template_id,owner_kind,owner_id) VALUES ('invalid',$1,$2)`, tc.kind, tc.owner); err == nil {
			t.Fatalf("accepted dangling/invalid owner %+v", tc)
		}
	}
	if _, err := manager.GetDB().Exec(`DELETE FROM room_definitions WHERE room_id=$1`, id); err == nil {
		t.Fatal("deleted room while retaining owned item")
	}
	if _, err := manager.Items().TransferItem(charItem.ID, items.CharacterOwner(id), items.RoomOwner("missing_room")); err == nil {
		t.Fatal("transferred to nonexistent owner")
	}
	unchanged, err := manager.Items().GetItemInstance(charItem.ID)
	if err != nil || unchanged.Owner.Kind != items.OwnerCharacter || unchanged.Owner.ID != id {
		t.Fatalf("failed transfer changed ownership: %+v %v", unchanged, err)
	}
}

func TestItemOwnerMustBeExplicit(t *testing.T) {
	manager := setupTestDB(t)
	character := existingItemCharacter(t, manager)
	for _, owner := range []items.Owner{items.RoomOwner("newbie_east"), items.CharacterOwner(character)} {
		item := createTestItemInstance()
		item.Owner = owner
		if err := manager.Items().CreateItemInstance(item); err != nil {
			t.Fatalf("create with %v: %v", owner, err)
		}
		if stored, err := manager.Items().GetItemInstance(item.ID); err != nil || stored.Owner != owner {
			t.Fatalf("stored owner = %+v %v, want %v", stored, err, owner)
		}
	}
	for _, owner := range []items.Owner{{ID: "newbie_west"}, {ID: character}, {Kind: "npc", ID: "newbie_west"}, {Kind: items.OwnerRoom}} {
		item := createTestItemInstance()
		item.Owner = owner
		if err := manager.Items().CreateItemInstance(item); err == nil {
			t.Fatalf("created item with %v", owner)
		}
		if _, err := manager.Items().GetItemInstance(item.ID); err == nil {
			t.Fatalf("item with %v was written", owner)
		}
		if _, err := manager.Items().TransferItem(starterSword, items.RoomOwner("starting_room"), owner); err == nil {
			t.Fatalf("transferred to %v", owner)
		}
		if _, err := manager.Items().TransferItem(starterSword, owner, items.RoomOwner("newbie_west")); err == nil {
			t.Fatalf("transferred from %v", owner)
		}
	}
}

func TestTransferItemOnlyFromExpectedOwner(t *testing.T) {
	manager := setupTestDB(t)
	character := existingItemCharacter(t, manager)
	repo := manager.Items()
	start, carried := items.RoomOwner("starting_room"), items.CharacterOwner(character)

	if moved, err := repo.TransferItem(starterSword, items.RoomOwner("newbie_west"), carried); err != nil || moved {
		t.Fatalf("transfer from wrong owner: moved=%v err=%v", moved, err)
	}
	if moved, err := repo.TransferItem("90000000-0000-4000-8000-0000000000ff", start, carried); err != nil || moved {
		t.Fatalf("transfer of missing item: moved=%v err=%v", moved, err)
	}
	if item, err := repo.GetItemInstance(starterSword); err != nil || item.Owner != start {
		t.Fatalf("failed transfers changed ownership: %+v %v", item, err)
	}
	if moved, err := repo.TransferItem(starterSword, start, carried); err != nil || !moved {
		t.Fatalf("transfer from expected owner: moved=%v err=%v", moved, err)
	}
	// A second mover still holding the old view of the sword loses.
	if moved, err := repo.TransferItem(starterSword, start, items.RoomOwner("newbie_west")); err != nil || moved {
		t.Fatalf("stale transfer: moved=%v err=%v", moved, err)
	}
	if item, err := repo.GetItemInstance(starterSword); err != nil || item.Owner != carried {
		t.Fatalf("owner after stale transfer = %+v %v, want %v", item, err, carried)
	}
}

func TestCompetingTransfersExactlyOneWins(t *testing.T) {
	manager := setupTestDB(t)
	repo := manager.Items()
	start := items.RoomOwner("starting_room")
	for round := 0; round < 20; round++ {
		contenders := []items.Owner{items.CharacterOwner(existingItemCharacter(t, manager)), items.CharacterOwner(existingItemCharacter(t, manager))}
		results := make(chan bool, len(contenders))
		errs := make(chan error, len(contenders))
		var ready, done sync.WaitGroup
		release := make(chan struct{})
		for _, to := range contenders {
			ready.Add(1)
			done.Add(1)
			go func(to items.Owner) {
				defer done.Done()
				ready.Done()
				<-release
				moved, err := repo.TransferItem(starterSword, start, to)
				results <- moved
				errs <- err
			}(to)
		}
		ready.Wait()
		close(release)
		done.Wait()
		close(results)
		close(errs)
		for err := range errs {
			if err != nil {
				t.Fatal(err)
			}
		}
		winners := 0
		for moved := range results {
			if moved {
				winners++
			}
		}
		if winners != 1 {
			t.Fatalf("round %d: %d transfers won, want exactly 1", round, winners)
		}
		item, err := repo.GetItemInstance(starterSword)
		if err != nil || (item.Owner != contenders[0] && item.Owner != contenders[1]) {
			t.Fatalf("round %d: sword owner = %+v %v", round, item, err)
		}
		if moved, err := repo.TransferItem(starterSword, item.Owner, start); err != nil || !moved {
			t.Fatalf("round %d: reset: moved=%v err=%v", round, moved, err)
		}
	}
}

func TestDeletingCharacterDestroysOnlyItsItems(t *testing.T) {
	manager := setupTestDB(t)
	doomed := existingItemCharacter(t, manager)
	survivor := existingItemCharacter(t, manager)
	empty := existingItemCharacter(t, manager)

	held := []*items.ItemInstance{createTestItemInstance(), createTestItemInstance()}
	for _, item := range held {
		item.Owner = items.CharacterOwner(doomed)
	}
	kept := createTestItemInstance()
	kept.Owner = items.CharacterOwner(survivor)
	for _, item := range append(held, kept) {
		if err := manager.Items().CreateItemInstance(item); err != nil {
			t.Fatal(err)
		}
	}
	roomItems, err := manager.Items().GetRoomItems("starting_room")
	if err != nil {
		t.Fatal(err)
	}

	for _, id := range []string{doomed, empty} {
		if err := manager.Characters().DeleteCharacter(id); err != nil {
			t.Fatalf("delete character %s: %v", id, err)
		}
		if _, err := manager.Characters().GetCharacter(id); err == nil {
			t.Fatalf("character %s still exists", id)
		}
	}
	for _, item := range held {
		if _, err := manager.Items().GetItemInstance(item.ID); err == nil {
			t.Fatalf("item %s outlived its deleted character", item.ID)
		}
	}
	if item, err := manager.Items().GetItemInstance(kept.ID); err != nil || item.Owner != kept.Owner {
		t.Fatalf("another character's item was affected: %+v %v", item, err)
	}
	after, err := manager.Items().GetRoomItems("starting_room")
	if err != nil || len(after) != len(roomItems) {
		t.Fatalf("room items changed: before %d, after %d (%v)", len(roomItems), len(after), err)
	}
	// Room ownership still restricts deletion: a room holding items can't vanish.
	if _, err := manager.GetDB().Exec(`DELETE FROM room_definitions WHERE room_id='starting_room'`); err == nil {
		t.Fatal("deleted a room that holds items")
	}
}
