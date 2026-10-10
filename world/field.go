package world

import "fmt"

// Field is a named grid of values over an area of the ground: elevation, water depth,
// temperature, tide amplitude… Values go row by row from MinimumZ, each row from MinimumX, and sit
// at cell centres.
type Field struct {
	Name    string    `json:"name"`
	Columns int       `json:"columns"`
	Rows    int       `json:"rows"`
	Extent  Extent    `json:"extent"`
	Values  []float32 `json:"-"` // travels as a binary attachment, not as JSON
}

// Extent is the area a field covers, in world units on the ground plane (x, z). Its edges are
// float64 so a plugin that reads a field gets back exactly the cells another one wrote: a cell of
// 0.0125 km stays 0.0125, which simulations sensitive to their inputs need.
type Extent struct {
	MinimumX float64 `json:"minimum_x"`
	MaximumX float64 `json:"maximum_x"`
	MinimumZ float64 `json:"minimum_z"`
	MaximumZ float64 `json:"maximum_z"`
}

// Region is a rectangle of cells inside a field.
type Region struct {
	Column  int `json:"column"`
	Row     int `json:"row"`
	Columns int `json:"columns"`
	Rows    int `json:"rows"`
}

// FieldPatch creates a field or sets the values of a region of it. Columns, Rows and Extent are
// only needed to create the field; without a Region the values cover the whole field.
type FieldPatch struct {
	Name    string    `json:"name"`
	Columns int       `json:"columns,omitempty"`
	Rows    int       `json:"rows,omitempty"`
	Extent  *Extent   `json:"extent,omitempty"`
	Region  *Region   `json:"region,omitempty"`
	Values  []float32 `json:"-"`
}

// At returns the value of a cell.
func (field *Field) At(column, row int) float32 { return field.Values[row*field.Columns+column] }

// Copy returns a deep copy of the field.
func (field *Field) Copy() *Field {
	copied := *field
	copied.Values = append([]float32(nil), field.Values...)
	return &copied
}

func newField(patch *FieldPatch) (*Field, error) {
	if patch.Columns <= 0 || patch.Rows <= 0 || patch.Extent == nil {
		return nil, fmt.Errorf("field %q does not exist: creating it needs columns, rows and extent", patch.Name)
	}
	field := &Field{Name: patch.Name, Columns: patch.Columns, Rows: patch.Rows, Extent: *patch.Extent,
		Values: make([]float32, patch.Columns*patch.Rows)}
	return field, nil
}

// apply copies the patch values into the field.
func (field *Field) apply(patch *FieldPatch) error {
	region := Region{Columns: field.Columns, Rows: field.Rows}
	if patch.Region != nil {
		region = *patch.Region
	}
	if err := field.checkRegion(region, len(patch.Values)); err != nil {
		return err
	}
	for row := range region.Rows {
		target := (region.Row+row)*field.Columns + region.Column
		copy(field.Values[target:target+region.Columns], patch.Values[row*region.Columns:(row+1)*region.Columns])
	}
	return nil
}

func (field *Field) checkRegion(region Region, valueCount int) error {
	if region.Column < 0 || region.Row < 0 || region.Column+region.Columns > field.Columns || region.Row+region.Rows > field.Rows {
		return fmt.Errorf("field %q: region %+v is outside %d×%d cells", field.Name, region, field.Columns, field.Rows)
	}
	if valueCount != region.Columns*region.Rows {
		return fmt.Errorf("field %q: %d values for a region of %d×%d cells", field.Name, valueCount, region.Columns, region.Rows)
	}
	return nil
}
