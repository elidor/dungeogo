package integration

import (
	"strings"
	"sync"
	"testing"

	"github.com/elidor/dungeogo/pkg/game/items"
	"github.com/elidor/dungeogo/pkg/persistence/postgres"
	"github.com/elidor/dungeogo/pkg/testutil"
)

// command sends one line and returns the output up to the next prompt.
func command(s *locationSession, input string) string {
	s.t.Helper()
	s.send(input)
	return s.until("> ")
}

func roomItem(t *testing.T, manager *postgres.PostgreSQLRepositoryManager, room, template, name string) *items.ItemInstance {
	t.Helper()
	item := testutil.CreateTestItemInstance(template, items.RoomOwner(room))
	item.CustomName = name
	if err := manager.Items().CreateItemInstance(item); err != nil {
		t.Fatal(err)
	}
	return item
}

func assertOwner(t *testing.T, manager *postgres.PostgreSQLRepositoryManager, itemID string, want items.Owner) {
	t.Helper()
	item, err := manager.Items().GetItemInstance(itemID)
	if err != nil || item.Owner != want {
		t.Fatalf("item %s owner = %+v %v, want %v", itemID, item, err, want)
	}
}

func assertContains(t *testing.T, out string, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(out, w) {
			t.Fatalf("missing %q in %q", w, out)
		}
	}
}

func TestItemActions_GetDropTargetsMessagesAndPersistence(t *testing.T) {
	manager, p, handler := locationFixture(t)
	actorID := createPresenceCharacter(t, manager, p, "Traveler")
	actor := openLocationSession(t, handler)
	actor.login(p)
	actor.selectCharacter()
	home := roomObserver(t, manager, handler, "HomePeer", "starting_room")
	away := roomObserver(t, manager, handler, "NorthPeer", "newbie_north")
	carried, start, north := items.CharacterOwner(actorID), items.RoomOwner("starting_room"), items.RoomOwner("newbie_north")

	// Pick up: the actor and the Room each get their own wording; other Rooms nothing.
	assertContains(t, command(actor, "get sword"), "You pick up a Training Sword.")
	home.until("Traveler picks up a Training Sword.")
	assertNoLocalMessage(t, away, "picks up")
	assertOwner(t, manager, starterItemSword, carried)
	assertItemLines(t, command(actor, "inventory"), []string{"Training Sword"})
	if out := roomLook(actor); strings.Contains(out, "Training Sword") {
		t.Fatalf("picked-up item still in room: %q", out)
	}
	assertContains(t, command(actor, "get sword"), "You don't see a 'sword' here.")

	// "take" is get; drop mirrors it.
	assertContains(t, command(actor, "take lan"), "You pick up a Travel Lantern.")
	home.until("Traveler picks up a Travel Lantern.")
	assertContains(t, command(actor, "drop lantern"), "You drop a Travel Lantern.")
	home.until("Traveler drops a Travel Lantern.")
	assertOwner(t, manager, starterItemLantern, start)

	// Numbered targets count matches in arrival order, the order look shows.
	older := roomItem(t, manager, "starting_room", "rusty_sword", "Rusty Sword")
	newer := roomItem(t, manager, "starting_room", "rusty_sword", "Rusty Sword")
	if out := roomLook(actor); strings.Index(out, "Travel Lantern") < 0 || strings.Count(out, "  Rusty Sword") != 2 {
		t.Fatalf("room listing: %q", out)
	}
	assertContains(t, command(actor, "get 2.rusty"), "You pick up a Rusty Sword.")
	home.until("Traveler picks up a Rusty Sword.")
	assertOwner(t, manager, newer.ID, carried)
	assertOwner(t, manager, older.ID, start)
	// Inventory now holds Training Sword then Rusty Sword: 2.sword is the rusty one.
	assertContains(t, command(actor, "drop 2.sword"), "You drop a Rusty Sword.")
	home.until("Traveler drops a Rusty Sword.")
	assertOwner(t, manager, newer.ID, start)

	// Failures change nothing and say why.
	for input, want := range map[string]string{
		"get 5.sword":  "You don't see a '5.sword' here.",
		"get axe":      "You don't see a 'axe' here.",
		"get 0.sword":  "Get what?",
		"drop stone":   "You aren't carrying a 'stone'.",
		"drop 2.sword": "You aren't carrying a '2.sword'.",
		"drop 0.x":     "Drop what?",
	} {
		assertContains(t, command(actor, input), want)
	}
	assertOwner(t, manager, starterItemSword, carried)
	assertOwner(t, manager, older.ID, start)

	// Moving changes which Room a drop lands in and who hears it.
	command(actor, "north")
	assertContains(t, command(actor, "drop sword"), "You drop a Training Sword.")
	away.until("Traveler drops a Training Sword.")
	assertNoLocalMessage(t, home, "Traveler drops a Training Sword.")
	assertOwner(t, manager, starterItemSword, north)
	assertContains(t, command(actor, "get stone"), "You pick up a Smooth Stone.")

	// Ownership survives reconnecting.
	actor.disconnect()
	resumed := openLocationSession(t, handler)
	resumed.login(p)
	out := resumed.selectCharacter()
	assertItemLines(t, out, []string{"Training Sword"})
	assertItemLines(t, command(resumed, "inventory"), []string{"Smooth Stone"})
}

func TestItemActions_CompetingPickupsExactlyOneWins(t *testing.T) {
	manager, _, handler := locationFixture(t)
	first := roomObserver(t, manager, handler, "Ana", "starting_room")
	second := roomObserver(t, manager, handler, "Bo", "starting_room")
	contenders := []*locationSession{first, second}

	for round := 0; round < 10; round++ {
		gem := roomItem(t, manager, "starting_room", "smooth_stone", "Contested Gem")
		outputs := make([]string, len(contenders))
		var start, done sync.WaitGroup
		start.Add(1)
		for i, s := range contenders {
			done.Add(1)
			go func(i int, s *locationSession) {
				defer done.Done()
				start.Wait()
				s.send("get gem")
			}(i, s)
		}
		start.Done()
		done.Wait()
		for i, s := range contenders {
			outputs[i] = s.until("> ")
		}
		winners := 0
		for i, out := range outputs {
			won := strings.Contains(out, "You pick up a Contested Gem.")
			lost := strings.Contains(out, "You don't see a 'gem' here.") || strings.Contains(out, "Someone got to the Contested Gem first.")
			if won == lost {
				t.Fatalf("round %d: contender %d output %q is neither a clear win nor a clear loss", round, i, out)
			}
			if won {
				winners++
			}
		}
		if winners != 1 {
			t.Fatalf("round %d: %d pickups succeeded, want exactly 1: %q", round, winners, outputs)
		}
		item, err := manager.Items().GetItemInstance(gem.ID)
		if err != nil || item.Owner.Kind != items.OwnerCharacter {
			t.Fatalf("round %d: gem owner %+v %v", round, item, err)
		}
	}
}
