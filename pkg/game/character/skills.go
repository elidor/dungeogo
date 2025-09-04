package character

import (
	"time"
)

type SkillSet struct {
	Skills map[SkillType]*Skill
}

type Skill struct {
	Type        SkillType
	Level       int
	Experience  int
	Modifiers   []SkillModifier
	LastUsed    time.Time
	Trainers    []string
}

type SkillType int

const (
	SkillSwords SkillType = iota
	SkillAxes
	SkillMaces
	SkillDaggers
	SkillArchery
	SkillCrossbows
	SkillShields
	SkillDodge
	SkillParry
	SkillMagic
	SkillHealing
	SkillEvocation
	SkillDivination
	SkillLockpicking
	SkillStealth
	SkillCrafting
	SkillFishing
	SkillMining
)

type SkillModifier struct {
	Source string
	Value  int
	Type   ModifierType
}

type ModifierType int

const (
	ModifierBonus ModifierType = iota
	ModifierPenalty
	ModifierMultiplier
)

func NewSkillSet() *SkillSet {
	skills := make(map[SkillType]*Skill)
	
	for skillType := SkillSwords; skillType <= SkillMining; skillType++ {
		skills[skillType] = &Skill{
			Type:       skillType,
			Level:      0,
			Experience: 0,
			Modifiers:  []SkillModifier{},
			Trainers:   []string{},
		}
	}
	
	return &SkillSet{
		Skills: skills,
	}
}

func (ss *SkillSet) GetSkill(skillType SkillType) *Skill {
	if skill, exists := ss.Skills[skillType]; exists {
		return skill
	}
	return nil
}

func (ss *SkillSet) GetSkillLevel(skillType SkillType) int {
	skill := ss.GetSkill(skillType)
	if skill == nil {
		return 0
	}
	return skill.Level
}

func (ss *SkillSet) GetEffectiveSkillLevel(skillType SkillType) int {
	skill := ss.GetSkill(skillType)
	if skill == nil {
		return 0
	}
	
	effective := skill.Level
	for _, modifier := range skill.Modifiers {
		switch modifier.Type {
		case ModifierBonus:
			effective += modifier.Value
		case ModifierPenalty:
			effective -= modifier.Value
		case ModifierMultiplier:
			effective = (effective * modifier.Value) / 100
		}
	}
	
	return effective
}

func (ss *SkillSet) AddExperience(skillType SkillType, exp int) bool {
	skill := ss.GetSkill(skillType)
	if skill == nil {
		return false
	}
	
	skill.Experience += exp
	skill.LastUsed = time.Now()
	
	return ss.checkLevelUp(skill)
}

func (ss *SkillSet) checkLevelUp(skill *Skill) bool {
	expNeeded := ss.experienceNeededForLevel(skill.Level + 1)
	if skill.Experience >= expNeeded {
		skill.Level++
		return true
	}
	return false
}

func (ss *SkillSet) experienceNeededForLevel(level int) int {
	if level <= 0 {
		return 0
	}
	return level * level * 100
}

func (ss *SkillSet) AddModifier(skillType SkillType, modifier SkillModifier) {
	skill := ss.GetSkill(skillType)
	if skill != nil {
		skill.Modifiers = append(skill.Modifiers, modifier)
	}
}

func (ss *SkillSet) RemoveModifier(skillType SkillType, source string) {
	skill := ss.GetSkill(skillType)
	if skill == nil {
		return
	}
	
	for i, modifier := range skill.Modifiers {
		if modifier.Source == source {
			skill.Modifiers = append(skill.Modifiers[:i], skill.Modifiers[i+1:]...)
			break
		}
	}
}

func GetSkillName(skillType SkillType) string {
	names := map[SkillType]string{
		SkillSwords:      "Swords",
		SkillAxes:        "Axes",
		SkillMaces:       "Maces",
		SkillDaggers:     "Daggers",
		SkillArchery:     "Archery",
		SkillCrossbows:   "Crossbows",
		SkillShields:     "Shields",
		SkillDodge:       "Dodge",
		SkillParry:       "Parry",
		SkillMagic:       "Magic",
		SkillHealing:     "Healing",
		SkillEvocation:   "Evocation",
		SkillDivination:  "Divination",
		SkillLockpicking: "Lockpicking",
		SkillStealth:     "Stealth",
		SkillCrafting:    "Crafting",
		SkillFishing:     "Fishing",
		SkillMining:      "Mining",
	}
	
	if name, exists := names[skillType]; exists {
		return name
	}
	return "Unknown"
}