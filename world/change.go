package world

// Operation says what a change does.
type Operation string

const (
	Add            Operation = "add"             // a new entity (Entity)
	Remove         Operation = "remove"          // an entity goes away with everything attached to it (Identifier)
	Modify         Operation = "modify"          // an entity changes (Entity as a patch)
	SetField       Operation = "set_field"       // a field is created or some of its cells change (Field)
	SetEnvironment Operation = "set_environment" // the environment changes (Environment as a patch)
	Emit           Operation = "emit"            // something happened worth telling (Event); the world does not change
)

// Change is one thing that happens to the world. Only the member that matches the operation is set.
type Change struct {
	Operation   Operation    `json:"op"`
	Entity      *Entity      `json:"entity,omitempty"`
	Identifier  Identifier   `json:"id,omitempty"`
	Field       *FieldPatch  `json:"field,omitempty"`
	Environment *Environment `json:"environment,omitempty"`
	Event       *Event       `json:"event,omitempty"`
}

// Event is something that happened: "second wall raised", with free data.
type Event struct {
	Name string         `json:"name"`
	Data map[string]any `json:"data,omitempty"`
}

// ChangeSet is what a plugin returns after a step: the changes, in order, at a simulated time.
type ChangeSet struct {
	Origin  string   `json:"origin,omitempty"` // plugin that made the changes
	Stage   string   `json:"stage,omitempty"`
	Time    float64  `json:"time"`
	Changes []Change `json:"changes"`
}

// AddEntity returns a change that adds an entity.
func AddEntity(entity *Entity) Change { return Change{Operation: Add, Entity: entity} }

// RemoveEntity returns a change that removes an entity.
func RemoveEntity(identifier Identifier) Change {
	return Change{Operation: Remove, Identifier: identifier}
}

// ModifyEntity returns a change that patches an entity.
func ModifyEntity(patch *Entity) Change { return Change{Operation: Modify, Entity: patch} }

// SetFieldValues returns a change that creates a field or sets some of its cells.
func SetFieldValues(patch *FieldPatch) Change { return Change{Operation: SetField, Field: patch} }

// ChangeEnvironment returns a change that patches the environment.
func ChangeEnvironment(patch *Environment) Change {
	return Change{Operation: SetEnvironment, Environment: patch}
}

// EmitEvent returns a change that tells something happened.
func EmitEvent(name string, data map[string]any) Change {
	return Change{Operation: Emit, Event: &Event{Name: name, Data: data}}
}
