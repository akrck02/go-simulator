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

// Event data: any emit event can carry a short message for whoever watches the run. The viewer
// shows it for a few seconds under its header, newest first.
const (
	EventDataNotice = "notice" // text of the message
	EventDataIcon   = "icon"   // optional: name of a Lucide icon (https://lucide.dev) shown before it
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

// Overlay properties: what the viewer writes over an entity. All are optional, and a null value
// takes them away. They are for whoever watches a run; models should not read them.
const (
	PropertyLabel      = "label"       // a short text shown above the entity
	PropertyIcon       = "icon"        // name of a Lucide icon (https://lucide.dev) shown before the label, or alone
	PropertyBadge      = "badge"       // a very short text in a small pill under the label: "3/10"
	PropertyLabelColor = "label_color" // "#rrggbb": colour of the icon and the badge (the text stays dark)
	PropertyPinned     = "pinned"      // a number: show the label as that line of the header, top-left, instead
)
