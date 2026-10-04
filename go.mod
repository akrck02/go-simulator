module github.com/akrck02/go-simulator

go 1.25.6

require (
	github.com/akrck02/go-renderer v0.0.0
	github.com/akrck02/go-scene-builder v0.0.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/go-gl/gl v0.0.0-20260331235117-4566fea9a276 // indirect
	github.com/go-gl/glfw/v3.4/glfw v0.1.0-pre.1 // indirect
	github.com/twpayne/go-geos v0.23.0 // indirect
)

// the renderer and the scene builder are developed next to this project
replace (
	github.com/akrck02/go-renderer => ../go-renderer
	github.com/akrck02/go-scene-builder => ../go-scene-builder
)
