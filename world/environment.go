package world

import "github.com/akrck02/go-simulator/spatial"

// Environment is what surrounds everything: the sun, the sky, the wind and the water. As a state
// every set field is known; as a patch only the set fields change. A nil field in the state means
// the world does not have it (no tides, for example).
type Environment struct {
	SunDirection  *spatial.Vector `json:"sun_direction,omitempty"`  // towards the sun, length 1
	CelestialPole *spatial.Vector `json:"celestial_pole,omitempty"` // axis the sky turns around
	Wind          *Wind           `json:"wind,omitempty"`
	CloudOffset   *spatial.Vector `json:"cloud_offset,omitempty"` // how far the clouds have drifted (x, z)
	TideAngle     *float32        `json:"tide_angle,omitempty"`   // radians; the tide heights come with the scene
}

// Wind blows towards a horizontal direction with a strength from 0 (calm) to 1 (strong).
type Wind struct {
	Direction spatial.Vector `json:"direction"`
	Strength  float32        `json:"strength"`
}

// IsEmpty tells whether the patch sets nothing.
func (environment *Environment) IsEmpty() bool {
	return environment.SunDirection == nil && environment.CelestialPole == nil && environment.Wind == nil &&
		environment.CloudOffset == nil && environment.TideAngle == nil
}

// Copy returns a deep copy.
func (environment Environment) Copy() Environment {
	return Environment{
		SunDirection:  copyPointer(environment.SunDirection),
		CelestialPole: copyPointer(environment.CelestialPole),
		Wind:          copyPointer(environment.Wind),
		CloudOffset:   copyPointer(environment.CloudOffset),
		TideAngle:     copyPointer(environment.TideAngle),
	}
}

// merge applies a patch: every set field replaces the current one.
func (environment *Environment) merge(patch *Environment) {
	if patch.SunDirection != nil {
		environment.SunDirection = copyPointer(patch.SunDirection)
	}
	if patch.CelestialPole != nil {
		environment.CelestialPole = copyPointer(patch.CelestialPole)
	}
	if patch.Wind != nil {
		environment.Wind = copyPointer(patch.Wind)
	}
	if patch.CloudOffset != nil {
		environment.CloudOffset = copyPointer(patch.CloudOffset)
	}
	if patch.TideAngle != nil {
		environment.TideAngle = copyPointer(patch.TideAngle)
	}
}

func copyPointer[Value any](pointer *Value) *Value {
	if pointer == nil {
		return nil
	}
	copied := *pointer
	return &copied
}
