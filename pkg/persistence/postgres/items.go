package postgres

import (
	"database/sql"
	"encoding/json"
	"fmt"
	
	"github.com/elidor/dungeogo/pkg/game/items"
)

type ItemRepository struct {
	db *sql.DB
}

func NewItemRepository(db *sql.DB) *ItemRepository {
	return &ItemRepository{db: db}
}

func (r *ItemRepository) CreateItemInstance(item *items.ItemInstance) error {
	enchantmentsJSON, err := json.Marshal(item.Enchantments)
	if err != nil {
		return fmt.Errorf("failed to marshal enchantments: %w", err)
	}
	
	modificationsJSON, err := json.Marshal(item.Modifications)
	if err != nil {
		return fmt.Errorf("failed to marshal modifications: %w", err)
	}
	
	query := `
		INSERT INTO item_instances (id, template_id, owner_id, quantity, durability,
			enchantments, custom_name, modifications, created_at, last_used)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	
	_, err = r.db.Exec(query, item.ID, item.TemplateID, item.OwnerID,
		item.Quantity, item.Durability, enchantmentsJSON, item.CustomName,
		modificationsJSON, item.CreatedAt, item.LastUsed)
	
	if err != nil {
		return fmt.Errorf("failed to create item instance: %w", err)
	}
	
	return nil
}

func (r *ItemRepository) GetItemInstance(itemID string) (*items.ItemInstance, error) {
	query := `
		SELECT id, template_id, owner_id, quantity, durability, enchantments,
			custom_name, modifications, created_at, last_used
		FROM item_instances WHERE id = $1`
	
	item := &items.ItemInstance{}
	var enchantmentsJSON, modificationsJSON []byte
	
	err := r.db.QueryRow(query, itemID).Scan(
		&item.ID, &item.TemplateID, &item.OwnerID, &item.Quantity,
		&item.Durability, &enchantmentsJSON, &item.CustomName,
		&modificationsJSON, &item.CreatedAt, &item.LastUsed)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("item instance not found: %s", itemID)
		}
		return nil, fmt.Errorf("failed to get item instance: %w", err)
	}
	
	if err := json.Unmarshal(enchantmentsJSON, &item.Enchantments); err != nil {
		return nil, fmt.Errorf("failed to unmarshal enchantments: %w", err)
	}
	
	if err := json.Unmarshal(modificationsJSON, &item.Modifications); err != nil {
		return nil, fmt.Errorf("failed to unmarshal modifications: %w", err)
	}
	
	return item, nil
}

func (r *ItemRepository) UpdateItemInstance(item *items.ItemInstance) error {
	enchantmentsJSON, err := json.Marshal(item.Enchantments)
	if err != nil {
		return fmt.Errorf("failed to marshal enchantments: %w", err)
	}
	
	modificationsJSON, err := json.Marshal(item.Modifications)
	if err != nil {
		return fmt.Errorf("failed to marshal modifications: %w", err)
	}
	
	query := `
		UPDATE item_instances SET template_id = $2, owner_id = $3, quantity = $4,
			durability = $5, enchantments = $6, custom_name = $7, modifications = $8,
			last_used = $9
		WHERE id = $1`
	
	_, err = r.db.Exec(query, item.ID, item.TemplateID, item.OwnerID,
		item.Quantity, item.Durability, enchantmentsJSON, item.CustomName,
		modificationsJSON, item.LastUsed)
	
	if err != nil {
		return fmt.Errorf("failed to update item instance: %w", err)
	}
	
	return nil
}

func (r *ItemRepository) DeleteItemInstance(itemID string) error {
	query := `DELETE FROM item_instances WHERE id = $1`
	_, err := r.db.Exec(query, itemID)
	if err != nil {
		return fmt.Errorf("failed to delete item instance: %w", err)
	}
	return nil
}

func (r *ItemRepository) GetPlayerItems(characterID string) ([]*items.ItemInstance, error) {
	query := `
		SELECT id, template_id, owner_id, quantity, durability, enchantments,
			custom_name, modifications, created_at, last_used
		FROM item_instances WHERE owner_id = $1`
	
	rows, err := r.db.Query(query, characterID)
	if err != nil {
		return nil, fmt.Errorf("failed to get player items: %w", err)
	}
	defer rows.Close()
	
	var itemInstances []*items.ItemInstance
	for rows.Next() {
		item := &items.ItemInstance{}
		var enchantmentsJSON, modificationsJSON []byte
		
		err := rows.Scan(&item.ID, &item.TemplateID, &item.OwnerID,
			&item.Quantity, &item.Durability, &enchantmentsJSON,
			&item.CustomName, &modificationsJSON, &item.CreatedAt, &item.LastUsed)
		if err != nil {
			return nil, fmt.Errorf("failed to scan item instance: %w", err)
		}
		
		if err := json.Unmarshal(enchantmentsJSON, &item.Enchantments); err != nil {
			return nil, fmt.Errorf("failed to unmarshal enchantments: %w", err)
		}
		
		if err := json.Unmarshal(modificationsJSON, &item.Modifications); err != nil {
			return nil, fmt.Errorf("failed to unmarshal modifications: %w", err)
		}
		
		itemInstances = append(itemInstances, item)
	}
	
	return itemInstances, nil
}

func (r *ItemRepository) GetRoomItems(roomID string) ([]*items.ItemInstance, error) {
	return r.GetPlayerItems(roomID) // Same logic, different owner
}

func (r *ItemRepository) TransferItem(itemID, newOwnerID string) error {
	query := `UPDATE item_instances SET owner_id = $1 WHERE id = $2`
	_, err := r.db.Exec(query, newOwnerID, itemID)
	if err != nil {
		return fmt.Errorf("failed to transfer item: %w", err)
	}
	return nil
}