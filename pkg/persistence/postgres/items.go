package postgres

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/elidor/dungeogo/pkg/game/items"
	"github.com/elidor/dungeogo/pkg/persistence/interfaces"
)

type ItemRepository struct{ db *sql.DB }

func NewItemRepository(db *sql.DB) *ItemRepository { return &ItemRepository{db: db} }

// validateOwner rejects an Owner whose kind is missing or unknown, or whose ID
// is empty. Callers always say which kind of owner they mean; nothing infers
// the kind from the ID. The generated foreign keys check the owner exists.
func validateOwner(owner items.Owner) error {
	if !owner.Kind.Valid() {
		return fmt.Errorf("invalid item %s: kind must be %q or %q", owner, items.OwnerRoom, items.OwnerCharacter)
	}
	if owner.ID == "" {
		return fmt.Errorf("invalid item owner: %s has no ID", owner.Kind)
	}
	return nil
}

// storedItem is an Item's fields in the form both create and update write.
type storedItem struct {
	enchantments  []byte
	modifications []byte
	lastUsed      sql.NullTime
}

// prepareItem runs the save steps create and update share: serializing the
// fields stored as JSON or nullable columns.
func prepareItem(item *items.ItemInstance) (storedItem, error) {
	enchantments, err := json.Marshal(item.Enchantments)
	if err != nil {
		return storedItem{}, fmt.Errorf("failed to marshal enchantments for item %q: %w", item.ID, err)
	}
	modifications, err := json.Marshal(item.Modifications)
	if err != nil {
		return storedItem{}, fmt.Errorf("failed to marshal modifications for item %q: %w", item.ID, err)
	}
	return storedItem{
		enchantments:  enchantments,
		modifications: modifications,
		lastUsed:      sql.NullTime{Time: item.LastUsed, Valid: !item.LastUsed.IsZero()},
	}, nil
}

func (r *ItemRepository) CreateItemInstance(item *items.ItemInstance) error {
	if err := validateOwner(item.Owner); err != nil {
		return err
	}
	stored, err := prepareItem(item)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(`INSERT INTO item_instances (id,template_id,owner_kind,owner_id,quantity,durability,enchantments,custom_name,modifications,created_at,last_used) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, item.ID, item.TemplateID, item.Owner.Kind, item.Owner.ID, item.Quantity, item.Durability, stored.enchantments, item.CustomName, stored.modifications, item.CreatedAt, stored.lastUsed)
	if err != nil {
		return fmt.Errorf("failed to create item %q: %w", item.ID, err)
	}
	return nil
}

const itemColumns = `id,template_id,owner_kind,owner_id,quantity,durability,enchantments,COALESCE(custom_name,''),modifications,created_at,last_used`

type itemScanner interface{ Scan(...interface{}) error }

func scanItem(row itemScanner) (*items.ItemInstance, error) {
	item := &items.ItemInstance{}
	var enchantments, modifications []byte
	var lastUsed sql.NullTime
	if err := row.Scan(&item.ID, &item.TemplateID, &item.Owner.Kind, &item.Owner.ID, &item.Quantity, &item.Durability, &enchantments, &item.CustomName, &modifications, &item.CreatedAt, &lastUsed); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(enchantments, &item.Enchantments); err != nil {
		return nil, fmt.Errorf("failed to unmarshal enchantments for item %q: %w", item.ID, err)
	}
	if err := json.Unmarshal(modifications, &item.Modifications); err != nil {
		return nil, fmt.Errorf("failed to unmarshal modifications for item %q: %w", item.ID, err)
	}
	item.LastUsed = lastUsed.Time
	return item, nil
}

func (r *ItemRepository) GetItemInstance(id string) (*items.ItemInstance, error) {
	item, err := scanItem(r.db.QueryRow(`SELECT `+itemColumns+` FROM item_instances WHERE id=$1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: %s", interfaces.ErrItemNotFound, id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get item %q: %w", id, err)
	}
	return item, nil
}

// UpdateItemInstance saves an Item's own fields. It deliberately leaves
// ownership alone, so a stale copy of an Item can't undo a TransferItem.
func (r *ItemRepository) UpdateItemInstance(item *items.ItemInstance) error {
	stored, err := prepareItem(item)
	if err != nil {
		return err
	}
	result, err := r.db.Exec(`UPDATE item_instances SET template_id=$2,quantity=$3,durability=$4,enchantments=$5,custom_name=$6,modifications=$7,last_used=$8 WHERE id=$1`, item.ID, item.TemplateID, item.Quantity, item.Durability, stored.enchantments, item.CustomName, stored.modifications, stored.lastUsed)
	return requireOneItemRow("update", item.ID, result, err)
}

func (r *ItemRepository) DeleteItemInstance(id string) error {
	result, err := r.db.Exec(`DELETE FROM item_instances WHERE id=$1`, id)
	return requireOneItemRow("delete", id, result, err)
}

func (r *ItemRepository) GetPlayerItems(characterID string) ([]*items.ItemInstance, error) {
	return r.getOwnedItems(items.CharacterOwner(characterID))
}
func (r *ItemRepository) GetRoomItems(roomID string) ([]*items.ItemInstance, error) {
	return r.getOwnedItems(items.RoomOwner(roomID))
}
func (r *ItemRepository) getOwnedItems(owner items.Owner) ([]*items.ItemInstance, error) {
	rows, err := r.db.Query(`SELECT `+itemColumns+` FROM item_instances WHERE owner_kind=$1 AND owner_id=$2 ORDER BY id`, owner.Kind, owner.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list items held by %s: %w", owner, err)
	}
	defer rows.Close()
	var result []*items.ItemInstance
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to read item held by %s: %w", owner, err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to list items held by %s: %w", owner, err)
	}
	return result, nil
}

// TransferItem moves an Item from the Owner the caller expects to a new Owner.
// The expected Owner is part of the WHERE clause, so the check and the move are
// one atomic write: of two competing transfers of the same Item, the second
// re-checks the row after the first commits, matches nothing, and reports false.
func (r *ItemRepository) TransferItem(itemID string, from, to items.Owner) (bool, error) {
	if err := validateOwner(from); err != nil {
		return false, err
	}
	if err := validateOwner(to); err != nil {
		return false, err
	}
	result, err := r.db.Exec(`UPDATE item_instances SET owner_kind=$4,owner_id=$5 WHERE id=$1 AND owner_kind=$2 AND owner_id=$3`, itemID, from.Kind, from.ID, to.Kind, to.ID)
	if err != nil {
		return false, fmt.Errorf("failed to transfer item %q from %s to %s: %w", itemID, from, to, err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to transfer item %q: %w", itemID, err)
	}
	return count == 1, nil
}

// requireOneItemRow turns the result of a write that targets one Item by ID
// into an error: the write's own failure, or ErrItemNotFound if no row matched.
func requireOneItemRow(action, id string, result sql.Result, err error) error {
	if err != nil {
		return fmt.Errorf("failed to %s item %q: %w", action, id, err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to %s item %q: %w", action, id, err)
	}
	if count != 1 {
		return fmt.Errorf("failed to %s item: %w: %s", action, interfaces.ErrItemNotFound, id)
	}
	return nil
}
