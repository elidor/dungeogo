package character

type Class struct {
	ID                  string
	Name                string
	Description         string
	PrimaryStats        []StatType
	HitDie              int
	BaseAttackBonus     int
	Abilities           []ClassAbility
	WeaponProficiencies []WeaponType
	ArmorProficiencies  []ArmorType
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

type ClassAbility struct {
	ID           string
	Name         string
	Description  string
	Level        int
	Type         AbilityType
	Cooldown     int
	ManaCost     int
	Requirements []string
}

type WeaponType int

const (
	WeaponSwords WeaponType = iota
	WeaponAxes
	WeaponMaces
	WeaponDaggers
	WeaponBows
	WeaponCrossbows
	WeaponStaves
)

type ArmorType int

const (
	ArmorCloth ArmorType = iota
	ArmorLeather
	ArmorChain
	ArmorPlate
	ArmorShields
)

func GetClassByID(id string) (*Class, error) {
	classes := getStandardClasses()
	if class, exists := classes[id]; exists {
		return class, nil
	}
	return nil, ErrClassNotFound
}

func GetAllClasses() map[string]*Class {
	return getStandardClasses()
}

func getStandardClasses() map[string]*Class {
	return map[string]*Class{
		"warrior": {
			ID:          "warrior",
			Name:        "Warrior",
			Description: "Masters of combat and weapons, warriors excel in physical battle.",
			PrimaryStats: []StatType{
				StatStrength,
				StatConstitution,
			},
			HitDie:          10,
			BaseAttackBonus: 1,
			WeaponProficiencies: []WeaponType{
				WeaponSwords,
				WeaponAxes,
				WeaponMaces,
			},
			ArmorProficiencies: []ArmorType{
				ArmorLeather,
				ArmorChain,
				ArmorPlate,
				ArmorShields,
			},
			Abilities: []ClassAbility{
				{
					ID:          "power_attack",
					Name:        "Power Attack",
					Description: "Deal extra damage at the cost of accuracy",
					Level:       1,
					Type:        AbilityCombat,
					Cooldown:    0,
					ManaCost:    0,
				},
			},
		},
		"mage": {
			ID:          "mage",
			Name:        "Mage",
			Description: "Masters of arcane magic, wielding powerful spells.",
			PrimaryStats: []StatType{
				StatIntelligence,
				StatWisdom,
			},
			HitDie:          6,
			BaseAttackBonus: 0,
			WeaponProficiencies: []WeaponType{
				WeaponDaggers,
				WeaponStaves,
			},
			ArmorProficiencies: []ArmorType{
				ArmorCloth,
			},
			Abilities: []ClassAbility{
				{
					ID:          "magic_missile",
					Name:        "Magic Missile",
					Description: "Launches a magical projectile that always hits",
					Level:       1,
					Type:        AbilityMagic,
					Cooldown:    3,
					ManaCost:    5,
				},
			},
		},
		"rogue": {
			ID:          "rogue",
			Name:        "Rogue",
			Description: "Masters of stealth and precision, rogues strike from the shadows.",
			PrimaryStats: []StatType{
				StatDexterity,
				StatIntelligence,
			},
			HitDie:          8,
			BaseAttackBonus: 1,
			WeaponProficiencies: []WeaponType{
				WeaponDaggers,
				WeaponBows,
				WeaponCrossbows,
			},
			ArmorProficiencies: []ArmorType{
				ArmorCloth,
				ArmorLeather,
			},
			Abilities: []ClassAbility{
				{
					ID:          "sneak_attack",
					Name:        "Sneak Attack",
					Description: "Deal extra damage when attacking from stealth",
					Level:       1,
					Type:        AbilityCombat,
					Cooldown:    0,
					ManaCost:    0,
				},
			},
		},
	}
}