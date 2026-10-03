package world

import (
	"fmt"
	"slices"
)

// World holds the current state. It only changes through Apply.
type World struct {
	Time        float64
	Environment Environment
	entities    map[Identifier]*Entity
	order       []Identifier // entities in the order they were added, for stable iteration
	fields      map[string]*Field
}

// New returns an empty world.
func New() *World {
	return &World{entities: map[Identifier]*Entity{}, fields: map[string]*Field{}}
}

// Entity returns an entity, or nil when it does not exist. Callers must not change it.
func (world *World) Entity(identifier Identifier) *Entity { return world.entities[identifier] }

// Entities returns every entity in the order they were added. Callers must not change them.
func (world *World) Entities() []*Entity {
	entities := make([]*Entity, 0, len(world.order))
	for _, identifier := range world.order {
		entities = append(entities, world.entities[identifier])
	}
	return entities
}

// EntitiesOfKind returns the entities of a kind, in the order they were added.
func (world *World) EntitiesOfKind(kind string) []*Entity {
	var entities []*Entity
	for _, identifier := range world.order {
		if entity := world.entities[identifier]; entity.Kind == kind {
			entities = append(entities, entity)
		}
	}
	return entities
}

// Field returns a field, or nil when it does not exist. Callers must not change it.
func (world *World) Field(name string) *Field { return world.fields[name] }

// Fields returns every field, sorted by name.
func (world *World) Fields() []*Field {
	names := make([]string, 0, len(world.fields))
	for name := range world.fields {
		names = append(names, name)
	}
	slices.Sort(names)
	fields := make([]*Field, 0, len(names))
	for _, name := range names {
		fields = append(fields, world.fields[name])
	}
	return fields
}

// Apply applies the changes in order and moves the clock to the set's time. It stops at the
// first change that cannot be applied; the changes before it stay applied.
func (world *World) Apply(set ChangeSet) error {
	for index, change := range set.Changes {
		if err := world.ApplyChange(change); err != nil {
			return fmt.Errorf("change %d (%s) from %q: %w", index, change.Operation, set.Origin, err)
		}
	}
	if set.Time > world.Time {
		world.Time = set.Time
	}
	return nil
}

// ApplyChange applies one change.
func (world *World) ApplyChange(change Change) error {
	switch change.Operation {
	case Add:
		return world.add(change.Entity)
	case Remove:
		return world.remove(change.Identifier)
	case Modify:
		return world.modify(change.Entity)
	case SetField:
		return world.setField(change.Field)
	case SetEnvironment:
		return world.setEnvironment(change.Environment)
	case Emit, ClearArea:
		return nil
	}
	return fmt.Errorf("unknown operation %q", change.Operation)
}

func (world *World) add(entity *Entity) error {
	if entity == nil || entity.Identifier == "" {
		return fmt.Errorf("an added entity needs an identifier")
	}
	if _, exists := world.entities[entity.Identifier]; exists {
		return fmt.Errorf("entity %q already exists", entity.Identifier)
	}
	world.entities[entity.Identifier] = entity.Copy()
	world.order = append(world.order, entity.Identifier)
	return nil
}

// remove takes away an entity and, with it, every entity attached to it.
func (world *World) remove(identifier Identifier) error {
	if _, exists := world.entities[identifier]; !exists {
		return fmt.Errorf("entity %q does not exist", identifier)
	}
	removed := world.withDescendants(identifier)
	for removedIdentifier := range removed {
		delete(world.entities, removedIdentifier)
	}
	world.order = slices.DeleteFunc(world.order, func(candidate Identifier) bool { return removed[candidate] })
	return nil
}

// withDescendants returns an entity and every entity attached to it, directly or not.
func (world *World) withDescendants(identifier Identifier) map[Identifier]bool {
	found := map[Identifier]bool{identifier: true}
	for grew := true; grew; {
		grew = false
		for _, candidate := range world.order {
			if parent := world.entities[candidate].Parent; !found[candidate] && found[parent] {
				found[candidate], grew = true, true
			}
		}
	}
	return found
}

func (world *World) modify(patch *Entity) error {
	if patch == nil {
		return fmt.Errorf("a modification needs an entity patch")
	}
	entity, exists := world.entities[patch.Identifier]
	if !exists {
		return fmt.Errorf("entity %q does not exist", patch.Identifier)
	}
	entity.merge(patch)
	return nil
}

func (world *World) setField(patch *FieldPatch) error {
	if patch == nil || patch.Name == "" {
		return fmt.Errorf("a field change needs a field name")
	}
	field, exists := world.fields[patch.Name]
	if !exists {
		created, err := newField(patch)
		if err != nil {
			return err
		}
		field = created
		world.fields[patch.Name] = field
	}
	return field.apply(patch)
}

func (world *World) setEnvironment(patch *Environment) error {
	if patch == nil {
		return fmt.Errorf("an environment change needs a patch")
	}
	world.Environment.merge(patch)
	return nil
}

// Snapshot returns changes that rebuild this world from an empty one, keeping only what an
// interest covers.
func (world *World) Snapshot(interest Interest) ChangeSet {
	set := ChangeSet{Time: world.Time}
	if interest.wantsEnvironment() {
		environment := world.Environment.Copy()
		set.Changes = append(set.Changes, ChangeEnvironment(&environment))
	}
	for _, entity := range world.Entities() {
		if interest.coversEntity(entity) {
			set.Changes = append(set.Changes, AddEntity(entity.Copy()))
		}
	}
	for _, field := range world.Fields() {
		if interest.coversField(field.Name) {
			set.Changes = append(set.Changes, SetFieldValues(fieldAsPatch(field)))
		}
	}
	return set
}

func fieldAsPatch(field *Field) *FieldPatch {
	extent := field.Extent
	return &FieldPatch{Name: field.Name, Columns: field.Columns, Rows: field.Rows, Extent: &extent,
		Values: append([]float32(nil), field.Values...)}
}
