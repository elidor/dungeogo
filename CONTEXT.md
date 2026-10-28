# DungeoGo

A text-based multi-user dungeon: people connect over TCP, log in, and play a Character in a shared persistent world.

## Language

### People and sessions

**Player**:
The account a person logs in with. A Player owns zero or more Characters.
_Avoid_: account, user

**Character**:
An avatar a Player plays in the world. A Player plays at most one Character at a time.
_Avoid_: avatar, toon, player (for the in-world figure)

**Session**:
One authenticated connection of a Player. A Player has at most one live Session; logging in again replaces it.
_Avoid_: connection, client (for the gameplay concept)

### World

**Zone**:
A named area of the world made up of Rooms.
_Avoid_: area, region

**Room**:
A single location in a Zone, with exits leading to other Rooms.
_Avoid_: location, tile, cell

**Presence**:
The live record of which Characters are online and which Room each is in.
_Avoid_: who-list, online list

### Items

**Item**:
A single physical object in the world, made from an item template. Every Item is held by exactly one Owner.
_Avoid_: object, thing, item instance (outside persistence code)

**Owner**:
The one Room or Character that currently holds an Item. Deleting a Character destroys the Items it holds (see ADR-0001).
_Avoid_: holder, location (for items), container

**Display name**:
The name players see for an Item: its custom name if it has one, otherwise its template's name, e.g. "Training Sword".
_Avoid_: label, title

**Short description**:
How an Item is named inside a sentence: its display name with an article, e.g. "a Training Sword", "an Iron Key".
_Avoid_: short desc, sdesc

**Item target**:
What a player types to pick out an Item: a keyword matching the start of any word in its display name, optionally numbered as `2.sword` for the second match. A bare keyword means the first match. Matches are numbered in arrival order.
_Avoid_: selector, item spec, item argument

**Arrival order**:
The order Items came to their current Owner, oldest first. `look`, `inventory` and Item target numbers all follow it.
_Avoid_: item order, sort order
