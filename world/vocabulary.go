package world

// The vocabulary: the names plugins share so they can read each other's results without knowing
// each other. A plugin that writes the elevation calls its field FieldElevation; a plugin that
// reads it asks for that name in its interest. Names outside the vocabulary are allowed, but a
// name another plugin may want to read belongs here.
//
// Units: positions and the sizes of shapes are in world units, the unit of the scene (kilometres
// in the example scenes), on the ground plane (x, z) with y up: they are what gets
// drawn. Fields and properties are data, in physical units written next to their names (heights
// in metres, temperatures in °C), so models read them without knowing the scene's scale.

// Fields: grids over the ground.
const (
	FieldElevation     = "elevation"     // height of the ground, metres
	FieldWaterDepth    = "water_depth"   // depth of water over the ground, metres (0 = dry)
	FieldSlope         = "slope"         // rise over run of the ground, no unit
	FieldTemperature   = "temperature"   // mean yearly temperature, °C
	FieldPrecipitation = "precipitation" // yearly rain and snow, mm
	FieldBiome         = "biome"         // index into the names of the EventBiomes event, −1 for the sea
	FieldVegetation    = "vegetation"    // index into the names of the EventVegetation event, −1 for bare ground
	FieldTreeCover     = "tree_cover"    // share of the ground under trees, 0 to 1
	FieldShrubCover    = "shrub_cover"   // share of the ground under shrubs, 0 to 1
	FieldHerbCover     = "herb_cover"    // share of the ground under grasses and flowers, 0 to 1
)

// Events: what plugins announce. Events that name the categories of a field carry {"names": […]}.
const (
	EventBiomes     = "biomes"     // names of the values of FieldBiome
	EventVegetation = "vegetation" // names of the values of FieldVegetation, with each species' "labels" and "forms"
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
	PropertyRole    = "role"    // what an anchor means to the plugins: "obstacle", "road", "gate", "plant", "region"…
)
