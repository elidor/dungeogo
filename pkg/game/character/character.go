package character

import (
	"time"
)

type Character struct {
	ID          string
	PlayerID    string
	Name        string
	Race        *Race
	Class       *Class
	Skills      *SkillSet
	Stats       *CharacterStats
	Location    *Location
	State       CharacterState
	CreatedAt   time.Time
	LastPlayed  time.Time
	PlayTime    time.Duration
	Level       int
	Experience  int
	DeathCount  int
	KillCount   int
	Description string
	Appearance  CharacterAppearance
}

type CharacterState int

const (
	CharacterAlive CharacterState = iota
	CharacterDead
	CharacterSleeping
	CharacterAfk
	CharacterInCombat
)

type Location struct {
	RoomID string
	ZoneID string
	X      int
	Y      int
}

type CharacterStats struct {
	Strength     int
	Dexterity    int
	Intelligence int
	Constitution int
	Wisdom       int
	Charisma     int
	Health       int
	MaxHealth    int
	Mana         int
	MaxMana      int
	Stamina      int
	MaxStamina   int
}

type CharacterAppearance struct {
	Height      string
	Weight      string
	EyeColor    string
	HairColor   string
	SkinColor   string
	Build       string
	Description string
}

func NewCharacter(playerID, name string, race *Race, class *Class) *Character {
	stats := calculateStartingStats(race, class)
	
	return &Character{
		PlayerID:    playerID,
		Name:        name,
		Race:        race,
		Class:       class,
		Stats:       stats,
		Skills:      NewSkillSet(),
		State:       CharacterAlive,
		CreatedAt:   time.Now(),
		Level:       1,
		Experience:  0,
		DeathCount:  0,
		KillCount:   0,
		Location: &Location{
			RoomID: "starting_room",
			ZoneID: "newbie_zone",
		},
	}
}

func (c *Character) IsAlive() bool {
	return c.State == CharacterAlive && c.Stats.Health > 0
}

func (c *Character) IsDead() bool {
	return c.State == CharacterDead || c.Stats.Health <= 0
}

func (c *Character) UpdatePlayTime() {
	if !c.LastPlayed.IsZero() {
		c.PlayTime += time.Since(c.LastPlayed)
	}
	c.LastPlayed = time.Now()
}

func calculateStartingStats(race *Race, class *Class) *CharacterStats {
	stats := &CharacterStats{
		Strength:     10,
		Dexterity:    10,
		Intelligence: 10,
		Constitution: 10,
		Wisdom:       10,
		Charisma:     10,
	}
	
	if race != nil {
		stats.Strength += race.StatModifiers.Strength
		stats.Dexterity += race.StatModifiers.Dexterity
		stats.Intelligence += race.StatModifiers.Intelligence
		stats.Constitution += race.StatModifiers.Constitution
		stats.Wisdom += race.StatModifiers.Wisdom
		stats.Charisma += race.StatModifiers.Charisma
	}
	
	stats.MaxHealth = stats.Constitution * 10
	stats.Health = stats.MaxHealth
	stats.MaxMana = stats.Intelligence * 5
	stats.Mana = stats.MaxMana
	stats.MaxStamina = stats.Constitution * 5
	stats.Stamina = stats.MaxStamina
	
	return stats
}