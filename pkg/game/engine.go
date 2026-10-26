package game

import (
	"fmt"

	"github.com/elidor/dungeogo/pkg/commands"
	"github.com/elidor/dungeogo/pkg/game/world"
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

// ProcessCommand preserves callers that consume only the actor's responses.
func (e *Engine) ProcessCommand(characterID, input string) ([]string, error) {
	result, err := e.ProcessAction(characterID, input)
	return result.Replies(characterID), err
}

func (e *Engine) ProcessAction(characterID, input string) (world.Result, error) {
	actor, err := e.repoManager.Characters().GetCharacter(characterID)
	if err != nil {
		return world.Result{}, fmt.Errorf("character not found: %w", err)
	}
	cmd := e.parser.Parse(input, actor.PlayerID, characterID)
	result, err := e.executor.ExecuteAction(cmd)
	if err != nil {
		return world.Result{}, fmt.Errorf("command execution failed: %w", err)
	}
	return result, nil
}

func (e *Engine) SetPresence(presence world.Presence) { e.executor.SetPresence(presence) }

func (e *Engine) GetCharacterState(characterID string) (interface{}, error) {
	character, err := e.repoManager.Characters().GetCharacter(characterID)
	if err != nil {
		return nil, fmt.Errorf("failed to get character state: %w", err)
	}

	return character, nil
}

// SetOnlineCharacters supplies who with live process-owned presence.
func (e *Engine) SetOnlineCharacters(source func() []interfaces.OnlineCharacter) {
	e.executor.SetOnlineCharacters(source)
}
