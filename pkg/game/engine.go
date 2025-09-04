package game

import (
	"fmt"
	
	"github.com/elidor/dungeogo/pkg/commands"
	"github.com/elidor/dungeogo/pkg/persistence/interfaces"
)

type Engine struct {
	repoManager interfaces.RepositoryManager
	parser      *commands.Parser
	executor    *commands.Executor
}

func NewEngine(repoManager interfaces.RepositoryManager) *Engine {
	parser := commands.NewParser()
	executor := commands.NewExecutor(repoManager)
	
	return &Engine{
		repoManager: repoManager,
		parser:      parser,
		executor:    executor,
	}
}

func (e *Engine) ProcessCommand(characterID string, input string) ([]string, error) {
	// Get character to validate it exists and get player ID
	character, err := e.repoManager.Characters().GetCharacter(characterID)
	if err != nil {
		return nil, fmt.Errorf("character not found: %w", err)
	}
	
	// Parse the command
	cmd := e.parser.Parse(input, character.PlayerID, characterID)
	
	// Execute the command
	responses, err := e.executor.Execute(cmd)
	if err != nil {
		return nil, fmt.Errorf("command execution failed: %w", err)
	}
	
	return responses, nil
}

func (e *Engine) GetCharacterState(characterID string) (interface{}, error) {
	character, err := e.repoManager.Characters().GetCharacter(characterID)
	if err != nil {
		return nil, fmt.Errorf("failed to get character state: %w", err)
	}
	
	return character, nil
}