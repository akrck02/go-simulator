package bridge

import (
	"cmp"
	"slices"
	"strconv"

	"github.com/akrck02/go-renderer/graphics"
	"github.com/akrck02/go-renderer/scene"
	"github.com/akrck02/go-simulator/world"
)

// The overlay: entities with the shared properties label, icon, badge and label_color get a
// label above them in the viewer; those with pinned go to the header at the top-left corner
// instead; emit events with a notice show it for a few seconds. See world.PropertyLabel.

// overlayEntry is what the bridge knows to draw an entity's label.
type overlayEntry struct {
	label  *scene.Label // nil while the entity has nothing to show in the world
	anchor labelAnchor
}

// labelAnchor is where an entity's label stands: on its node, lifted by the height of its shape,
// or, for outlines and lines (drawn in ground coordinates, often in a shared batch), over their
// middle on the ground.
type labelAnchor struct {
	node   *scene.Node
	point  graphics.Vec4
	height float32
	ground bool // anchored at point, not at the node
}

type pinnedLine struct {
	identifier world.Identifier
	order      float64
	line       scene.HeaderLine
}

// placeLabelAnchor works out where an entity's label goes from its shape. Call it when the
// entity is added or its shape changes.
func (bridge *Bridge) placeLabelAnchor(identifier world.Identifier, node *scene.Node, shape *world.Shape) {
	entry := bridge.overlayEntry(identifier)
	entry.anchor = labelAnchor{node: node, height: shapeTop(shape, node)}
	if shape != nil && shape.OnTheGround() {
		entry.anchor = bridge.groundAnchor(shape)
	}
	if entry.label != nil {
		applyAnchor(entry.label, entry.anchor, node)
	}
}

func (bridge *Bridge) overlayEntry(identifier world.Identifier) *overlayEntry {
	entry, found := bridge.overlay[identifier]
	if !found {
		entry = &overlayEntry{}
		bridge.overlay[identifier] = entry
	}
	return entry
}

// shapeTop is the height of a shape above its origin: the top of a box, the highest vertex of a
// mesh, or 0 for an entity without a shape.
func shapeTop(shape *world.Shape, node *scene.Node) float32 {
	if shape != nil && shape.Primitive != "" {
		key := keyOf(shape)
		return key.offset[1] + key.size[1]/2
	}
	if node.Mesh != nil {
		return node.Mesh.Top()
	}
	return 0
}

// groundAnchor puts a label over the middle of an outline (on top of its height: a building's
// roof) or over the middle point of a line, just over the ground for a flat shape.
func (bridge *Bridge) groundAnchor(shape *world.Shape) labelAnchor {
	x, z := labelSpotOf(shape)
	return labelAnchor{point: graphics.Vec4{x, bridge.groundHeight(x, z) + liftOf(shape), z, 1}, height: shape.Height, ground: true}
}

// labelSpotOf returns the average of an outline's corners, or the middle point of a line (a point
// on the line, where an average could fall beside a bent one).
func labelSpotOf(shape *world.Shape) (x, z float32) {
	if len(shape.Outline) == 0 {
		middle := shape.Line[len(shape.Line)/2]
		return middle[0], middle[1]
	}
	for _, point := range shape.Outline {
		x, z = x+point[0], z+point[1]
	}
	count := float32(len(shape.Outline))
	return x / count, z / count
}

func applyAnchor(label *scene.Label, anchor labelAnchor, node *scene.Node) {
	label.Height = anchor.height
	if anchor.ground {
		label.Node, label.Point, label.Hidden = nil, anchor.point, node.Hidden
		return
	}
	label.Node, label.Hidden = anchor.node, false
}

