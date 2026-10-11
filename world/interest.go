package world

import "slices"

// Interest says which part of the world a rule reads. The host only sends a plugin the state and
// the changes its rules are interested in.
type Interest struct {
	Everything  bool     `json:"everything,omitempty"`
	Kinds       []string `json:"kinds,omitempty"`      // entities of these kinds
	Properties  []string `json:"properties,omitempty"` // entities with any of these properties
	Fields      []string `json:"fields,omitempty"`
	Environment bool     `json:"environment,omitempty"`
	Events      []string `json:"events,omitempty"` // events with these names ("*" = all)
}

// Join returns an interest that covers both.
func (interest Interest) Join(other Interest) Interest {
	return Interest{
		Everything:  interest.Everything || other.Everything,
		Kinds:       joinNames(interest.Kinds, other.Kinds),
		Properties:  joinNames(interest.Properties, other.Properties),
		Fields:      joinNames(interest.Fields, other.Fields),
		Environment: interest.Environment || other.Environment,
		Events:      joinNames(interest.Events, other.Events),
	}
}

// Covers tells whether a change concerns this interest. The world is the state before the change,
// so a removed or modified entity can still be looked up.
func (interest Interest) Covers(change Change, before *World) bool {
	switch change.Operation {
	case Add:
		return interest.coversEntity(change.Entity)
	case Remove:
		existing := before.Entity(change.Identifier)
		return existing != nil && interest.coversEntity(existing)
	case Modify:
		return interest.coversModification(change.Entity, before)
	case SetField:
		return change.Field != nil && interest.coversField(change.Field.Name)
	case SetEnvironment:
		return interest.wantsEnvironment()
	case Emit:
		return change.Event != nil && interest.coversEvent(change.Event.Name)
	}
	return false
}

// coversModification is true when the entity was covered before the change or will be after it.
func (interest Interest) coversModification(patch *Entity, before *World) bool {
	if patch == nil {
		return false
	}
	existing := before.Entity(patch.Identifier)
	if existing == nil {
		return false
	}
	if interest.coversEntity(existing) {
		return true
	}
	after := existing.Copy()
	after.merge(patch)
	return interest.coversEntity(after)
}

func (interest Interest) coversEntity(entity *Entity) bool {
	if entity == nil {
		return false
	}
	if interest.Everything || slices.Contains(interest.Kinds, entity.Kind) {
		return true
	}
	return slices.ContainsFunc(interest.Properties, entity.HasProperty)
}

func (interest Interest) coversField(name string) bool {
	return interest.Everything || slices.Contains(interest.Fields, name)
}

func (interest Interest) wantsEnvironment() bool { return interest.Everything || interest.Environment }

func (interest Interest) coversEvent(name string) bool {
	return interest.Everything || slices.Contains(interest.Events, "*") || slices.Contains(interest.Events, name)
}

func joinNames(first, second []string) []string {
	joined := append([]string(nil), first...)
	for _, name := range second {
		if !slices.Contains(joined, name) {
			joined = append(joined, name)
		}
	}
	return joined
}
