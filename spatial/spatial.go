// Package spatial is the small 3D math plugins need to describe where things are and how they
// turn, without depending on any renderer. Matrices are column-major 4×4 arrays with the
// translation in elements 12, 13 and 14, the same layout go-renderer uses, so the bridge copies
// them as they are.
package spatial

import "math"

// Vector is a position or a direction (x, y, z).
type Vector [3]float32

// Matrix is a column-major 4×4 transform.
type Matrix [16]float32

// Identity returns the transform that changes nothing.
func Identity() Matrix {
	return Matrix{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}
}

// Translation returns a transform that moves by an offset.
func Translation(offset Vector) Matrix {
	matrix := Identity()
	matrix[12], matrix[13], matrix[14] = offset[0], offset[1], offset[2]
	return matrix
}

// RotationX returns a rotation around the X axis (radians).
func RotationX(angle float64) Matrix {
	cosine, sine := float32(math.Cos(angle)), float32(math.Sin(angle))
	matrix := Identity()
	matrix[5], matrix[6], matrix[9], matrix[10] = cosine, sine, -sine, cosine
	return matrix
}

// RotationY returns a rotation around the Y axis (radians).
func RotationY(angle float64) Matrix {
	cosine, sine := float32(math.Cos(angle)), float32(math.Sin(angle))
	matrix := Identity()
	matrix[0], matrix[2], matrix[8], matrix[10] = cosine, -sine, sine, cosine
	return matrix
}

// RotationAround returns a rotation around any axis (radians).
func RotationAround(axis Vector, angle float64) Matrix {
	axis = axis.Normalize()
	x, y, z := axis[0], axis[1], axis[2]
	sine, cosine := float32(math.Sin(angle)), float32(math.Cos(angle))
	opposite := 1 - cosine
	matrix := Identity()
	matrix[0], matrix[1], matrix[2] = x*x*opposite+cosine, y*x*opposite+z*sine, z*x*opposite-y*sine
	matrix[4], matrix[5], matrix[6] = x*y*opposite-z*sine, y*y*opposite+cosine, z*y*opposite+x*sine
	matrix[8], matrix[9], matrix[10] = x*z*opposite+y*sine, y*z*opposite-x*sine, z*z*opposite+cosine
	return matrix
}

// Multiply returns first × second: second is applied first.
func (first Matrix) Multiply(second Matrix) Matrix {
	var result Matrix
	for column := range 4 {
		for row := range 4 {
			result[column*4+row] = first[row]*second[column*4] + first[4+row]*second[column*4+1] +
				first[8+row]*second[column*4+2] + first[12+row]*second[column*4+3]
		}
	}
	return result
}

// ApplyToPoint transforms a position (translation included).
func (matrix Matrix) ApplyToPoint(point Vector) Vector {
	return Vector{
		matrix[0]*point[0] + matrix[4]*point[1] + matrix[8]*point[2] + matrix[12],
		matrix[1]*point[0] + matrix[5]*point[1] + matrix[9]*point[2] + matrix[13],
		matrix[2]*point[0] + matrix[6]*point[1] + matrix[10]*point[2] + matrix[14],
	}
}

// ApplyToDirection transforms a direction (translation ignored).
func (matrix Matrix) ApplyToDirection(direction Vector) Vector {
	return Vector{
		matrix[0]*direction[0] + matrix[4]*direction[1] + matrix[8]*direction[2],
		matrix[1]*direction[0] + matrix[5]*direction[1] + matrix[9]*direction[2],
		matrix[2]*direction[0] + matrix[6]*direction[1] + matrix[10]*direction[2],
	}
}

// Position returns the translation of a transform.
func (matrix Matrix) Position() Vector { return Vector{matrix[12], matrix[13], matrix[14]} }

// WithoutTranslation returns the transform with its translation removed.
func (matrix Matrix) WithoutTranslation() Matrix {
	matrix[12], matrix[13], matrix[14] = 0, 0, 0
	return matrix
}

// IsZero tells whether every element is zero (an unset transform).
func (matrix Matrix) IsZero() bool { return matrix == Matrix{} }

// Length returns the length of a vector.
func (vector Vector) Length() float32 {
	return float32(math.Sqrt(float64(vector.Dot(vector))))
}

// Normalize returns the vector with length 1 (or the zero vector).
func (vector Vector) Normalize() Vector {
	length := vector.Length()
	if length == 0 {
		return Vector{}
	}
	return Vector{vector[0] / length, vector[1] / length, vector[2] / length}
}

// Dot returns the dot product.
func (vector Vector) Dot(other Vector) float32 {
	return vector[0]*other[0] + vector[1]*other[1] + vector[2]*other[2]
}

// Add returns the sum of two vectors.
func (vector Vector) Add(other Vector) Vector {
	return Vector{vector[0] + other[0], vector[1] + other[1], vector[2] + other[2]}
}

// Subtract returns vector − other.
func (vector Vector) Subtract(other Vector) Vector {
	return Vector{vector[0] - other[0], vector[1] - other[1], vector[2] - other[2]}
}

// Scale returns the vector multiplied by a factor.
func (vector Vector) Scale(factor float32) Vector {
	return Vector{vector[0] * factor, vector[1] * factor, vector[2] * factor}
}
