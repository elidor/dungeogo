package integration

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/elidor/dungeogo/internal/testdb"
	"github.com/elidor/dungeogo/pkg/game"
	"github.com/elidor/dungeogo/pkg/game/character"
	"github.com/elidor/dungeogo/pkg/game/items"
	"github.com/elidor/dungeogo/pkg/game/player"
	"github.com/elidor/dungeogo/pkg/persistence/postgres"
	"github.com/elidor/dungeogo/pkg/server"
	"golang.org/x/crypto/bcrypt"
)

// giveWorld is a migrated, bootstrapped database and a server handler on it,
// kept with its URL so a test can restart the server against the same data.
type giveWorld struct {
	url     string
	manager *postgres.PostgreSQLRepositoryManager
	handler *server.SessionHandler
}

func openGiveWorld(t *testing.T, url string) giveWorld {
	t.Helper()
	manager, err := postgres.NewPostgreSQLRepositoryManager(url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { manager.Close() })
	if err := manager.Bootstrap(context.Background(), "starting_room"); err != nil {
		t.Fatal(err)
	}
	return giveWorld{url, manager, server.NewSessionHandler(manager, game.NewEngine(manager))}
}

func newGiveWorld(t *testing.T) giveWorld {
	t.Helper()
	_, url := testdb.New(t, true)
	return openGiveWorld(t, url)
}

// party is a Player and their Character, logged in over TCP.
type party struct {
	*locationSession
	player *player.Player
	id     string
}

