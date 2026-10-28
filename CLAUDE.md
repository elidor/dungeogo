# CLAUDE.md

DungeoGo is a MUD server in Go: players connect over TCP, and all game state lives in PostgreSQL. What works today and what doesn't is in `PROJECT_STATUS.md`; read it before claiming or documenting a capability. Use the domain terms in `CONTEXT.md` (Player, Character, Session, Room, Item, Owner, Item target, …) in code, tests, issues and docs, and read the ADRs in `docs/adr/` before changing an area they cover.

## Architecture

- `cmd/server` wires config, PostgreSQL, the game engine and the TCP server, and owns startup and ordered shutdown.
- `pkg/game/actions` is where gameplay happens. Every action runs under the `World` action lock shared with movement, reads durable state, and returns a `world.Result` of addressed messages: to a Character, to a Room (excluding some Characters), or pinned to a Session. Actions never touch sockets.
- `pkg/server` owns connections and Sessions. It resolves a Result's audiences through the `SessionRegistry` and delivers them. A failed delivery never rolls back committed game state.
- `pkg/persistence` holds repository interfaces plus their PostgreSQL implementation. `pkg/game/items`, `character`, `player` and `world` hold the domain types.
- `pkg/commands` parses input and routes verbs. Verbs the game doesn't implement are listed in `notYetAvailable` and answer "isn't available yet". `help` and `commands` are built from `availableCommands`, so add a working verb there and remove it from `notYetAvailable` in the same change.

## Conventions

- **Item ownership changes only through `TransferItem(item, from, to)`**, the conditional write from ADR-0002. Treat `moved == false` as a normal outcome. Owners always name their kind: `items.RoomOwner(id)` or `items.CharacterOwner(id)`.
- **Report success after the durable write commits**, and build the success messages from what was committed.
- **Migrations are append-only.** Add the next numbered `migrations/NNN_*.sql` file; applied migrations never re-run, so editing one changes nothing for existing databases.
- **Database tests must actually run.** `DUNGEOGO_REQUIRE_DB=1` turns an unavailable PostgreSQL into a failure instead of a skip; each test gets a disposable database migrated with the real migrations (`internal/testdb`). The bar for a change is `DUNGEOGO_REQUIRE_DB=1 go test -race -count=1 ./...` passing, with `TEST_DATABASE_URL` pointing at a PostgreSQL role that can create databases. `TESTING.md` maps each test group to what it covers.
- **Player-visible behaviour gets a TCP test** in `pkg/integration`, driving real sessions; concurrency claims get a test that races real clients.
- **Commands only claim what happened.** A reply never describes an effect that wasn't made.
- `scripts/walkthrough.py` replays a two-player session against a built server; rerun it and refresh `docs/milestone-walkthrough.md` when player-facing behaviour changes.

## Configuration

Environment variables, or a `.env` file in the working directory: `DATABASE_URL` (required), `PORT` (default 8080), `BIND_ADDRESS` (default localhost), `STARTING_ROOM` (default `starting_room`). `make help` lists build, test and container targets.

## Agent skills

### Issue tracker

Issues are tracked in GitHub Issues on elidor/dungeogo (via `gh`). See `docs/agents/issue-tracker.md`.

### Triage labels

Default vocabulary: needs-triage, needs-info, ready-for-agent, ready-for-human, wontfix. See `docs/agents/triage-labels.md`.

### Domain docs

Single-context: one `CONTEXT.md` + `docs/adr/` at the repo root. See `docs/agents/domain.md`.
