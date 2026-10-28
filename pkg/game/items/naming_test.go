package items

import (
	"strings"
	"testing"
)

func TestDisplayNameAndLabel(t *testing.T) {
	registry := NewItemRegistry()
	for _, tc := range []struct {
		name        string
		item        ItemInstance
		displayName string
		label       string
	}{
		{"template name", ItemInstance{ID: "id-1", TemplateID: "rusty_sword", Quantity: 1}, "Rusty Sword", "Rusty Sword"},
		{"custom name wins", ItemInstance{ID: "id-2", TemplateID: "rusty_sword", CustomName: "Excalibur", Quantity: 1}, "Excalibur", "Excalibur"},
		{"lantern template", ItemInstance{ID: "id-3", TemplateID: "travel_lantern", Quantity: 1}, "Travel Lantern", "Travel Lantern"},
		{"stone template", ItemInstance{ID: "id-4", TemplateID: "smooth_stone", Quantity: 1}, "Smooth Stone", "Smooth Stone"},
		{"stack shows count", ItemInstance{ID: "id-5", TemplateID: "health_potion", Quantity: 3}, "Health Potion", "Health Potion x3"},
		{"unknown template", ItemInstance{ID: "id-6", TemplateID: "odd_trinket", Quantity: 1}, "odd trinket", "odd trinket"},
		{"no template", ItemInstance{ID: "id-7", Quantity: 1}, "Unknown Item", "Unknown Item"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := registry.DisplayName(&tc.item); got != tc.displayName {
				t.Errorf("DisplayName = %q, want %q", got, tc.displayName)
			}
			label := registry.Label(&tc.item)
			if label != tc.label {
				t.Errorf("Label = %q, want %q", label, tc.label)
			}
			if strings.Contains(label, tc.item.ID) {
				t.Errorf("Label %q exposes instance ID %q", label, tc.item.ID)
			}
		})
	}
}

func TestStarterItemTemplatesAreSingleItems(t *testing.T) {
	registry := NewItemRegistry()
	for _, id := range []string{"travel_lantern", "smooth_stone"} {
		template, err := registry.GetTemplate(id)
		if err != nil {
			t.Fatalf("starter template %q: %v", id, err)
		}
		if template.StackSize != 1 || template.IsStackable() {
			t.Errorf("starter template %q should not stack (stack size %d)", id, template.StackSize)
		}
	}
}
