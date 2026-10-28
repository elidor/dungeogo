# Deleting a Character destroys the Items it holds

Every Item is held by exactly one Room or Character, so deleting a Character that holds Items must decide where those Items go. We destroy them along with the Character, in the same transaction, rather than dropping them into the Character's last Room or refusing the deletion. A deleted Character has no meaningful Room to drop into: it isn't in the world, and its saved location may be stale.

## Considered Options

- **Drop held Items into the Character's last Room.** Rejected: puts Items into a Room on behalf of a Character who isn't there, at a location that may no longer reflect play.
- **Refuse deletion until the Character holds nothing.** Rejected: forces a Player back into the world just to empty an inventory before deleting.

## Consequences

Deletion is irreversible for the Items too; nothing is recoverable afterwards. Anyone tempted to "fix" this into a drop should reopen this decision rather than change the behaviour quietly.
