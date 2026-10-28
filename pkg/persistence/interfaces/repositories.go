package interfaces

import (
	"errors"

	"github.com/elidor/dungeogo/pkg/game/character"
	"github.com/elidor/dungeogo/pkg/game/items"
	"github.com/elidor/dungeogo/pkg/game/player"
	"github.com/elidor/dungeogo/pkg/game/world"
)

// ErrRoomNotFound distinguishes a missing world location from a storage failure.
var ErrRoomNotFound = errors.New("room definition not found")

type PlayerRepository interface {
	CreatePlayer(player *player.Player) error
	GetPlayer(playerID string) (*player.Player, error)
	GetPlayerByUsername(username string) (*player.Player, error)
	GetPlayerByEmail(email string) (*player.Player, error)
	UpdatePlayer(player *player.Player) error
	UpdatePlayerLogin(playerID string) error
	DeletePlayer(playerID string) error
}

type CharacterRepository interface {
	CreateCharacter(character *character.Character) error
	GetCharacter(characterID string) (*character.Character, error)
	GetCharactersByPlayer(playerID string) ([]*CharacterSummary, error)
	UpdateCharacter(character *character.Character) error
	DeleteCharacter(characterID string) error
	UpdateCharacterStats(characterID string, stats *character.CharacterStats) error
	UpdateCharacterLocation(characterID string, location *character.Location) error
	SaveCharacterSkills(characterID string, skills *character.SkillSet) error
}

type ItemRepository interface {
	CreateItemInstance(item *items.ItemInstance) error
	GetItemInstance(itemID string) (*items.ItemInstance, error)
	// UpdateItemInstance saves an Item's own fields. It never changes who holds
	// the Item; use TransferItem for that.
	UpdateItemInstance(item *items.ItemInstance) error
	DeleteItemInstance(itemID string) error
	GetPlayerItems(characterID string) ([]*items.ItemInstance, error)
	GetRoomItems(roomID string) ([]*items.ItemInstance, error)
	// TransferItem moves an Item from the Owner the caller expects it to have
	// to a new Owner in one conditional write. It returns moved=false, changing
	// nothing, if the Item doesn't exist or is no longer held by from, e.g.
	// because a competing action moved it first.
	TransferItem(itemID string, from, to items.Owner) (moved bool, err error)
}

type WorldRepository interface {
	SaveRoomDefinition(definition *RoomDefinition) error
	LoadRoomDefinition(roomID string) (*RoomDefinition, error)
	SaveRoomState(roomID string, state *RoomState) error
	LoadRoomState(roomID string) (*RoomState, error)
	SaveNPCState(npcID string, state *NPCState) error
	LoadNPCState(npcID string) (*NPCState, error)
	SaveWorldEvent(event *WorldEvent) error
	GetActiveWorldEvents() ([]*WorldEvent, error)
}

type CharacterSummary struct {
	ID         string
	Name       string
	Race       string
	Class      string
	Level      int
	Location   string
	LastPlayed string
	IsAlive    bool
}

// RoomDefinition is retained as an alias for existing repository callers.
type RoomDefinition = world.Room

type RoomState struct {
	ID         string
	Items      []string
	NPCs       []string
	Players    []string
	Flags      map[string]interface{}
	LastUpdate string
}

type NPCState struct {
	ID         string
	TemplateID string
	Health     int
	Location   *character.Location
	Inventory  []string
	State      string
	LastUpdate string
}

type WorldEvent struct {
	ID          string
	Type        string
	Description string
	StartTime   string
	EndTime     string
	Data        map[string]interface{}
}

type RepositoryManager interface {
	Players() PlayerRepository
	Characters() CharacterRepository
	Items() ItemRepository
	World() WorldRepository
	Close() error
}

// OnlineCharacter is retained as an alias for existing presence callers.
type OnlineCharacter = world.OnlineCharacter
