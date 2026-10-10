package plugin

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/akrck02/go-simulator/protocol"
	"github.com/akrck02/go-simulator/world"
)

// Serve talks to the host over standard input and output until the host stops it.
func (plugin *Plugin) Serve() error {
	return plugin.ServeConnection(protocol.NewConnection(os.Stdin, os.Stdout))
}

// ServeConnection talks to the host over a connection until the host stops it or the
// connection closes.
func (plugin *Plugin) ServeConnection(connection *protocol.Connection) error {
	session := &session{plugin: plugin, connection: connection, world: world.New(), instances: map[string]Rule{}}
	for {
		message, err := connection.Receive()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if message.Kind == protocol.Stop {
			return nil
		}
		if err := session.answer(message); err != nil {
			if sendErr := connection.Send(protocol.Failure, protocol.FailureReply{Message: err.Error()}); sendErr != nil {
				return sendErr
			}
		}
	}
}

// session is the plugin's side of one conversation: its copy of the world and its rule instances.
type session struct {
	plugin     *Plugin
	connection *protocol.Connection
	world      *world.World
	instances  map[string]Rule
	order      []string      // instances in the order they were attached
	events     []world.Event // events other plugins emitted, received with the current request
}

func (session *session) answer(message protocol.Message) error {
	switch message.Kind {
	case protocol.Describe:
		return session.connection.Send(protocol.Description, session.plugin.describe())
	case protocol.Attach:
		return session.attach(message)
	case protocol.Start:
		return session.start(message)
	case protocol.Step:
		return session.step(message)
	}
	return fmt.Errorf("plugin %q does not understand %q messages", session.plugin.Name, message.Kind)
}

func (session *session) attach(message protocol.Message) error {
	var request protocol.AttachRequest
	if err := message.Decode(&request); err != nil {
		return err
	}
	definition, found := session.plugin.definition(request.Rule)
	if !found {
		return fmt.Errorf("plugin %q has no rule %q", session.plugin.Name, request.Rule)
	}
	rule, err := definition.New(Parameters(request.Parameters))
	if err != nil {
		return fmt.Errorf("rule %q: %w", request.Rule, err)
	}
	if _, exists := session.instances[request.Instance]; !exists {
		session.order = append(session.order, request.Instance)
	}
	session.instances[request.Instance] = rule
	return session.connection.Send(protocol.Attached, protocol.AttachReply{Instance: request.Instance})
}

func (session *session) start(message protocol.Message) error {
	var request protocol.StartRequest
	if err := message.Decode(&request); err != nil {
		return err
	}
	if err := session.world.Apply(request.Snapshot); err != nil {
		return fmt.Errorf("start snapshot: %w", err)
	}
	session.events = eventsOf(request.Snapshot.Changes)
	reply := world.ChangeSet{Origin: session.plugin.Name, Time: session.world.Time}
	for _, instance := range session.order {
		starter, startsSomething := session.instances[instance].(Starter)
		if !startsSomething {
			continue
		}
		context := session.context("", instance, session.world.Time, 0)
		if err := runStart(starter, context); err != nil {
			return err
		}
		reply.Changes = append(reply.Changes, context.changes...)
	}
	if err := session.afterStep("", session.world.Time, 0, &reply); err != nil {
		return err
	}
	return session.connection.Send(protocol.Changes, &reply)
}

// afterStep runs the plugin's AfterStep hook, if any, and adds its changes to the reply.
func (session *session) afterStep(stage string, time, step float64, reply *world.ChangeSet) error {
	if session.plugin.AfterStep == nil {
		return nil
	}
	context := session.context(stage, "", time, step)
	if err := session.plugin.AfterStep(context); err != nil {
		return fmt.Errorf("after the step: %w", err)
	}
	if err := context.failureOf(); err != nil {
		return err
	}
	reply.Changes = append(reply.Changes, context.changes...)
	return nil
}

func (session *session) step(message protocol.Message) error {
	var request protocol.StepRequest
	if err := message.Decode(&request); err != nil {
		return err
	}
	if err := session.world.Apply(world.ChangeSet{Changes: request.Incoming}); err != nil {
		return fmt.Errorf("incoming changes: %w", err)
	}
	session.events = eventsOf(request.Incoming)
	session.world.Time = request.Time
	reply := world.ChangeSet{Origin: session.plugin.Name, Stage: request.Stage, Time: request.Time}
	for _, instance := range request.Instances {
		rule, found := session.instances[instance]
		if !found {
			return fmt.Errorf("no rule instance %q", instance)
		}
		context := session.context(request.Stage, instance, request.Time, request.Step)
		if err := runStep(rule, context); err != nil {
			return err
		}
		reply.Changes = append(reply.Changes, context.changes...)
	}
	if err := session.afterStep(request.Stage, request.Time, request.Step, &reply); err != nil {
		return err
	}
	return session.connection.Send(protocol.Changes, &reply)
}

func (session *session) context(stage, instance string, time, step float64) *Context {
	return &Context{World: session.world, Stage: stage, Time: time, Step: step, Instance: instance, Events: session.events}
}

// eventsOf returns the events among some changes, in order.
func eventsOf(changes []world.Change) []world.Event {
	var events []world.Event
	for _, change := range changes {
		if change.Operation == world.Emit && change.Event != nil {
			events = append(events, *change.Event)
		}
	}
	return events
}

func runStart(starter Starter, context *Context) error {
	if err := starter.Start(context); err != nil {
		return fmt.Errorf("%s: %w", context.Instance, err)
	}
	return context.failureOf()
}

func runStep(rule Rule, context *Context) error {
	if err := rule.Step(context); err != nil {
		return fmt.Errorf("%s: %w", context.Instance, err)
	}
	return context.failureOf()
}

func (context *Context) failureOf() error {
	if context.failure != nil {
		return fmt.Errorf("%s made a change that cannot apply: %w", context.Instance, context.failure)
	}
	return nil
}
