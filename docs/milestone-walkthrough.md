# Milestone 1 walkthrough: two players in the starter zone

Recorded 2026-10-28 against the server built from the milestone-completion branch, on an empty PostgreSQL 16 database.

Two clients connect over TCP the way `telnet` would. Ana and Bram each create an account and a character. They:

- explore and talk (`who`, `say`, `tell`, movement)
- exchange items (`get`, `give`, `drop`)
- try commands outside the milestone
- disconnect and reconnect

Then the server is stopped with SIGTERM, started again, and both log back in to check that locations and item ownership survived.

The session was driven by [`scripts/walkthrough.py`](../scripts/walkthrough.py) rather than typed by hand, so it can be repeated exactly:

```bash
go build -o bin/server ./cmd/server
createdb walkthrough   # must be empty
python3 scripts/walkthrough.py bin/server "postgres://…/walkthrough" walkthrough.md server.log
```

Every line each client sent (`→`) and received (`←`) is below; passwords are masked. Lines starting with PASS/FAIL are the script's checks on what it just saw.

Checks: 37/37 passed.

- PASS: Ana's client: account created
- PASS: Ana's client: character Ana created
- PASS: Bram's client: account created
- PASS: Bram's client: character Bram created
- PASS: Ana's client: enters the world as Ana
- PASS: Ana sees the starter items in the starting room
- PASS: Bram's client: enters the world as Bram
- PASS: who lists both characters
- PASS: Bram hears Ana in the same room
- PASS: Ana receives Bram's tell
- PASS: Bram walks north and sees the Smooth Stone
- PASS: Ana sees Bram leave
- PASS: Ana picks up the Training Sword
- PASS: Bram sees Ana arrive
- PASS: Ana gives the sword to Bram
- PASS: Bram is told he received it
- PASS: Bram's inventory shows the sword
- PASS: the 'give <item> to <character>' form is refused with usage
- PASS: Bram picks up the Smooth Stone
- PASS: Ana sees Bram pick it up
- PASS: Bram drops the stone again
- PASS: wear is reported as not available
- PASS: kill is reported as not available
- PASS: commands lists what isn't available
- PASS: Ana is gone from who after quitting
- PASS: Ana's client: logs back in
- PASS: Ana's client: enters the world as Ana
- PASS: Ana is restored to the starting room
- PASS: server exits cleanly on SIGTERM (exit code 0)
- PASS: Bram's client: logs back in
- PASS: Bram's client: enters the world as Bram
- PASS: after restart Bram is back in the northern room with the stone on the floor
- PASS: after restart Bram still carries the Training Sword
- PASS: Ana's client: logs back in
- PASS: Ana's client: enters the world as Ana
- PASS: after restart the starting room no longer has the sword (no respawn)
- PASS: server exits cleanly again (exit code 0)

## Transcript


### Server started against an empty database; migrations seed the starter zone.

**Ana's client ←**

```text
Welcome to DungeoGo!
Please enter your username:
> 
```

**Ana's client →** `ana`

**Ana's client ←**

```text
New player! Creating account for: ana
Please enter your email address:
Email: 
```

**Ana's client →** `ana@example.com`

**Ana's client ←**

```text
Please choose a password (minimum 6 characters):
```

**Ana's client →** `******`

**Ana's client ←**

```text


Please confirm your password
```

**Ana's client →** `******`

**Ana's client ←**

```text
:

Account created successfully! Welcome to DungeoGo, ana!

--- Character Selection ---
Commands:
  list (l)                 - List your characters
  select (s)               - Select character by number
  create (c)               - Guided character creation
  delete (d)               - Delete character by number
  quit (q)                 - Disconnect

Character> 
```

> PASS Ana's client: account created

**Ana's client →** `create`

**Ana's client ←**

```text

--- Character Creation ---
Type 'cancel' at any time to return to the character menu.
Enter character name (3-20 letters/numbers):
Name: 
```

**Ana's client →** `Ana`

**Ana's client ←**

```text
Choose race: human, elf, dwarf
Race: 
```

**Ana's client →** `human`

**Ana's client ←**

```text
Choose class: warrior, mage, rogue
Class: 
```

**Ana's client →** `warrior`

**Ana's client ←**

