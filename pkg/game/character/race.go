package character

type Race struct {
	ID            string
	Name          string
	StatModifiers StatModifiers
	SkillBonuses  map[SkillType]int
	SizeCategory  SizeType
	Lifespan      int
	Description   string
	Abilities     []RacialAbility
}

type StatModifiers struct {
	Strength     int
	Dexterity    int
	Intelligence int
	Constitution int
	Wisdom       int
	Charisma     int
}

type SizeType int

const (
	SizeTiny SizeType = iota
	SizeSmall
	SizeMedium
	SizeLarge
	SizeHuge
)

type RacialAbility struct {
	ID          string
	Name        string
	Description string
	Type        AbilityType
	Passive     bool
}

type AbilityType int

const (
	AbilityVision AbilityType = iota
	AbilityResistance
	AbilityMovement
	AbilityCombat
	AbilityMagic
)

func GetRaceByID(id string) (*Race, error) {
	races := getStandardRaces()
	if race, exists := races[id]; exists {
		return race, nil
	}
	return nil, ErrRaceNotFound
}

func GetAllRaces() map[string]*Race {
	return getStandardRaces()
}

func getStandardRaces() map[string]*Race {
	return map[string]*Race{
		"human": {
			ID:           "human",
			Name:         "Human",
			SizeCategory: SizeMedium,
			Lifespan:     80,
			Description:  "Versatile and adaptable, humans are the most common race.",
			StatModifiers: StatModifiers{
				Strength:     0,
				Dexterity:    0,
				Intelligence: 0,
				Constitution: 0,
				Wisdom:       0,
				Charisma:     1,
			},
			SkillBonuses: map[SkillType]int{},
			Abilities:    []RacialAbility{},
		},
		"elf": {
			ID:           "elf",
			Name:         "Elf",
			SizeCategory: SizeMedium,
			Lifespan:     500,
			Description:  "Graceful and magical, elves live in harmony with nature.",
			StatModifiers: StatModifiers{
				Strength:     -1,
				Dexterity:    2,
				Intelligence: 1,
				Constitution: -1,
				Wisdom:       1,
				Charisma:     0,
			},
			SkillBonuses: map[SkillType]int{
				SkillArchery: 10,
				SkillMagic:   5,
			},
			Abilities: []RacialAbility{
				{
					ID:          "darkvision",
					Name:        "Darkvision",
					Description: "Can see in darkness up to 60 feet",
					Type:        AbilityVision,
					Passive:     true,
				},
			},
		},
		"dwarf": {
			ID:           "dwarf",
			Name:         "Dwarf",
			SizeCategory: SizeMedium,
			Lifespan:     200,
			Description:  "Hardy and strong, dwarves are master craftsmen and warriors.",
			StatModifiers: StatModifiers{
				Strength:     1,
				Dexterity:    -1,
				Intelligence: 0,
				Constitution: 2,
				Wisdom:       1,
				Charisma:     -1,
			},
			SkillBonuses: map[SkillType]int{
				SkillAxes:     15,
				SkillCrafting: 20,
			},
			Abilities: []RacialAbility{
				{
					ID:          "poison_resistance",
					Name:        "Poison Resistance",
					Description: "Resistance to poison effects",
					Type:        AbilityResistance,
					Passive:     true,
				},
			},
		},
	}
}