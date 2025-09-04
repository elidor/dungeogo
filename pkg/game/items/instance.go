package items

import (
	"time"
)

type ItemInstance struct {
	ID           string
	TemplateID   string
	OwnerID      string
	Quantity     int
	Durability   int
	Enchantments []Enchantment
	CustomName   string
	Modifications map[string]interface{}
	CreatedAt    time.Time
	LastUsed     time.Time
}

type Enchantment struct {
	ID          string
	Name        string
	Description string
	Type        EnchantmentType
	Power       int
	Duration    time.Duration
	AppliedAt   time.Time
}

type EnchantmentType int

const (
	EnchantmentDamage EnchantmentType = iota
	EnchantmentDefense
	EnchantmentStat
	EnchantmentResistance
	EnchantmentSpecial
)

func NewItemInstance(templateID, ownerID string, quantity int) *ItemInstance {
	return &ItemInstance{
		TemplateID:    templateID,
		OwnerID:       ownerID,
		Quantity:      quantity,
		Durability:    100, // Will be set from template
		Enchantments:  []Enchantment{},
		Modifications: make(map[string]interface{}),
		CreatedAt:     time.Now(),
	}
}

func (ii *ItemInstance) GetDisplayName() string {
	if ii.CustomName != "" {
		return ii.CustomName
	}
	// Would normally look up template name
	return "Unknown Item"
}

func (ii *ItemInstance) IsBroken() bool {
	return ii.Durability <= 0
}

func (ii *ItemInstance) TakeDamage(damage int) {
	ii.Durability -= damage
	if ii.Durability < 0 {
		ii.Durability = 0
	}
}

func (ii *ItemInstance) Repair(amount int) {
	ii.Durability += amount
	// Would need to check max durability from template
}

func (ii *ItemInstance) AddEnchantment(enchantment Enchantment) {
	enchantment.AppliedAt = time.Now()
	ii.Enchantments = append(ii.Enchantments, enchantment)
}

func (ii *ItemInstance) RemoveEnchantment(enchantmentID string) bool {
	for i, enchantment := range ii.Enchantments {
		if enchantment.ID == enchantmentID {
			ii.Enchantments = append(ii.Enchantments[:i], ii.Enchantments[i+1:]...)
			return true
		}
	}
	return false
}

func (ii *ItemInstance) HasEnchantment(enchantmentType EnchantmentType) bool {
	for _, enchantment := range ii.Enchantments {
		if enchantment.Type == enchantmentType {
			return true
		}
	}
	return false
}

func (ii *ItemInstance) GetEnchantmentBonus(enchantmentType EnchantmentType) int {
	bonus := 0
	for _, enchantment := range ii.Enchantments {
		if enchantment.Type == enchantmentType {
			bonus += enchantment.Power
		}
	}
	return bonus
}

func (ii *ItemInstance) UpdateLastUsed() {
	ii.LastUsed = time.Now()
}

func (ii *ItemInstance) CanStack(other *ItemInstance) bool {
	return ii.TemplateID == other.TemplateID &&
		   len(ii.Enchantments) == 0 &&
		   len(other.Enchantments) == 0 &&
		   ii.Durability == other.Durability &&
		   ii.CustomName == other.CustomName
}