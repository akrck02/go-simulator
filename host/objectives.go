package host

import (
	"fmt"
	"math"
	"strings"

	"github.com/akrck02/go-simulator/conditions"
	"github.com/akrck02/go-simulator/configuration"
	"github.com/akrck02/go-simulator/world"
)

// ObjectiveResult is what an objective found when it was checked.
type ObjectiveResult struct {
	Stage     string                  `json:"stage"`
	Time      float64                 `json:"time"` // stage time when it was checked
	Objective configuration.Objective `json:"-"`
	Name      string                  `json:"name"`
	Value     float64                 `json:"value"`
	Found     bool                    `json:"found"` // false when the event or its value never came
	Reached   bool                    `json:"reached"`
	Expected  string                  `json:"expected"`
	Source    string                  `json:"source,omitempty"`
}

// DueObjectives returns the objectives of a stage whose time is after "before" and not after the
// stage's current time.
func (stage *Stage) DueObjectives(before float64) []configuration.Objective {
	var due []configuration.Objective
	for _, objective := range stage.Objectives {
		if objective.At > before && objective.At <= stage.Time {
			due = append(due, objective)
		}
	}
	return due
}

// CheckObjective measures an objective on the world as it is now.
func (engine *Engine) CheckObjective(stage *Stage, objective configuration.Objective) ObjectiveResult {
	result := ObjectiveResult{Stage: stage.Name, Time: stage.Time, Objective: objective, Name: objective.Name,
		Expected: expectation(objective), Source: objective.Source}
	result.Value, result.Found = engine.measure(objective)
	result.Reached = result.Found && reaches(objective, result.Value)
	return result
}

// Describe returns one line for people: whether it was reached, the value and what was expected.
func (result ObjectiveResult) Describe() string {
	mark := "reached"
	if !result.Reached {
		mark = "NOT REACHED"
	}
	value := "nothing to measure"
	if result.Found {
		value = fmt.Sprintf("%g", result.Value)
	}
	return fmt.Sprintf("%s · %s at %g: %s (expected %s)", mark, result.Name, result.Time, value, result.Expected)
}

func (engine *Engine) measure(objective configuration.Objective) (float64, bool) {
	if objective.Counts() {
		return float64(engine.count(objective.Count)), true
	}
	event, found := engine.LastEvent(objective.Event)
	if !found {
		return 0, false
	}
	return conditions.Number(event.Data[objective.Value])
}

func (engine *Engine) count(when map[string]any) int {
	total := 0
	for _, entity := range engine.World.Entities() {
		if conditions.Holds(when, conditions.OfEntity(entity)) {
			total++
		}
	}
	return total
}

// reaches tells whether a value meets the objective; a count with no bounds asks for at least one.
func reaches(objective configuration.Objective, value float64) bool {
	if objective.Counts() && objective.Minimum == nil && objective.Maximum == nil && objective.Near == nil {
		return value >= 1
	}
	if objective.Minimum != nil && value < *objective.Minimum {
		return false
	}
	if objective.Maximum != nil && value > *objective.Maximum {
		return false
	}
	if objective.Near != nil {
		return math.Abs(value-*objective.Near) <= objective.Tolerance*math.Abs(*objective.Near)
	}
	return true
}

func expectation(objective configuration.Objective) string {
	var parts []string
	if objective.Minimum != nil {
		parts = append(parts, fmt.Sprintf("at least %g", *objective.Minimum))
	}
	if objective.Maximum != nil {
		parts = append(parts, fmt.Sprintf("at most %g", *objective.Maximum))
	}
	if objective.Near != nil {
		parts = append(parts, fmt.Sprintf("%g ± %g %%", *objective.Near, objective.Tolerance*100))
	}
	if len(parts) == 0 {
		return "at least 1"
	}
	return strings.Join(parts, ", ")
}

// LastEvent returns the last event of a name the plugins emitted, if any.
func (engine *Engine) LastEvent(name string) (world.Event, bool) {
	event, found := engine.lastEvents[name]
	return event, found
}

func (engine *Engine) rememberEvent(change world.Change) {
	if change.Operation != world.Emit || change.Event == nil {
		return
	}
	if engine.lastEvents == nil {
		engine.lastEvents = map[string]world.Event{}
	}
	engine.lastEvents[change.Event.Name] = *change.Event
}