func (w giveWorld) join(t *testing.T, name, roomID string) party {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte("testpass123"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	p := player.NewPlayer(strings.ToLower(name), strings.ToLower(name)+"@example.com", string(hash))
	if err := w.manager.Players().CreatePlayer(p); err != nil {
		t.Fatal(err)
	}
	id := createPresenceCharacter(t, w.manager, p, name)
	if err := w.manager.Characters().UpdateCharacterLocation(id, &character.Location{RoomID: roomID, ZoneID: "newbie_zone"}); err != nil {
		t.Fatal(err)
	}
	return w.rejoin(t, party{player: p, id: id})
}

func (w giveWorld) rejoin(t *testing.T, p party) party {
	t.Helper()
	s := openLocationSession(t, w.handler)
	s.login(p.player)
	s.selectCharacter()
	return party{s, p.player, p.id}
}

func heldItem(t *testing.T, manager *postgres.PostgreSQLRepositoryManager, characterID, template, name string) *items.ItemInstance {
	t.Helper()
	item := roomItem(t, manager, "starting_room", template, name)
	if moved, err := manager.Items().TransferItem(item.ID, items.RoomOwner("starting_room"), items.CharacterOwner(characterID)); err != nil || !moved {
		t.Fatalf("hand %s to %s: moved=%v err=%v", name, characterID, moved, err)
	}
	return item
}

func TestGive_ExchangeMessagesRejectionsAndRestart(t *testing.T) {
	w := newGiveWorld(t)
	ana := w.join(t, "Ana", "starting_room")
	bo := w.join(t, "Bo", "starting_room")
	cy := w.join(t, "Cy", "starting_room")
	dee := w.join(t, "Dee", "newbie_north")

	// A successful give has three wordings, one per audience; other Rooms hear nothing.
	assertContains(t, command(ana.locationSession, "get sword"), "You pick up a Training Sword.")
	assertContains(t, command(ana.locationSession, "give sword Bo"), "You give a Training Sword to Bo.")
	bo.until("Ana gives you a Training Sword.")
	cy.until("Ana gives a Training Sword to Bo.")
	assertNoLocalMessage(t, bo.locationSession, "Ana gives a Training Sword to Bo.")
	assertNoLocalMessage(t, ana.locationSession, "Ana gives a Training Sword to Bo.")
	assertNoLocalMessage(t, dee.locationSession, "gives")
	assertOwner(t, w.manager, starterItemSword, items.CharacterOwner(bo.id))
	assertItemLines(t, command(bo.locationSession, "inventory"), []string{"Training Sword"})

	// Recipient names ignore case; numbered targets count the giver's inventory.
	first := heldItem(t, w.manager, ana.id, "rusty_sword", "Rusty Sword")
	second := heldItem(t, w.manager, ana.id, "rusty_sword", "Rusty Sword")
	assertContains(t, command(ana.locationSession, "give 2.rusty bo"), "You give a Rusty Sword to Bo.")
	bo.until("Ana gives you a Rusty Sword.")
	assertOwner(t, w.manager, second.ID, items.CharacterOwner(bo.id))
	assertOwner(t, w.manager, first.ID, items.CharacterOwner(ana.id))

	// Rejections say why and change nothing.
	for input, want := range map[string]string{
		"give axe Bo":      "You aren't carrying a 'axe'.",
		"give 2.rusty Bo":  "You aren't carrying a '2.rusty'.",
		"give rusty Dee":   "There's no one called 'Dee' here.",
		"give rusty Zed":   "There's no one called 'Zed' here.",
		"give rusty Ana":   "You can't give something to yourself.",
		"give rusty to Bo": "Give what to whom?",
		"give rusty":       "Give what to whom?",
		"give 0.rusty Bo":  "Give what to whom?",
	} {
		assertContains(t, command(ana.locationSession, input), want)
	}
	assertOwner(t, w.manager, first.ID, items.CharacterOwner(ana.id))
	assertNoLocalMessage(t, bo.locationSession, "Ana gives you")

	// An offline recipient isn't there to receive.
	bo.disconnect()
	assertContains(t, command(ana.locationSession, "give rusty Bo"), "There's no one called 'Bo' here.")
	assertOwner(t, w.manager, first.ID, items.CharacterOwner(ana.id))

	// Ownership survives a full restart.
	for _, p := range []party{ana, cy, dee} {
		p.disconnect()
	}
	if err := w.manager.Close(); err != nil {
		t.Fatal(err)
	}
	restarted := openGiveWorld(t, w.url)
	back := restarted.rejoin(t, bo)
	assertItemLines(t, command(back.locationSession, "inventory"), []string{"Training Sword", "Rusty Sword"})
	ana = restarted.rejoin(t, ana)
	assertItemLines(t, command(ana.locationSession, "inventory"), []string{"Rusty Sword"})
}

// TestGive_RacesWithRecipientMovingOrLeaving runs a give at the same moment the
// recipient walks out or disconnects. Either order may win, but the outcome
// must be consistent: the giver is told "You give" exactly when the item now
// belongs to the recipient, and otherwise it stays with the giver.
func TestGive_RacesWithRecipientMovingOrLeaving(t *testing.T) {
	w := newGiveWorld(t)
	ana := w.join(t, "Ana", "starting_room")
	bo := w.join(t, "Bo", "starting_room")

	// Each round gives its own gem: refused gems stay with Ana, and a shared
	// keyword would pick the oldest of them instead of this round's.
	rounds := 0
	race := func(recipientAction func()) {
		t.Helper()
		rounds++
		round := rounds
		gem := heldItem(t, w.manager, ana.id, "smooth_stone", fmt.Sprintf("Gem%02d", round))
		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); ana.send(fmt.Sprintf("give gem%02d Bo", round)) }()
		go func() { defer wg.Done(); recipientAction() }()
		wg.Wait()
		out := ana.until("> ")
		gave := strings.Contains(out, fmt.Sprintf("You give a Gem%02d to Bo.", round))
		refused := strings.Contains(out, "There's no one called 'Bo' here.")
		if gave == refused {
			t.Fatalf("round %d: give output %q is neither a clear success nor a clear refusal", round, out)
		}
		want := items.CharacterOwner(ana.id)
		if gave {
			want = items.CharacterOwner(bo.id)
		}
		assertOwner(t, w.manager, gem.ID, want)
	}

	// Bo walks north and back while Ana gives.
	for round := 0; round < 8; round++ {
		direction := "north"
		if round%2 == 1 {
			direction = "south"
		}
		race(func() { bo.send(direction) })
		bo.until("> ")
	}

	// Bo disconnects while Ana gives, then logs back in for the next round.
	for i := 0; i < 4; i++ {
		race(func() { bo.disconnect() })
		bo = w.rejoin(t, bo)
	}
}
