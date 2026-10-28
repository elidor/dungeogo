package items

import (
	"errors"
	"testing"
)

func TestParseTarget(t *testing.T) {
	for input, want := range map[string]Target{
		"sword":      {Keyword: "sword", Ordinal: 1},
		"  Sword ":   {Keyword: "sword", Ordinal: 1},
		"2.sword":    {Keyword: "sword", Ordinal: 2},
		"10.SW":      {Keyword: "sw", Ordinal: 10},
		"st.ives":    {Keyword: "st.ives", Ordinal: 1},
		"traveler's": {Keyword: "traveler's", Ordinal: 1},
	} {
		got, err := ParseTarget(input)
		if err != nil || got != want {
			t.Errorf("ParseTarget(%q) = %+v, %v; want %+v", input, got, err, want)
		}
	}
	for _, input := range []string{"", "   ", "0.sword", "-1.sword", "2.", "training sword"} {
		if got, err := ParseTarget(input); !errors.Is(err, ErrInvalidTarget) {
			t.Errorf("ParseTarget(%q) = %+v, %v; want ErrInvalidTarget", input, got, err)
		}
	}
}

func TestTargetMatchesStartOfAnyWord(t *testing.T) {
	for keyword, want := range map[string]bool{
		"sword": true, "sw": true, "training": true, "t": true,
		"word": false, "swords": false, "trainingsword": false,
	} {
		if got := (Target{Keyword: keyword, Ordinal: 1}).Matches("Training Sword"); got != want {
			t.Errorf("%q matches Training Sword = %v, want %v", keyword, got, want)
		}
	}
	if !(Target{Keyword: "traveler", Ordinal: 1}).Matches("Traveler's Blade") {
		t.Error("possessive word should match its prefix")
	}
}

func TestTargetPickCountsMatchesInListOrder(t *testing.T) {
	registry := NewItemRegistry()
	list := []*ItemInstance{
		{ID: "a", TemplateID: "rusty_sword", CustomName: "Training Sword"},
		{ID: "b", TemplateID: "travel_lantern"},
		{ID: "c", TemplateID: "rusty_sword"},
		{ID: "d", TemplateID: "rusty_sword", CustomName: "Traveler's Blade"},
	}
	for input, wantID := range map[string]string{
		"sword": "a", "1.sword": "a", "2.sword": "c", "lantern": "b",
		"2.t": "b", "3.t": "d", "blade": "d", "rusty": "c",
	} {
		target, err := ParseTarget(input)
		if err != nil {
			t.Fatal(err)
		}
		got, ok := target.Pick(list, registry)
		if !ok || got.ID != wantID {
			t.Errorf("%q picked %+v (found=%v), want %s", input, got, ok, wantID)
		}
	}
	for _, input := range []string{"3.sword", "axe", "5.t"} {
		target, _ := ParseTarget(input)
		if got, ok := target.Pick(list, registry); ok {
			t.Errorf("%q picked %+v, want no match", input, got)
		}
	}
}

func TestShortDescription(t *testing.T) {
	registry := NewItemRegistry()
	// A template with no name falls back to its ID rather than an empty name.
	if err := registry.RegisterTemplate(&ItemTemplate{ID: "nameless"}); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		item ItemInstance
		want string
	}{
		{ItemInstance{TemplateID: "rusty_sword", CustomName: "Training Sword"}, "a Training Sword"},
		{ItemInstance{TemplateID: "travel_lantern"}, "a Travel Lantern"},
		{ItemInstance{TemplateID: "rusty_sword", CustomName: "Iron Key"}, "an Iron Key"},
		{ItemInstance{TemplateID: "rusty_sword", CustomName: "Traveler's Blade"}, "a Traveler's Blade"},
		{ItemInstance{}, "an Unknown Item"},
		{ItemInstance{TemplateID: "nameless"}, "a nameless"},
	} {
		if got := registry.ShortDescription(&tc.item); got != tc.want {
			t.Errorf("ShortDescription(%+v) = %q, want %q", tc.item, got, tc.want)
		}
	}
}
