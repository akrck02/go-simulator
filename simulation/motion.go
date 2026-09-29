package simulation

import (
	"math"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/scene"
)

// animatedNode remembers the transform a node had before any motion was applied.
type animatedNode struct {
	node *scene.Node
	rest graphics.Mat4
}

func newAnimatedNode(node *scene.Node) animatedNode {
	rest := node.Transform
	if rest == (graphics.Mat4{}) {
		rest = graphics.Identity()
	}
	return animatedNode{node: node, rest: rest}
}

// Bobbing floats a node up and down with a slight roll, like a boat on the water.
type Bobbing struct {
	Height  float32 // amplitude of the vertical motion (world units)
	Period  float64 // seconds per oscillation (0 = 4)
	Roll    float64 // amplitude of the roll in radians
	target  animatedNode
	phase   float64
	elapsed float64
}

// NewBobbing makes a node bob; the phase depends on its position so neighbours do not move together.
func NewBobbing(node *scene.Node, height float32, period, roll float64) *Bobbing {
	target := newAnimatedNode(node)
	return &Bobbing{Height: height, Period: period, Roll: roll, target: target, phase: positionPhase(target.rest)}
}

// Step places the node at its rest transform plus the current bob and roll.
func (bobbing *Bobbing) Step(world *scene.Scene, step float64) {
	bobbing.elapsed += step
	period := bobbing.Period
	if period <= 0 {
		period = 4
	}
	wave := 2*math.Pi*bobbing.elapsed/period + bobbing.phase
	lift := graphics.Translate(graphics.Vec4{0, bobbing.Height * float32(math.Sin(wave)), 0, 0})
	roll := graphics.RotateX(bobbing.Roll * math.Sin(wave*0.7+1))
	bobbing.target.node.Transform = lift.Multiply(bobbing.target.rest).Multiply(roll)
}

// Spin turns a node around an axis of its own, like a mill's sails.
type Spin struct {
	Axis   graphics.Vec4
	Speed  float64 // radians per second
	target animatedNode
	angle  float64
}

// NewSpin makes a node spin around a local axis.
func NewSpin(node *scene.Node, axis graphics.Vec4, speed float64) *Spin {
	return &Spin{Axis: axis.Normalize(), Speed: speed, target: newAnimatedNode(node)}
}

// Step advances the rotation.
func (spin *Spin) Step(world *scene.Scene, step float64) {
	spin.angle += spin.Speed * step
	spin.target.node.Transform = spin.target.rest.Multiply(graphics.Rotate(spin.Axis, spin.angle))
}

// PathFollower moves a node along waypoints at a constant speed, facing where it goes (the model's
// +Z axis points along the path).
type PathFollower struct {
	Points   []graphics.Vec4 // world positions of the waypoints
	Speed    float32         // world units per second
	Loop     bool            // go back to the first point after the last; otherwise turn around
	target   animatedNode
	distance float32 // travelled along the path
	lengths  []float32
}

// NewPathFollower moves a node along the points; its rest transform keeps its scale and model orientation.
func NewPathFollower(node *scene.Node, points []graphics.Vec4, speed float32, loop bool) *PathFollower {
	follower := &PathFollower{Points: points, Speed: speed, Loop: loop, target: newAnimatedNode(node)}
	follower.lengths = segmentLengths(follower.route())
	return follower
}

// route is the list of points travelled in one cycle.
func (follower *PathFollower) route() []graphics.Vec4 {
	if follower.Loop && len(follower.Points) > 1 {
		return append(append([]graphics.Vec4{}, follower.Points...), follower.Points[0])
	}
	route := append([]graphics.Vec4{}, follower.Points...)
	for index := len(follower.Points) - 2; index >= 0; index-- {
		route = append(route, follower.Points[index])
	}
	return route
}

func segmentLengths(route []graphics.Vec4) []float32 {
	lengths := make([]float32, 0, len(route))
	for index := 1; index < len(route); index++ {
		lengths = append(lengths, distanceBetween(route[index-1], route[index]))
	}
	return lengths
}

// Step moves the node forward along the route.
func (follower *PathFollower) Step(world *scene.Scene, step float64) {
	total := sum(follower.lengths)
	if total == 0 {
		return
	}
	follower.distance = float32(math.Mod(float64(follower.distance+follower.Speed*float32(step)), float64(total)))
	position, heading := follower.pointAt(follower.distance)
	placement := graphics.Translate(position).Multiply(graphics.RotateY(heading))
	follower.target.node.Transform = placement.Multiply(withoutTranslation(follower.target.rest))
}

// pointAt returns the position at a distance along the route and the heading (radians around Y).
func (follower *PathFollower) pointAt(distance float32) (graphics.Vec4, float64) {
	route := follower.route()
	for index, length := range follower.lengths {
		if distance <= length || index == len(follower.lengths)-1 {
			from, to := route[index], route[index+1]
			var amount float32
			if length > 0 {
				amount = distance / length
			}
			position := graphics.Vec4{from[0] + (to[0]-from[0])*amount, from[1] + (to[1]-from[1])*amount, from[2] + (to[2]-from[2])*amount, 0}
			return position, math.Atan2(float64(to[0]-from[0]), float64(to[2]-from[2]))
		}
		distance -= length
	}
	return route[0], 0
}

func withoutTranslation(transform graphics.Mat4) graphics.Mat4 {
	transform[12], transform[13], transform[14] = 0, 0, 0
	return transform
}

// positionPhase derives a phase from a transform's position.
func positionPhase(transform graphics.Mat4) float64 {
	return math.Mod(float64(transform[12])*12.9898+float64(transform[14])*78.233, 2*math.Pi)
}

func distanceBetween(first, second graphics.Vec4) float32 {
	deltaX, deltaY, deltaZ := second[0]-first[0], second[1]-first[1], second[2]-first[2]
	return float32(math.Sqrt(float64(deltaX*deltaX + deltaY*deltaY + deltaZ*deltaZ)))
}

func sum(values []float32) float32 {
	var total float32
	for _, value := range values {
		total += value
	}
	return total
}