```text
Create character 'Ana' as human warrior? (yes/no)
Confirm: 
```

**Ana's client →** `yes`

**Ana's client ←**

```text
Character 'Ana' created successfully!

--- Character Selection ---
Commands:
  list (l)                 - List your characters
  select (s)               - Select character by number
  create (c)               - Guided character creation
  delete (d)               - Delete character by number
  quit (q)                 - Disconnect

Character> 
```

> PASS Ana's client: character Ana created

**Bram's client ←**

```text
Welcome to DungeoGo!
Please enter your username:
> 
```

**Bram's client →** `bram`

**Bram's client ←**

```text
New player! Creating account for: bram
Please enter your email address:
Email: 
```

**Bram's client →** `bram@example.com`

**Bram's client ←**

```text
Please choose a password (minimum 6 characters):
```

**Bram's client →** `******`

**Bram's client ←**

```text


Please confirm your password
```

**Bram's client →** `******`

**Bram's client ←**

```text
:

Account created successfully! Welcome to DungeoGo, bram!

--- Character Selection ---
Commands:
  list (l)                 - List your characters
  select (s)               - Select character by number
  create (c)               - Guided character creation
  delete (d)               - Delete character by number
  quit (q)                 - Disconnect

Character> 
```

> PASS Bram's client: account created

**Bram's client →** `create`

**Bram's client ←**

```text

--- Character Creation ---
Type 'cancel' at any time to return to the character menu.
Enter character name (3-20 letters/numbers):
Name: 
```

**Bram's client →** `Bram`

**Bram's client ←**

```text
Choose race: human, elf, dwarf
Race: 
```

**Bram's client →** `elf`

**Bram's client ←**

```text
Choose class: warrior, mage, rogue
Class: 
```

**Bram's client →** `mage`

**Bram's client ←**

```text
Create character 'Bram' as elf mage? (yes/no)
Confirm: 
```

**Bram's client →** `yes`

**Bram's client ←**

```text
Character 'Bram' created successfully!

--- Character Selection ---
Commands:
  list (l)                 - List your characters
  select (s)               - Select character by number
  create (c)               - Guided character creation
  delete (d)               - Delete character by number
  quit (q)                 - Disconnect

Character> 
```

> PASS Bram's client: character Bram created

**Ana's client →** `select`

**Ana's client ←**

```text

--- Character Select ---
Type a character number, or 'cancel' to return.
No. Name           Race      Class     Level  Status    Last Played
--------------------------------------------------------------------
1   Ana            Human     Warrior   1      Alive     2026-10-28 18:49
Select #: 
```

**Ana's client →** `1`

**Ana's client ←**

```text
Welcome, Ana!
You enter the game world...
A Simple Room
You are in a basic room with stone walls and a dirt floor.
Exits: east, north, south, west.
Items here:
  Training Sword
  Travel Lantern
> 
```

> PASS Ana's client: enters the world as Ana

> PASS Ana sees the starter items in the starting room

**Bram's client →** `select`

**Bram's client ←**

```text

--- Character Select ---
Type a character number, or 'cancel' to return.
No. Name           Race      Class     Level  Status    Last Played
--------------------------------------------------------------------
1   Bram           Elf       Mage      1      Alive     2026-10-28 18:49
Select #: 
```

**Bram's client →** `1`

**Bram's client ←**

```text
Welcome, Bram!
You enter the game world...
A Simple Room
You are in a basic room with stone walls and a dirt floor.
Exits: east, north, south, west.
Also here: Ana.
Items here:
  Training Sword
  Travel Lantern
> 
```

> PASS Bram's client: enters the world as Bram


### Exploration and speech

**Ana's client →** `who`

**Ana's client ←**

```text
Players currently online:
  Ana (Human Warrior, Level 1)
  Bram (Elf Mage, Level 1)

2 players online.
> 
```

> PASS who lists both characters

**Ana's client →** `say Hello, Bram!`

**Ana's client ←**

```text
You say: Hello, Bram!
> 
```

**Bram's client ←**

```text
Ana says: Hello, Bram!
```

> PASS Bram hears Ana in the same room

**Bram's client →** `tell Ana Meet me up north.`

**Bram's client ←**

```text
You tell Ana: Meet me up north.
> 
```

