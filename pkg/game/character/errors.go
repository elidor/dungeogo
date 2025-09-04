package character

import "errors"

var (
	ErrRaceNotFound     = errors.New("race not found")
	ErrClassNotFound    = errors.New("class not found")
	ErrInvalidCharacter = errors.New("invalid character")
	ErrCharacterDead    = errors.New("character is dead")
	ErrSkillNotFound    = errors.New("skill not found")
)