package items

type ItemTemplate struct {
	ID          string
	Name        string
	Type        ItemType
	BaseStats   ItemStats
	Description string
	Rarity      RarityType
	Weight      float64
	Value       int
	Durability  int
	Enchantable bool
	StackSize   int
	Requirements Requirements
}

type ItemType int

const (
	ItemWeapon ItemType = iota
	ItemArmor
	ItemShield
	ItemConsumable
	ItemContainer
	ItemKey
	ItemTreasure
	ItemTool
	ItemMaterial
)

type ItemStats struct {
	Damage       int
	Defense      int
	MagicDefense int
	HitBonus     int
	DodgeBonus   int
	StatBonuses  map[StatType]int
}

type StatType int

const (
	StatStrength StatType = iota
	StatDexterity
	StatIntelligence
	StatConstitution
	StatWisdom
	StatCharisma
)

type RarityType int

const (
	RarityCommon RarityType = iota
	RarityUncommon
	RarityRare
	RarityEpic
	RarityLegendary
)

type Requirements struct {
	MinLevel     int
	MinStats     map[StatType]int
	RequiredRace []string
	RequiredClass []string
	Forbidden    []string
}

func NewItemTemplate(id, name string, itemType ItemType) *ItemTemplate {
	return &ItemTemplate{
		ID:          id,
		Name:        name,
		Type:        itemType,
		BaseStats:   ItemStats{StatBonuses: make(map[StatType]int)},
		Rarity:      RarityCommon,
		Weight:      1.0,
		Value:       10,
		Durability:  100,
		Enchantable: true,
		StackSize:   1,
		Requirements: Requirements{
			MinStats: make(map[StatType]int),
		},
	}
}

func (it *ItemTemplate) IsStackable() bool {
	return it.StackSize > 1
}

func (it *ItemTemplate) CanUse(character interface{}) bool {
	// This would need to check character stats, level, race, class
	// For now, return true - implement actual logic later
	return true
}

func GetItemTypeName(itemType ItemType) string {
	names := map[ItemType]string{
		ItemWeapon:     "Weapon",
		ItemArmor:      "Armor",
		ItemShield:     "Shield",
		ItemConsumable: "Consumable",
		ItemContainer:  "Container",
		ItemKey:        "Key",
		ItemTreasure:   "Treasure",
		ItemTool:       "Tool",
		ItemMaterial:   "Material",
	}
	
	if name, exists := names[itemType]; exists {
		return name
	}
	return "Unknown"
}

func GetRarityName(rarity RarityType) string {
	names := map[RarityType]string{
		RarityCommon:    "Common",
		RarityUncommon:  "Uncommon",
		RarityRare:      "Rare",
		RarityEpic:      "Epic",
		RarityLegendary: "Legendary",
	}
	
	if name, exists := names[rarity]; exists {
		return name
	}
	return "Unknown"
}