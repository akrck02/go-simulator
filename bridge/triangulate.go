package bridge

// triangulate splits a simple polygon (no holes, no crossing edges, either winding) into
// triangles by clipping ears. It returns indices into the points, three per triangle, all wound
// counter-clockwise as seen from +y (the same way whatever the input winding).
func triangulate(points [][2]float32) []uint32 {
	count := len(points)
	if count < 3 {
		return nil
	}
	remaining := make([]int, count)
	for index := range remaining {
		remaining[index] = index
	}
	if signedArea(points) < 0 {
		reverse(remaining)
	}
	var triangles []uint32
	for attempts := 0; len(remaining) > 3 && attempts < 2*count*count; attempts++ {
		ear, found := findEar(points, remaining)
		if !found {
			break // degenerate polygon: what is left stays open rather than looping forever
		}
		previous, next := remaining[(ear+len(remaining)-1)%len(remaining)], remaining[(ear+1)%len(remaining)]
		triangles = append(triangles, uint32(previous), uint32(remaining[ear]), uint32(next))
		remaining = append(remaining[:ear], remaining[ear+1:]...)
	}
	if len(remaining) == 3 {
		triangles = append(triangles, uint32(remaining[0]), uint32(remaining[1]), uint32(remaining[2]))
	}
	return flipToFaceUp(triangles)
}

// flipToFaceUp: the ear clipping works counter-clockwise in (x, z); seen from +y with z pointing
// south that is clockwise, so every triangle is turned around.
func flipToFaceUp(triangles []uint32) []uint32 {
	for index := 0; index+2 < len(triangles); index += 3 {
		triangles[index+1], triangles[index+2] = triangles[index+2], triangles[index+1]
	}
	return triangles
}

func findEar(points [][2]float32, remaining []int) (int, bool) {
	for position := range remaining {
		previous := points[remaining[(position+len(remaining)-1)%len(remaining)]]
		current := points[remaining[position]]
		next := points[remaining[(position+1)%len(remaining)]]
		if cross(previous, current, next) <= 0 {
			continue // a reflex or flat corner is not an ear
		}
		if !anyPointInside(points, remaining, previous, current, next) {
			return position, true
		}
	}
	return 0, false
}

func anyPointInside(points [][2]float32, remaining []int, first, second, third [2]float32) bool {
	for _, index := range remaining {
		point := points[index]
		if point == first || point == second || point == third {
			continue
		}
		if cross(first, second, point) >= 0 && cross(second, third, point) >= 0 && cross(third, first, point) >= 0 {
			return true
		}
	}
	return false
}

// cross is positive when first → second → third turns counter-clockwise in (x, z).
func cross(first, second, third [2]float32) float32 {
	return (second[0]-first[0])*(third[1]-first[1]) - (second[1]-first[1])*(third[0]-first[0])
}

// signedArea is positive for counter-clockwise polygons in (x, z).
func signedArea(points [][2]float32) float32 {
	var area float32
	for index, point := range points {
		next := points[(index+1)%len(points)]
		area += point[0]*next[1] - next[0]*point[1]
	}
	return area / 2
}

func reverse(values []int) {
	for left, right := 0, len(values)-1; left < right; left, right = left+1, right-1 {
		values[left], values[right] = values[right], values[left]
	}
}
