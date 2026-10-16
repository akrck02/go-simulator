// Package protocol is how the host and its plugins talk. A plugin is a separate program; the host
// starts it and they exchange messages over the plugin's standard input and output (its standard
// error is free for logs).
//
// Conversation:
//
//	host → describe            plugin → description  (name, version, rules it offers)
//	host → attach (per rule)   plugin → attached     (or an error)
//	host → start  (snapshot)   plugin → changes      (what the rules do when they start)
//	host → step   (repeated)   plugin → changes
//	host → stop                plugin exits
//
// Every message is a frame: a 4-byte little-endian length, a JSON header of that length
// ({"kind", "body", "attachments": [byte lengths]}) and then the attachments, one after another.
// Field values travel as attachments of little-endian float32: they belong, in order, to the
// set_field changes of the message in the order those appear. Any language that can read and
// write bytes can be a plugin.
package protocol

import "github.com/akrck02/go-simulator/world"

// Version is the protocol version plugins of this toolkit speak. Version 2 added the contracts of
// rules (Publishes and Expects; see contracts.go).
const Version = 2

// OldestVersion is the oldest protocol version the host still accepts. A version 1 plugin works
// as before; its rules are just not checked, because they do not describe their contracts.
const OldestVersion = 1

// Kind names a message.
type Kind string

const (
	Describe    Kind = "describe"
	Description Kind = "description"
	Attach      Kind = "attach"
	Attached    Kind = "attached"
	Start       Kind = "start"
	Step        Kind = "step"
	Changes     Kind = "changes"
	Stop        Kind = "stop"
	Failure     Kind = "error"
)

// PluginDescription tells the host who the plugin is and which rules it offers.
type PluginDescription struct {
	Protocol int               `json:"protocol"`
	Name     string            `json:"name"`
	Version  string            `json:"version"`
	Summary  string            `json:"summary,omitempty"`
	Rules    []RuleDescription `json:"rules"`
}

// RuleDescription describes a rule a plugin offers.
type RuleDescription struct {
	Name       string                 `json:"name"`
	Summary    string                 `json:"summary,omitempty"`
	Parameters []ParameterDescription `json:"parameters,omitempty"`
	Reads      world.Interest         `json:"reads"`
	Writes     []string               `json:"writes,omitempty"`    // kinds, fields or "environment" it changes
	Publishes  []OutputDescription    `json:"publishes,omitempty"` // what it publishes, with units (version 2)
	Expects    []InputDescription     `json:"expects,omitempty"`   // what it needs from what it reads (version 2)
	Extra      map[string]any         `json:"extra,omitempty"`
}

// ParameterDescription describes one parameter of a rule.
type ParameterDescription struct {
	Name    string `json:"name"`
	Type    string `json:"type"` // "number", "boolean", "text", "vector", "list", "map"
	Default any    `json:"default,omitempty"`
	Summary string `json:"summary,omitempty"`
}

// AttachRequest asks the plugin to set up one use of a rule with its parameters. The same rule
// can be attached several times with different instance names.
type AttachRequest struct {
	Instance   string         `json:"instance"`
	Rule       string         `json:"rule"`
	Parameters map[string]any `json:"parameters,omitempty"`
}

// AttachReply confirms an attachment.
type AttachReply struct {
	Instance string `json:"instance"`
}

// StartRequest gives the plugin the part of the world its rules read.
type StartRequest struct {
	Snapshot world.ChangeSet `json:"snapshot"`
}

// StepRequest asks the plugin to advance some rule instances. Incoming holds the changes other
// plugins made since the last message, so the plugin's copy of the world stays current.
type StepRequest struct {
	Stage     string         `json:"stage"`
	Time      float64        `json:"time"` // simulated time after the step, in the stage's unit
	Step      float64        `json:"step"` // how much time the step covers
	Instances []string       `json:"instances"`
	Incoming  []world.Change `json:"incoming,omitempty"`
}

// FailureReply reports an error; the host stops the simulation with it.
type FailureReply struct {
	Message string `json:"message"`
}
