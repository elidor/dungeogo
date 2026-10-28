package postgres

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/elidor/dungeogo/pkg/game/items"
)

type ItemRepository struct{ db *sql.DB }

func NewItemRepository(db *sql.DB) *ItemRepository { return &ItemRepository{db: db} }

// resolveOwner is the single place that decides whether an owner ID names a Room
// or a Character. An Owner with a known Kind is checked and returned unchanged;
// an Owner with no Kind is classified by looking the ID up. Generated foreign
// keys enforce that the owner exists.
func (r *ItemRepository) resolveOwner(owner items.Owner) (items.Owner, error) {
	if owner.Kind != "" {
		if !owner.Kind.Valid() {
			return items.Owner{}, fmt.Errorf("invalid item owner kind %q", owner.Kind)
		}
		return owner, nil
	}
	var room, character bool
	err := r.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM room_definitions WHERE room_id=$1), EXISTS(SELECT 1 FROM characters WHERE id::text=$1)`, owner.ID).Scan(&room, &character)
	if err != nil {
		return items.Owner{}, fmt.Errorf("resolve item owner: %w", err)
	}
	if room == character {
		return items.Owner{}, fmt.Errorf("item %s must identify exactly one existing room or character; specify owner kind for ambiguous IDs", owner)
	}
	if room {
		return items.RoomOwner(owner.ID), nil
	}
	return items.CharacterOwner(owner.ID), nil
}

func (r *ItemRepository) CreateItemInstance(item *items.ItemInstance) error {
	owner, err := r.resolveOwner(item.Owner)
	if err != nil {
		return err
	}
	enchantments, err := json.Marshal(item.Enchantments)
	if err != nil {
		return err
	}
	modifications, err := json.Marshal(item.Modifications)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(`INSERT INTO item_instances (id,template_id,owner_kind,owner_id,quantity,durability,enchantments,custom_name,modifications,created_at,last_used) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, item.ID, item.TemplateID, owner.Kind, owner.ID, item.Quantity, item.Durability, enchantments, item.CustomName, modifications, item.CreatedAt, sql.NullTime{Time: item.LastUsed, Valid: !item.LastUsed.IsZero()})
	if err != nil {
		return fmt.Errorf("create item instance: %w", err)
	}
	item.Owner = owner
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
		return nil, err
	}
	if err := json.Unmarshal(modifications, &item.Modifications); err != nil {
		return nil, err
	}
	item.LastUsed = lastUsed.Time
	return item, nil
}

func (r *ItemRepository) GetItemInstance(id string) (*items.ItemInstance, error) {
	item, err := scanItem(r.db.QueryRow(`SELECT `+itemColumns+` FROM item_instances WHERE id=$1`, id))
	if err != nil {
		return nil, fmt.Errorf("get item instance %q: %w", id, err)
	}
	return item, nil
}

func (r *ItemRepository) UpdateItemInstance(item *items.ItemInstance) error {
	owner, err := r.resolveOwner(item.Owner)
	if err != nil {
		return err
	}
	enchantments, err := json.Marshal(item.Enchantments)
	if err != nil {
		return err
	}
	modifications, err := json.Marshal(item.Modifications)
	if err != nil {
		return err
	}
	result, err := r.db.Exec(`UPDATE item_instances SET template_id=$2,owner_kind=$3,owner_id=$4,quantity=$5,durability=$6,enchantments=$7,custom_name=$8,modifications=$9,last_used=$10 WHERE id=$1`, item.ID, item.TemplateID, owner.Kind, owner.ID, item.Quantity, item.Durability, enchantments, item.CustomName, modifications, sql.NullTime{Time: item.LastUsed, Valid: !item.LastUsed.IsZero()})
	if err := requireItemUpdate(result, err); err != nil {
		return err
	}
	item.Owner = owner
	return nil
}

func (r *ItemRepository) DeleteItemInstance(id string) error {
	result, err := r.db.Exec(`DELETE FROM item_instances WHERE id=$1`, id)
	return requireItemUpdate(result, err)
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
		return nil, fmt.Errorf("list item instances: %w", err)
	}
	defer rows.Close()
	var result []*items.ItemInstance
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, fmt.Errorf("read item instance: %w", err)
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

// TransferItem preserves the legacy repository API for valid unambiguous owners.
// Gameplay's conditional pickup/drop operations are introduced separately in #10.
func (r *ItemRepository) TransferItem(id, newOwnerID string) error {
	owner, err := r.resolveOwner(items.Owner{ID: newOwnerID})
	if err != nil {
		return err
	}
	result, err := r.db.Exec(`UPDATE item_instances SET owner_kind=$1,owner_id=$2 WHERE id=$3`, owner.Kind, owner.ID, id)
	return requireItemUpdate(result, err)
}

func requireItemUpdate(result sql.Result, err error) error {
	if err != nil {
		return fmt.Errorf("write item instance: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return fmt.Errorf("item instance not found")
	}
	return nil
}
