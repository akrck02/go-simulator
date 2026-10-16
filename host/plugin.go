// Package host runs simulations: it starts plugin programs, attaches their rules, steps them on
// their stages' clocks and applies what they change to the world. Whoever drives the host (the
// simulator command, a batch run, a test) receives every change set and decides what to do with
// it: draw it, record it or write it to files.
package host

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/akrck02/go-simulator/protocol"
	"github.com/akrck02/go-simulator/world"
)

// stopWaiting is how long a plugin has to exit after being asked to stop.
const stopWaiting = 3 * time.Second

// Plugin is a running plugin as the host sees it.
type Plugin struct {
	Name        string // name in the rules file
	Description protocol.PluginDescription
	connection  *protocol.Connection
	process     *exec.Cmd
	finished    chan error
	interest    world.Interest // what its attached rules read
	pending     []world.Change // changes by other plugins, sent with the next step
}

// RulesDirectoryVariable is the environment variable that tells a plugin program where the rules
// file is, so relative paths in its parameters can start there (see plugin.Path).
const RulesDirectoryVariable = "GO_SIMULATOR_RULES_DIRECTORY"

// Launch starts a plugin program and asks it to describe itself. The command runs in a directory
// (empty = the current one); its standard error goes to the host's, prefixed with the name.
// rulesDirectory is passed to the program in RulesDirectoryVariable.
func Launch(name string, command []string, directory, rulesDirectory string) (*Plugin, error) {
	if len(command) == 0 {
		return nil, fmt.Errorf("plugin %q has no command", name)
	}
	process := exec.Command(command[0], command[1:]...)
	process.Dir = directory
	process.Env = append(os.Environ(), RulesDirectoryVariable+"="+rulesDirectory)
	process.Stderr = &prefixedWriter{prefix: "[" + name + "] ", target: os.Stderr, startOfLine: true}
	input, err := process.StdinPipe()
	if err != nil {
		return nil, err
	}
	output, err := process.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := process.Start(); err != nil {
		return nil, fmt.Errorf("starting plugin %q: %w", name, err)
	}
	plugin := &Plugin{Name: name, connection: protocol.NewConnection(output, input), process: process, finished: make(chan error, 1)}
	go func() { plugin.finished <- process.Wait() }()
	return plugin, plugin.describe()
}

// Connect runs a plugin inside the host, talking through in-memory pipes with the same protocol
// a program would use. It suits small built-in plugins and tests.
func Connect(name string, serve func(*protocol.Connection) error) (*Plugin, error) {
	hostReader, pluginWriter := io.Pipe()
	pluginReader, hostWriter := io.Pipe()
	plugin := &Plugin{Name: name, connection: protocol.NewConnection(hostReader, hostWriter), finished: make(chan error, 1)}
	go func() {
		err := serve(protocol.NewConnection(pluginReader, pluginWriter))
		pluginWriter.CloseWithError(io.EOF)
		plugin.finished <- err
	}()
	return plugin, plugin.describe()
}

func (plugin *Plugin) describe() error {
	if err := plugin.connection.Send(protocol.Describe, struct{}{}); err != nil {
		return fmt.Errorf("plugin %q: %w", plugin.Name, err)
	}
	if err := plugin.receive(protocol.Description, &plugin.Description); err != nil {
		return err
	}
	if speaks := plugin.Description.Protocol; speaks < protocol.OldestVersion || speaks > protocol.Version {
		return fmt.Errorf("plugin %q speaks protocol %d; the host speaks %d to %d", plugin.Name, speaks, protocol.OldestVersion, protocol.Version)
	}
	for _, rule := range plugin.Description.Rules {
		if err := rule.Check(); err != nil {
			return fmt.Errorf("plugin %q: %w", plugin.Name, err)
		}
	}
	return nil
}

// Rule returns the description of a rule the plugin offers.
func (plugin *Plugin) Rule(name string) (protocol.RuleDescription, bool) {
	for _, rule := range plugin.Description.Rules {
		if rule.Name == name {
			return rule, true
		}
	}
	return protocol.RuleDescription{}, false
}

// Attach sets up one use of a rule. The plugin's interest grows with what the rule reads.
func (plugin *Plugin) Attach(instance, rule string, parameters map[string]any) error {
	description, found := plugin.Rule(rule)
	if !found {
		return fmt.Errorf("plugin %q has no rule %q", plugin.Name, rule)
	}
	request := protocol.AttachRequest{Instance: instance, Rule: rule, Parameters: parameters}
	if err := plugin.connection.Send(protocol.Attach, request); err != nil {
		return err
	}
	if err := plugin.receive(protocol.Attached, &protocol.AttachReply{}); err != nil {
		return err
	}
	plugin.interest = plugin.interest.Join(description.Reads)
	return nil
}

// Start gives the plugin its part of the world and returns what its rules do when they start.
func (plugin *Plugin) Start(snapshot world.ChangeSet) (world.ChangeSet, error) {
	if err := plugin.connection.Send(protocol.Start, &protocol.StartRequest{Snapshot: snapshot}); err != nil {
		return world.ChangeSet{}, err
	}
	var reply world.ChangeSet
	return reply, plugin.receive(protocol.Changes, &reply)
}

// Step advances rule instances; the changes other plugins made since the last call go with it.
func (plugin *Plugin) Step(stage string, time, step float64, instances []string) (world.ChangeSet, error) {
	request := &protocol.StepRequest{Stage: stage, Time: time, Step: step, Instances: instances, Incoming: plugin.pending}
	plugin.pending = nil
	if err := plugin.connection.Send(protocol.Step, request); err != nil {
		return world.ChangeSet{}, err
	}
	var reply world.ChangeSet
	return reply, plugin.receive(protocol.Changes, &reply)
}

// Close asks the plugin to stop and waits a little for it; a program that does not exit is killed.
func (plugin *Plugin) Close() error {
	_ = plugin.connection.Send(protocol.Stop, struct{}{})
	select {
	case err := <-plugin.finished:
		return err
	case <-time.After(stopWaiting):
		if plugin.process != nil {
			_ = plugin.process.Process.Kill()
		}
		return fmt.Errorf("plugin %q did not stop in time", plugin.Name)
	}
}

func (plugin *Plugin) receive(kind protocol.Kind, body any) error {
	message, err := plugin.connection.Receive()
	if err != nil {
		return fmt.Errorf("plugin %q: %w", plugin.Name, err)
	}
	if err := message.Expect(kind); err != nil {
		return fmt.Errorf("plugin %q: %w", plugin.Name, err)
	}
	return message.Decode(body)
}

// prefixedWriter starts every line with a prefix, so plugin logs can be told apart.
type prefixedWriter struct {
	prefix      string
	target      io.Writer
	startOfLine bool
}

func (writer *prefixedWriter) Write(bytes []byte) (int, error) {
	for _, character := range bytes {
		if writer.startOfLine {
			if _, err := io.WriteString(writer.target, writer.prefix); err != nil {
				return 0, err
			}
		}
		if _, err := writer.target.Write([]byte{character}); err != nil {
			return 0, err
		}
		writer.startOfLine = character == '\n'
	}
	return len(bytes), nil
}
