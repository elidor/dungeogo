package items

import "fmt"

// OwnerKind says whether an Item is held by a Room or a Character.
type OwnerKind string

const (
	OwnerRoom      OwnerKind = "room"
	OwnerCharacter OwnerKind = "character"
)

// Valid reports whether k is a known owner kind. The empty kind is not valid.
func (k OwnerKind) Valid() bool {
	return k == OwnerRoom || k == OwnerCharacter
}

// Owner is the one Room or Character that holds an Item.
// A zero Kind means the kind is not yet known; storage resolves it once.
type Owner struct {
	Kind OwnerKind
	ID   string
}

// RoomOwner is the Owner for an Item lying in the Room with the given ID.
func RoomOwner(roomID string) Owner { return Owner{Kind: OwnerRoom, ID: roomID} }

// CharacterOwner is the Owner for an Item carried by the Character with the given ID.
func CharacterOwner(characterID string) Owner {
	return Owner{Kind: OwnerCharacter, ID: characterID}
}

func (o Owner) String() string {
	if o.Kind == "" {
		return fmt.Sprintf("owner %q", o.ID)
	}
	return fmt.Sprintf("%s %q", o.Kind, o.ID)
}
