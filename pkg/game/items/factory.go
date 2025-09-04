package items

import (
	"fmt"
	"github.com/google/uuid"
)

type ItemFactory struct {
	registry *ItemRegistry
}

func NewItemFactory() *ItemFactory {
	return &ItemFactory{
		registry: NewItemRegistry(),
	}
}

func (f *ItemFactory) CreateInstance(templateID, ownerID string, quantity int) (*ItemInstance, error) {
	template, err := f.registry.GetTemplate(templateID)
	if err != nil {
		return nil, fmt.Errorf("failed to create item instance: %w", err)
	}
	
	if quantity <= 0 {
		quantity = 1
	}
	
	if !template.IsStackable() && quantity > 1 {
		return nil, fmt.Errorf("item %s is not stackable", template.Name)
	}
	
	if template.IsStackable() && quantity > template.StackSize {
		return nil, fmt.Errorf("quantity %d exceeds max stack size %d for item %s", 
			quantity, template.StackSize, template.Name)
	}
	
	instance := &ItemInstance{
		ID:           generateItemID(),
		TemplateID:   templateID,
		OwnerID:      ownerID,
		Quantity:     quantity,
		Durability:   template.Durability,
		Enchantments: []Enchantment{},
		Modifications: make(map[string]interface{}),
	}
	
	return instance, nil
}

func (f *ItemFactory) GetTemplate(templateID string) (*ItemTemplate, error) {
	return f.registry.GetTemplate(templateID)
}

func (f *ItemFactory) GetAllTemplates() map[string]*ItemTemplate {
	return f.registry.GetAllTemplates()
}

func (f *ItemFactory) GetTemplatesByType(itemType ItemType) []*ItemTemplate {
	return f.registry.GetTemplatesByType(itemType)
}

func (f *ItemFactory) RegisterTemplate(template *ItemTemplate) error {
	return f.registry.RegisterTemplate(template)
}

func (f *ItemFactory) CreateEnchantedInstance(templateID, ownerID string, enchantments []Enchantment) (*ItemInstance, error) {
	instance, err := f.CreateInstance(templateID, ownerID, 1)
	if err != nil {
		return nil, err
	}
	
	template, err := f.GetTemplate(templateID)
	if err != nil {
		return nil, err
	}
	
	if !template.Enchantable {
		return nil, fmt.Errorf("item %s cannot be enchanted", template.Name)
	}
	
	for _, enchantment := range enchantments {
		instance.AddEnchantment(enchantment)
	}
	
	return instance, nil
}

func (f *ItemFactory) CreateCustomInstance(templateID, ownerID, customName string) (*ItemInstance, error) {
	instance, err := f.CreateInstance(templateID, ownerID, 1)
	if err != nil {
		return nil, err
	}
	
	instance.CustomName = customName
	return instance, nil
}

func generateItemID() string {
	return uuid.New().String()
}