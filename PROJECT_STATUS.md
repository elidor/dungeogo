# DungeoGo project status

Last updated 2026-10-28, at the end of **milestone 1: a persistent multiplayer starter zone** in one Go server process backed by PostgreSQL (issues #3–#13, plus review follow-ups #23–#30).

"Working" below means covered by tests against a real PostgreSQL database and exercised in the [recorded two-player walkthrough](docs/milestone-walkthrough.md). Nothing here has been load-tested or hardened for public deployment.

## Working

| Capability | What players can do | Verified by |
|---|---|---|
| Accounts | Create an account (username, email, password), log in again later | `TestPresenceSession`, walkthrough |
| Characters | Create (Human/Elf/Dwarf, Warrior/Mage/Rogue), select and delete by number; deleting destroys carried items (ADR-0001) | `TestServerIntegration_GuidedSelectAndDeleteByNumber`, `TestDeletingCharacterDestroysOnlyItsItems` |
| Starter zone | Nine connected rooms seeded by migrations; startup rejects a missing spawn or broken exits | `TestBootstrapFreshAndRepeated`, `TestBootstrapRejectsInvalidWorld`, migration upgrade tests |
| Movement | `north`/`south`/… with arrival and departure messages; characters return to the room they left | `TestSessionLocation_CreateMoveAndReconnect`, `TestRoomPresence_LookMovementScopeAndReplacement` |
| Presence | `who`, and `look` showing who's here; one live session per account, a new login replaces the old | `TestPresenceSession`, `pkg/server` registry tests |
| Speech | `say` and `emote` to the room, `tell` to anyone online | `TestLocalCommunication_SayEmoteMovementAndDisconnect`, `TestPrivateTell` |
| Items | `inventory`, `get`/`take`, `drop`, `give <item> <character>`, with targets like `2.sword` in arrival order; ownership is durable; simultaneous grabs can't duplicate (ADR-0002) | `TestItemActions_*`, `TestGive_*`, `TestCompetingTransfersExactlyOneWins` |
| Persistence | Locations and item ownership survive reconnects and server restarts; seeded items never respawn | `TestStarterItems_RoomInventoryAndServerRestart`, `TestItemActions_OwnershipSurvivesServerRestart`, walkthrough |
| Sessions | `quit`, dropped connections and idle expiry release presence; stalled clients time out after 5 seconds | `TestLifecycle_IdleExpiryRemovesPresenceAndShutdownWaits`, `pkg/server` tests |
| Shutdown | SIGINT/SIGTERM stops accepting, closes sessions, waits for running commands, closes the database, and exits nonzero on failure | `TestShutdown*`, `cmd/server` tests, walkthrough |
| Honest commands | `help` and `commands` list only what works; everything else says "isn't available yet" | `TestUnavailableCommandsSaySo`, `TestEveryParsedCommandIsAvailableOrSaysItIsnt` |

## Partial

- **Character stats, skills and experience** are stored and shown by `score`, but nothing in play changes them.
- **The item model** has durability, enchantments, stacking and modifications, but gameplay ignores them. Stacks only show a count, e.g. `Health Potion x3`.
- **`look`** describes the room only; looking at a particular thing isn't available.
- **`save`** records play time; everything else is already saved as it happens.

## Not available yet

These commands are recognised but reply that they aren't available: `yell`, `whisper`, `chat`, `wear`, `remove`, `kill`, `flee`, `defend`, `cast`, `prepare`, `examine`, `time`, `weather`, `skills`, `practice`, `smile`, `wave`, `bow`.

## Deferred backlog

Not part of milestone 1 and not implemented:

- Equipment and its effects (`wear`, `remove`)
- NPCs and combat (`kill`, `flee`, `defend`)
- Progression: skills, practice, levelling
- Spells and quests
- Item stacking and respawning
- Offline messages; `yell`, `whisper` and a global `chat` channel
- Web or HTTP access
- Guilds, PvP, crafting
- Running more than one server process (presence lives in one process's memory)

## Known issues

- `emote` echoes the action back to the actor with the third-person verb: `emote waves` shows you "You waves".
- Articles are chosen by spelling, so a name like "Unique Sword" would read "an Unique Sword".
- "Someone got to the … first." only appears if something outside the game moves an item mid-action. Within one server process, the slower player instead sees that the item isn't there.
- The connection limit (100) and idle timeout (30 minutes) are fixed in `cmd/server/main.go` rather than configurable.
- Character names must be 3–20 letters or numbers.

## How this is verified

```bash
# Everything, failing if PostgreSQL isn't available
TEST_DATABASE_URL='postgres://testuser:testpass@localhost:5433/postgres?sslmode=disable' \
  DUNGEOGO_REQUIRE_DB=1 go test -race -count=1 ./...

# The two-player walkthrough against a built server and an empty database
go build -o bin/server ./cmd/server
python3 scripts/walkthrough.py bin/server "$DATABASE_URL" walkthrough.md server.log
```

[TESTING.md](TESTING.md) describes each test group. Architecture decisions are in [docs/adr/](docs/adr/), and domain terms in [CONTEXT.md](CONTEXT.md).
