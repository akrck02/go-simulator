package simulation

import (
	"math"
	"testing"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/scene"
)

func TestPausedSimulationDoesNothing(t *testing.T) {
	calls := 0
	running := New(SystemFunc(func(*scene.Scene, float64) { calls++ }))
	running.Paused = true
	running.Advance(scene.New(), 0.1)
	if calls != 0 || running.Time != 0 {
		t.Fatalf("a paused simulation advanced")
	}
}

func TestSpeedScalesTheStep(t *testing.T) {
	var received float64
	running := New(SystemFunc(func(_ *scene.Scene, step float64) { received = step }))
	running.Speed = 3
	running.Advance(scene.New(), 0.5)
	if received != 1.5 || running.Time != 1.5 {
		t.Fatalf("step %v, time %v", received, running.Time)
	}
}

func TestPathFollowerLoopsThroughItsPoints(t *testing.T) {
	node := &scene.Node{}
	points := []graphics.Vec4{{0, 0, 0, 0}, {10, 0, 0, 0}}
	follower := NewPathFollower(node, points, 1, false)
	follower.Step(nil, 5)
	if position := node.Transform.Apply(graphics.Vec4{0, 0, 0, 1}); math.Abs(float64(position[0]-5)) > 1e-4 {
		t.Fatalf("after 5 s the node should be halfway, got %v", position)
	}
	follower.Step(nil, 10)
	if position := node.Transform.Apply(graphics.Vec4{0, 0, 0, 1}); math.Abs(float64(position[0]-5)) > 1e-4 {
		t.Fatalf("going back, after 15 s the node should be halfway again, got %v", position)
	}
}

func TestDayCycleTurnsTheSun(t *testing.T) {
	world := scene.New()
	before := world.Environment.SunDirection
	cycle := &DayCycle{SecondsPerDay: 100, Enabled: true}
	cycle.Step(world, 50)
	after := world.Environment.SunDirection
	if before.Dot(after) > 0.99 {
		t.Fatalf("half a day should move the sun")
	}
}

func TestCloudsDriftWithTheWind(t *testing.T) {
	world := scene.New()
	world.Environment.Wind = scene.Wind{Direction: graphics.Vec4{1, 0, 0, 0}, Strength: 1}
	(&CloudDrift{Speed: 1}).Step(world, 2)
	if world.Environment.Sky.Clouds.Offset[0] <= 0 || world.Environment.Sky.Clouds.Offset[2] != 0 {
		t.Fatalf("clouds should drift east, offset %v", world.Environment.Sky.Clouds.Offset)
	}
}

func TestFromTagsBuildsTheRequestedMotions(t *testing.T) {
	world := scene.New()
	world.Nodes = []*scene.Node{
		{Tags: map[string]any{"bob": map[string]any{"height": 0.5}}},
		{Tags: map[string]any{"spin": map[string]any{"speed": 1.0}, "path": map[string]any{"points": []any{[]any{0.0, 0.0, 0.0}, []any{1.0, 0.0, 0.0}}}}},
		{},
	}
	if systems := FromTags(world); len(systems) != 3 {
		t.Fatalf("expected 3 systems, got %d", len(systems))
	}
}
