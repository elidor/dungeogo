package integration

import (
	"context"
	"strings"
	"testing"

	"github.com/elidor/dungeogo/internal/testdb"
	"github.com/elidor/dungeogo/pkg/game"
	"github.com/elidor/dungeogo/pkg/game/items"
	"github.com/elidor/dungeogo/pkg/game/player"
	"github.com/elidor/dungeogo/pkg/persistence/postgres"
	"github.com/elidor/dungeogo/pkg/server"
	"github.com/elidor/dungeogo/pkg/testutil"
	"golang.org/x/crypto/bcrypt"
)

const starterItemSword = "90000000-0000-4000-8000-000000000001"

func inventoryOutput(s *locationSession) string {
	s.t.Helper()
	s.send("inventory")
	return s.until("> ")
}

func TestStarterItems_RoomInventoryAndServerRestart(t *testing.T) {
	_, url := testdb.New(t, true)
	manager, err := postgres.NewPostgreSQLRepositoryManager(url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { manager.Close() })
	hash, err := bcrypt.GenerateFromPassword([]byte("testpass123"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	p := player.NewPlayer("itemtraveler", "itemtraveler@example.com", string(hash))
	if err := manager.Players().CreatePlayer(p); err != nil {
		t.Fatal(err)
	}
	id := createPresenceCharacter(t, manager, p, "Traveler")
	handler := server.NewSessionHandler(manager, game.NewEngine(manager))
	actor := openLocationSession(t, handler)
	actor.login(p)
	out := actor.selectCharacter()
	if !strings.Contains(out, "Training Sword ["+starterItemSword+"]") || !strings.Contains(out, "Travel Lantern") || strings.Contains(out, "Smooth Stone") {
		t.Fatalf("spawn room items: %q", out)
	}
	if out := inventoryOutput(actor); !strings.Contains(out, "carrying nothing") {
		t.Fatalf("new inventory: %q", out)
	}
	// Exercise persistence through the repository; get/drop commands are issue #10.
	if moved, err := manager.Items().TransferItem(starterItemSword, items.RoomOwner("starting_room"), items.CharacterOwner(id)); err != nil || !moved {
		t.Fatalf("transfer starter sword: moved=%v err=%v", moved, err)
	}
	sword, err := manager.Items().GetItemInstance(starterItemSword)
	if err != nil {
		t.Fatal(err)
	}
	sword.CustomName = "Traveler's Blade"
	if err := manager.Items().UpdateItemInstance(sword); err != nil {
		t.Fatal(err)
	}
	armor := testutil.CreateTestItemInstance("leather_armor", items.CharacterOwner(id))
	if err := manager.Items().CreateItemInstance(armor); err != nil {
		t.Fatal(err)
	}
	if out := roomLook(actor); strings.Contains(out, "Training Sword") || strings.Contains(out, "Traveler's Blade") || !strings.Contains(out, "Travel Lantern") {
		t.Fatalf("held item remained in room: %q", out)
	}
	out = inventoryOutput(actor)
	if !strings.Contains(out, "Traveler's Blade ["+starterItemSword+"]") || !strings.Contains(out, "Leather Armor ["+armor.ID+"]") || strings.Contains(out, "Travel Lantern") {
		t.Fatalf("persisted inventory names: %q", out)
	}
	actor.disconnect()
	if err := manager.Close(); err != nil {
		t.Fatal(err)
	}
	manager, err = postgres.NewPostgreSQLRepositoryManager(url)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.Bootstrap(context.Background(), "starting_room"); err != nil {
		t.Fatal(err)
	}
	restarted := server.NewSessionHandler(manager, game.NewEngine(manager))
	resumed := openLocationSession(t, restarted)
	resumed.login(p)
	out = resumed.selectCharacter()
	if strings.Contains(out, "Traveler's Blade") || !strings.Contains(out, "Travel Lantern") {
		t.Fatalf("restart reset ownership: %q", out)
	}
	out = inventoryOutput(resumed)
	if !strings.Contains(out, "Traveler's Blade") || !strings.Contains(out, "Leather Armor") {
		t.Fatalf("restart lost inventory: %q", out)
	}
	saved, err := manager.Items().GetItemInstance(starterItemSword)
	if err != nil || saved.Owner.Kind != items.OwnerCharacter || saved.Owner.ID != id {
		t.Fatalf("restart ownership: %+v %v", saved, err)
	}
	resumed.send("north")
	out = resumed.until("> ")
	if !strings.Contains(out, "Smooth Stone") || strings.Contains(out, "Travel Lantern") || strings.Contains(out, "Traveler's Blade") {
		t.Fatalf("destination item scope: %q", out)
	}
	if _, err := manager.GetDB().Exec(`ALTER TABLE item_instances RENAME TO unavailable_items`); err != nil {
		t.Fatal(err)
	}
	if out := roomLook(resumed); !strings.Contains(out, "Room contents are unavailable.") {
		t.Fatalf("failed room read fabricated empty contents: %q", out)
	}
	if out := inventoryOutput(resumed); !strings.Contains(out, "Error retrieving inventory.") {
		t.Fatalf("failed inventory read fabricated empty inventory: %q", out)
	}
	if _, err := manager.GetDB().Exec(`ALTER TABLE unavailable_items RENAME TO item_instances`); err != nil {
		t.Fatal(err)
	}
}
