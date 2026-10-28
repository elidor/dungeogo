package items

import "testing"

func TestOwnerConstructors(t *testing.T) {
	if got := RoomOwner("starting_room"); got != (Owner{Kind: OwnerRoom, ID: "starting_room"}) {
		t.Fatalf("RoomOwner = %+v", got)
	}
	if got := CharacterOwner("c1"); got != (Owner{Kind: OwnerCharacter, ID: "c1"}) {
		t.Fatalf("CharacterOwner = %+v", got)
	}
}

func TestOwnerKindValid(t *testing.T) {
	for kind, want := range map[OwnerKind]bool{OwnerRoom: true, OwnerCharacter: true, "": false, "npc": false} {
		if got := kind.Valid(); got != want {
			t.Errorf("OwnerKind(%q).Valid() = %v, want %v", kind, got, want)
		}
	}
}

func TestOwnerString(t *testing.T) {
	for owner, want := range map[Owner]string{
		RoomOwner("starting_room"): `room "starting_room"`,
		CharacterOwner("c1"):       `character "c1"`,
		{ID: "x"}:                  `owner "x"`,
	} {
		if got := owner.String(); got != want {
			t.Errorf("%+v.String() = %q, want %q", owner, got, want)
		}
	}
}