// showProperties updates an entity's label or header line from its properties (all of them,
// after merging). Call it whenever the properties change.
func (bridge *Bridge) showProperties(identifier world.Identifier, node *scene.Node, properties map[string]any) {
	entry := bridge.overlayEntry(identifier)
	label := labelFromProperties(properties)
	if order, pinned := numberOf(properties[world.PropertyPinned]); pinned && !label.IsEmpty() {
		bridge.hideLabel(entry)
		bridge.pin(pinnedLine{identifier: identifier, order: order, line: headerLineOf(label)})
		return
	}
	bridge.unpin(identifier)
	if label.IsEmpty() {
		bridge.hideLabel(entry)
		return
	}
	if entry.label == nil {
		entry.label = &scene.Label{}
		bridge.scene.Overlay.AddLabel(entry.label)
	}
	entry.label.Text, entry.label.Icon, entry.label.Badge, entry.label.Tint = label.Text, label.Icon, label.Badge, label.Tint
	applyAnchor(entry.label, entry.anchor, node)
}

func labelFromProperties(properties map[string]any) scene.Label {
	label := scene.Label{
		Text:  textOf(properties[world.PropertyLabel]),
		Icon:  textOf(properties[world.PropertyIcon]),
		Badge: textOf(properties[world.PropertyBadge]),
	}
	label.Tint, _ = scene.ColorFromHex(textOf(properties[world.PropertyLabelColor]))
	return label
}

// headerLineOf turns a label into a header line; a badge follows the text after a dot.
func headerLineOf(label scene.Label) scene.HeaderLine {
	text := label.Text
	if label.Badge != "" && text != "" {
		text += " · " + label.Badge
	} else if label.Badge != "" {
		text = label.Badge
	}
	return scene.HeaderLine{Icon: label.Icon, Text: text, Tint: label.Tint}
}

func (bridge *Bridge) hideLabel(entry *overlayEntry) {
	if entry.label == nil {
		return
	}
	bridge.scene.Overlay.RemoveLabel(entry.label)
	entry.label = nil
}

func (bridge *Bridge) pin(line pinnedLine) {
	bridge.pinned[line.identifier] = line
	bridge.rebuildHeader()
}

func (bridge *Bridge) unpin(identifier world.Identifier) {
	if _, pinned := bridge.pinned[identifier]; !pinned {
		return
	}
	delete(bridge.pinned, identifier)
	bridge.rebuildHeader()
}

// rebuildHeader lists the pinned entities by their number (ties by identifier, so the order never
// depends on map iteration).
func (bridge *Bridge) rebuildHeader() {
	lines := make([]pinnedLine, 0, len(bridge.pinned))
	for _, line := range bridge.pinned {
		lines = append(lines, line)
	}
	slices.SortFunc(lines, func(first, second pinnedLine) int {
		if order := cmp.Compare(first.order, second.order); order != 0 {
			return order
		}
		return cmp.Compare(first.identifier, second.identifier)
	})
	header := make([]scene.HeaderLine, 0, len(lines))
	for _, line := range lines {
		header = append(header, line.line)
	}
	bridge.scene.Overlay.Header = header
}

// forgetOverlay takes away the label or header line of a removed entity.
func (bridge *Bridge) forgetOverlay(identifier world.Identifier) {
	entry, found := bridge.overlay[identifier]
	if found {
		bridge.hideLabel(entry)
	}
	delete(bridge.overlay, identifier)
	bridge.unpin(identifier)
}

// showEvent turns an event carrying a notice into a notice on the overlay; the viewer's clock
// times it.
func (bridge *Bridge) showEvent(event *world.Event) {
	if event == nil {
		return
	}
	text := textOf(event.Data[world.EventDataNotice])
	if text == "" {
		return
	}
	tint, _ := scene.ColorFromHex(textOf(event.Data[world.PropertyLabelColor]))
	bridge.scene.Overlay.AddNotice(textOf(event.Data[world.EventDataIcon]), text, tint)
}

// textOf writes a property as text: strings as they are, numbers without trailing zeros.
func textOf(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(typed), 'f', -1, 32)
	case int:
		return strconv.Itoa(typed)
	case bool:
		return strconv.FormatBool(typed)
	}
	return ""
}

func numberOf(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	}
	return 0, false
}
