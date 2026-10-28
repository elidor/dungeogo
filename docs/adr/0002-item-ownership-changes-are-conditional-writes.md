# Item ownership changes only through a conditional write

An Item changes Owner only through `TransferItem(item, from, to)`, a single `UPDATE … WHERE owner = from` whose row count says whether the move happened. General item updates never write ownership, and callers always name the Owner's kind; nothing infers it from the ID. We rely on the database's row-level check rather than on the game server's in-process action lock, so a pickup, drop or give can't duplicate or lose an Item, even if a stale copy is saved later or an action runs outside that lock.

## Considered Options

- **Serialize item actions with the in-process action lock alone.** Rejected: correct only while every writer runs inside that lock in one process, and a later write of a stale in-memory Item could still move it back.
- **Version column with optimistic locking on every item write.** Rejected for now: heavier than needed when ownership is the only field actions compete for.

## Consequences

Callers must handle `moved == false` as a normal outcome ("someone else got it first"), not an error. Anything that needs to change who holds an Item goes through `TransferItem`; adding ownership back to the general update would reopen the duplication bug this prevents.
