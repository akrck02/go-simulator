package world

// The vocabulary: the names plugins share so they can read each other's results without knowing
// each other. A plugin that writes the elevation calls its field FieldElevation; a plugin that
// reads it asks for that name in its interest. Names outside the vocabulary are allowed, but a
// name another plugin may want to read belongs here.
//
// Units: lengths, positions and heights are in world units, the unit of the scene (kilometres in
// the scenes of go-renderer's examples), on the ground plane (x, z) with y up. Every other quantity
// has the unit written next to its name.

// Fields: grids over the ground.
const (
	FieldElevation     = "elevation"     // height of the ground, world units
	FieldWaterDepth    = "water_depth"   // depth of water over the ground, world units (0 = dry)
	FieldSlope         = "slope"         // rise over run of the ground, no unit
	FieldTemperature   = "temperature"   // mean yearly temperature, °C
	FieldPrecipitation = "precipitation" // yearly rain and snow, mm
	FieldBiome         = "biome"         // index into the biome names a plugin publishes in an event
)

// Kinds: what an entity is.
const (
	KindStreet   = "street"   // a line with a width; property PropertyClass says which street
	KindParcel   = "parcel"   // a plot of land: an outline
	KindBuilding = "building" // a volume: an outline with a height
	KindWall     = "wall"     // a town wall: a line with a width and a height
	KindGate     = "gate"     // an opening in a wall
	KindSquare   = "square"   // an open public space: an outline
	KindBridge   = "bridge"   // a line over water
	KindRiver    = "river"    // water along a line or in an outline
	KindTree     = "tree"     // one tree; its species in PropertySpecies
	KindAnchor   = "anchor"   // a place a rules file fixes (see PropertyAnchor)
)

// Properties: free data on entities that more than one plugin reads.
const (
	PropertyYear    = "year"    // when the entity appeared, in the stage's clock
	PropertyUse     = "use"     // what it is used for: "housing", "shop", "workshop"…
	PropertyFloors  = "floors"  // number of storeys
	PropertyClass   = "class"   // a finer kind: "main", "road", "quay"…
	PropertySector  = "sector"  // the part of a place it belongs to
	PropertySpecies = "species" // what kind of plant or animal
	PropertyAnchor  = "anchor"  // true on entities a rules file fixed: plugins build around them, never over them
	PropertyRole    = "role"    // what an anchor means to the plugins: "obstacle", "road", "gate"…
)