**Ana's client ←**

```text
Bram tells you: Meet me up north.
```

> PASS Ana receives Bram's tell

**Bram's client →** `north`

**Bram's client ←**

```text
You move north.
Northern Walk
A worn path runs along the northern edge of the newbie zone.
Exits: east, south, west.
Items here:
  Smooth Stone
> 
```

> PASS Bram walks north and sees the Smooth Stone

**Ana's client ←**

```text
Bram leaves north.
```

> PASS Ana sees Bram leave


### Items: pick up, give, drop

**Ana's client →** `get sword`

**Ana's client ←**

```text
You pick up a Training Sword.
> 
```

> PASS Ana picks up the Training Sword

**Ana's client →** `north`

**Ana's client ←**

```text
You move north.
Northern Walk
A worn path runs along the northern edge of the newbie zone.
Exits: east, south, west.
Also here: Bram.
Items here:
  Smooth Stone
> 
```

**Bram's client ←**

```text
Ana arrives.
```

> PASS Bram sees Ana arrive

**Ana's client →** `give sword Bram`

**Ana's client ←**

```text
You give a Training Sword to Bram.
> 
```

> PASS Ana gives the sword to Bram

**Bram's client ←**

```text
Ana gives you a Training Sword.
```

> PASS Bram is told he received it

**Bram's client →** `inventory`

**Bram's client ←**

```text
You are carrying:
  Training Sword
> 
```

> PASS Bram's inventory shows the sword

**Ana's client →** `give sword to Bram`

**Ana's client ←**

```text
Give what to whom? Use give <item> <character>, e.g. give sword Bo.
> 
```

> PASS the 'give <item> to <character>' form is refused with usage

**Bram's client →** `get stone`

**Bram's client ←**

```text
You pick up a Smooth Stone.
> 
```

> PASS Bram picks up the Smooth Stone

**Ana's client ←**

```text
Bram picks up a Smooth Stone.
```

> PASS Ana sees Bram pick it up

**Bram's client →** `drop stone`

**Bram's client ←**

```text
You drop a Smooth Stone.
> 
```

> PASS Bram drops the stone again

**Ana's client ←**

```text
Bram drops a Smooth Stone.
```


### Commands outside the milestone say so

**Bram's client →** `wear sword`

**Bram's client ←**

```text
'wear' isn't available yet. Type 'commands' to see what you can do.
> 
```

> PASS wear is reported as not available

**Ana's client →** `kill Bram`

**Ana's client ←**

```text
'kill' isn't available yet. Type 'commands' to see what you can do.
> 
```

> PASS kill is reported as not available

**Ana's client →** `commands`

**Ana's client ←**

```text
Commands you can use:

Movement:
  north, south, east, west (n, s, e, w)
  up, down (u, d)
  northeast, northwest, southeast, southwest (ne, nw, se, sw)

Communication:
  say <message> (') - say something to everyone in the room
  tell <character> <message> (t) - send a private message to someone online
  emote <action> (:) - describe an action to the room

Items:
  inventory (i) - list what you carry
  get <item> (take) - pick something up from the room
  drop <item> - put something down
  give <item> <character> - hand something to someone in the room
  Items are named by a word in their name; 2.sword means the second sword.

Information:
  look (l) - describe the room, who's here and what's lying around
  who - list characters online
  score (sc) - show your character

System:
  save - save your character
  quit (q) - save and disconnect
  help [topic] (h) - this help
  commands (cmd) - list every command

Not available yet: bow, cast, chat, defend, examine, flee, kill, practice, prepare, remove, skills, smile, time, wave, wear, weather, whisper, yell.
> 
```

> PASS commands lists what isn't available


### Disconnect and reconnect

**Ana's client →** `south`

**Ana's client ←**

```text
You move south.
A Simple Room
You are in a basic room with stone walls and a dirt floor.
Exits: east, north, south, west.
Items here:
  Travel Lantern
> 
```

**Bram's client ←**

```text
Ana leaves south.
```

**Ana's client →** `quit`

**Ana's client ←**

```text
Saving character
```

_Ana's client (closes the connection)_

**Bram's client →** `who`

**Bram's client ←**

```text
Players currently online:
  Bram (Elf Mage, Level 1)

1 player online.
> 
```

