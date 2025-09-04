package items

import (
	"errors"
	"sync"
)

var (
	ErrTemplateNotFound = errors.New("item template not found")
	ErrInvalidTemplate  = errors.New("invalid item template")
)

type ItemRegistry struct {
	templates map[string]*ItemTemplate
	mutex     sync.RWMutex
}

func NewItemRegistry() *ItemRegistry {
	registry := &ItemRegistry{
		templates: make(map[string]*ItemTemplate),
	}
	
	registry.loadDefaultTemplates()
	return registry
}

func (ir *ItemRegistry) RegisterTemplate(template *ItemTemplate) error {
	if template == nil || template.ID == "" {
		return ErrInvalidTemplate
	}
	
	ir.mutex.Lock()
	defer ir.mutex.Unlock()
	
	ir.templates[template.ID] = template
	return nil
}

func (ir *ItemRegistry) GetTemplate(templateID string) (*ItemTemplate, error) {
	ir.mutex.RLock()
	defer ir.mutex.RUnlock()
	
	template, exists := ir.templates[templateID]
	if !exists {
		return nil, ErrTemplateNotFound
	}
	
	return template, nil
}

func (ir *ItemRegistry) GetAllTemplates() map[string]*ItemTemplate {
	ir.mutex.RLock()
	defer ir.mutex.RUnlock()
	
	result := make(map[string]*ItemTemplate)
	for id, template := range ir.templates {
		result[id] = template
	}
	
	return result
}

func (ir *ItemRegistry) GetTemplatesByType(itemType ItemType) []*ItemTemplate {
	ir.mutex.RLock()
	defer ir.mutex.RUnlock()
	
	var result []*ItemTemplate
	for _, template := range ir.templates {
		if template.Type == itemType {
			result = append(result, template)
		}
	}
	
	return result
}

func (ir *ItemRegistry) loadDefaultTemplates() {
	templates := []*ItemTemplate{
		{
			ID:          "rusty_sword",
			Name:        "Rusty Sword",
			Type:        ItemWeapon,
			Description: "A worn and rusty sword, but still functional.",
			BaseStats:   ItemStats{Damage: 5, StatBonuses: make(map[StatType]int)},
			Rarity:      RarityCommon,
			Weight:      3.0,
			Value:       10,
			Durability:  50,
			Enchantable: true,
			StackSize:   1,
			Requirements: Requirements{
				MinLevel: 1,
				MinStats: map[StatType]int{StatStrength: 8},
			},
		},
		{
			ID:          "leather_armor",
			Name:        "Leather Armor",
			Type:        ItemArmor,
			Description: "Basic leather armor providing minimal protection.",
			BaseStats:   ItemStats{Defense: 3, StatBonuses: make(map[StatType]int)},
			Rarity:      RarityCommon,
			Weight:      8.0,
			Value:       25,
			Durability:  75,
			Enchantable: true,
			StackSize:   1,
			Requirements: Requirements{
				MinLevel: 1,
				MinStats: make(map[StatType]int),
			},
		},
		{
			ID:          "health_potion",
			Name:        "Health Potion",
			Type:        ItemConsumable,
			Description: "A small vial of red liquid that restores health.",
			BaseStats:   ItemStats{StatBonuses: make(map[StatType]int)},
			Rarity:      RarityCommon,
			Weight:      0.5,
			Value:       50,
			Durability:  1,
			Enchantable: false,
			StackSize:   10,
			Requirements: Requirements{
				MinLevel: 1,
				MinStats: make(map[StatType]int),
			},
		},
		{
			ID:          "magic_staff",
			Name:        "Magic Staff",
			Type:        ItemWeapon,
			Description: "A wooden staff imbued with magical energy.",
			BaseStats: ItemStats{
				Damage:      2,
				MagicDefense: 5,
				StatBonuses: map[StatType]int{
					StatIntelligence: 2,
				},
			},
			Rarity:      RarityUncommon,
			Weight:      2.0,
			Value:       150,
			Durability:  80,
			Enchantable: true,
			StackSize:   1,
			Requirements: Requirements{
				MinLevel: 3,
				MinStats: map[StatType]int{StatIntelligence: 12},
				RequiredClass: []string{"mage"},
			},
		},
	}
	
	for _, template := range templates {
		ir.RegisterTemplate(template)
	}
}