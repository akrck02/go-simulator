package snapshot

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/akrck02/go-simulator/world"
)

// WriteFields saves every field of the world as a grid other tools can read: a header
// "<prefix><name>.json" and the values "<prefix><name>.f32" (little-endian float32, row by row
// from the smallest z). The header carries the size and the extent in world coordinates, and the
// same layout as a relief in east-south axes: west = minimum x, north = −minimum z.
func WriteFields(folder, prefix string, current *world.World, extra map[string]any) error {
	for _, field := range current.Fields() {
		header := map[string]any{"name": field.Name, "columns": field.Columns, "rows": field.Rows, "extent": field.Extent,
			"cell": float64(field.Extent.MaximumX-field.Extent.MinimumX) / float64(field.Columns),
			"west": field.Extent.MinimumX, "north": -field.Extent.MinimumZ, "units": "km", "axes": "east-south"}
		for key, value := range extra {
			header[key] = value
		}
		encoded, err := json.Marshal(header)
		if err != nil {
			return err
		}
		base := filepath.Join(folder, fmt.Sprintf("%s%s", prefix, field.Name))
		if err := os.WriteFile(base+".json", encoded, 0o644); err != nil {
			return err
		}
		values := make([]byte, 4*len(field.Values))
		for index, value := range field.Values {
			binary.LittleEndian.PutUint32(values[4*index:], math.Float32bits(value))
		}
		if err := os.WriteFile(base+".f32", values, 0o644); err != nil {
			return err
		}
	}
	return nil
}
