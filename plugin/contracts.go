package plugin

import "github.com/akrck02/go-simulator/protocol"

// Helpers to describe a rule's contracts in one line each (see protocol/contracts.go):
//
//	Publishes: []protocol.OutputDescription{
//		plugin.Field("elevation", "m", "height of the ground"),
//		plugin.Kind("river", "a line with a width", plugin.Property("width", "world", "")),
//	},
//	Expects: []protocol.InputDescription{plugin.NeedsField("elevation", "m")},

// Field is a field a rule publishes, in a unit of protocol.Units.
func Field(name, unit, summary string) protocol.OutputDescription {
	return protocol.OutputDescription{Name: name, What: protocol.WhatField, Unit: unit, Summary: summary}
}

// IndexField is a field of categories whose names an event announces.
func IndexField(name, namesEvent, summary string) protocol.OutputDescription {
	return protocol.OutputDescription{Name: name, What: protocol.WhatField, Unit: "index", Names: namesEvent, Summary: summary}
}

// Kind is a kind of entities a rule publishes, with the properties they carry.
func Kind(name, summary string, properties ...protocol.OutputDescription) protocol.OutputDescription {
	return protocol.OutputDescription{Name: name, What: protocol.WhatKind, Summary: summary, Properties: properties}
}

// Event is an event a rule emits, with the members of its data.
func Event(name, summary string, members ...protocol.OutputDescription) protocol.OutputDescription {
	return protocol.OutputDescription{Name: name, What: protocol.WhatEvent, Summary: summary, Properties: members}
}

// Environment is a member of the environment a rule changes.
func Environment(name, summary string) protocol.OutputDescription {
	return protocol.OutputDescription{Name: name, What: protocol.WhatEnvironment, Summary: summary}
}

// Property is a property of a kind or a member of an event; unit is empty for what is not a
// quantity (a name, a class).
func Property(name, unit, summary string) protocol.OutputDescription {
	return protocol.OutputDescription{Name: name, Unit: unit, Summary: summary}
}

// NeedsField is a field a rule cannot work without, in a unit.
func NeedsField(name, unit string) protocol.InputDescription {
	return protocol.InputDescription{Name: name, What: protocol.WhatField, Unit: unit, Required: true}
}

// ReadsField is a field a rule uses when the world has it.
func ReadsField(name, unit string) protocol.InputDescription {
	return protocol.InputDescription{Name: name, What: protocol.WhatField, Unit: unit}
}

// ReadsKind is a kind of entities a rule uses when the world has them.
func ReadsKind(name string) protocol.InputDescription {
	return protocol.InputDescription{Name: name, What: protocol.WhatKind}
}

// ReadsEvent is an event a rule listens to.
func ReadsEvent(name string) protocol.InputDescription {
	return protocol.InputDescription{Name: name, What: protocol.WhatEvent}
}