> PASS Ana is gone from who after quitting

**Ana's client ←**

```text
Welcome to DungeoGo!
Please enter your username:
> 
```

**Ana's client →** `ana`

**Ana's client ←**

```text
Please enter your password:
```

**Ana's client →** `******`

**Ana's client ←**

```text


Welcome back, ana!

--- Character Selection ---
Commands:
  list (l)                 - List your characters
  select (s)               - Select character by number
  create (c)               - Guided character creation
  delete (d)               - Delete character by number
  quit (q)                 - Disconnect

Character> 
```

> PASS Ana's client: logs back in

**Ana's client →** `select`

**Ana's client ←**

```text

--- Character Select ---
Type a character number, or 'cancel' to return.
No. Name           Race      Class     Level  Status    Last Played
--------------------------------------------------------------------
1   Ana            Human     Warrior   1      Alive     2026-10-28 18:49
Select #: 
```

**Ana's client →** `1`

**Ana's client ←**

```text
Welcome, Ana!
You enter the game world...
A Simple Room
You are in a basic room with stone walls and a dirt floor.
Exits: east, north, south, west.
Items here:
  Travel Lantern
> 
```

> PASS Ana's client: enters the world as Ana

> PASS Ana is restored to the starting room


### Server restart: SIGTERM, then start again

_Ana's client (closes the connection)_

_Bram's client (closes the connection)_

> PASS server exits cleanly on SIGTERM (exit code 0)

**Bram's client ←**

```text
Welcome to DungeoGo!
Please enter your username:
> 
```

**Bram's client →** `bram`

**Bram's client ←**

```text
Please enter your password:
```

**Bram's client →** `******`

**Bram's client ←**

```text


Welcome back, bram!

--- Character Selection ---
Commands:
  list (l)                 - List your characters
  select (s)               - Select character by number
  create (c)               - Guided character creation
  delete (d)               - Delete character by number
  quit (q)                 - Disconnect

Character> 
```

> PASS Bram's client: logs back in

**Bram's client →** `select`

**Bram's client ←**

```text

--- Character Select ---
Type a character number, or 'cancel' to return.
No. Name           Race      Class     Level  Status    Last Played
--------------------------------------------------------------------
1   Bram           Elf       Mage      1      Alive     2026-10-28 18:49
Select #: 
```

**Bram's client →** `1`

**Bram's client ←**

```text
Welcome, Bram!
You enter the game world...
Northern Walk
A worn path runs along the northern edge of the newbie zone.
Exits: east, south, west.
Items here:
  Smooth Stone
> 
```

> PASS Bram's client: enters the world as Bram

> PASS after restart Bram is back in the northern room with the stone on the floor

**Bram's client →** `inventory`

**Bram's client ←**

```text
You are carrying:
  Training Sword
> 
```

> PASS after restart Bram still carries the Training Sword

**Ana's client ←**

```text
Welcome to DungeoGo!
Please enter your username:
> 
```

**Ana's client →** `ana`

**Ana's client ←**

```text
Please enter your password:
```

**Ana's client →** `******`

**Ana's client ←**

```text


Welcome back, ana!

--- Character Selection ---
Commands:
  list (l)                 - List your characters
  select (s)               - Select character by number
  create (c)               - Guided character creation
  delete (d)               - Delete character by number
  quit (q)                 - Disconnect

Character> 
```

> PASS Ana's client: logs back in

**Ana's client →** `select`

**Ana's client ←**

```text

--- Character Select ---
Type a character number, or 'cancel' to return.
No. Name           Race      Class     Level  Status    Last Played
--------------------------------------------------------------------
1   Ana            Human     Warrior   1      Alive     2026-10-28 18:49
Select #: 
```

**Ana's client →** `1`

**Ana's client ←**

```text
Welcome, Ana!
You enter the game world...
A Simple Room
You are in a basic room with stone walls and a dirt floor.
Exits: east, north, south, west.
Items here:
  Travel Lantern
> 
```

> PASS Ana's client: enters the world as Ana

> PASS after restart the starting room no longer has the sword (no respawn)

_Ana's client (closes the connection)_

_Bram's client (closes the connection)_

> PASS server exits cleanly again (exit code 0)

