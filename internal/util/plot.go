package util

import (
	"image/color"
	"math"
	"sort"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

// CreateHeatmap writes an N×N heatmap where each axis lists the same servers
// in the same order. The cell (i, j) shows traffic from servers[i] to servers[j].
// Any missing traffic entry is treated as 0 (white).
func CreateHeatmap(traffic map[string]map[string]int, filename string) error {
	// 1) Gather and sort a unified list of all servers (both sources and destinations).
	allServers := make(map[string]struct{})
	for src, dstMap := range traffic {
		allServers[src] = struct{}{}
		for dst := range dstMap {
			allServers[dst] = struct{}{}
		}
	}
	var servers []string
	for s := range allServers {
		servers = append(servers, s)
	}
	sort.Strings(servers)

	// 2) Build a Grid that returns 0 for missing keys.
	mg := &matrixGrid{
		nodes: servers,
		data:  traffic,
	}
	// 3) Create the plot.
	p := plot.New()
	p.Title.Text = "Traffic Heatmap"
	p.X.Label.Text = "Destination"
	p.Y.Label.Text = "Source"

	// 4) Create a custom palette from white (min) to dark blue (max).
	// Use 256 color steps for a smooth gradient.
	cp := customPalette{}
	heat := plotter.NewHeatMap(mg, cp)
	// Explicitly set the min and max values.
	heat.Min = 0
	heat.Max = 10000000

	p.Add(heat)

	// 5) Label both X and Y axes with the same ordered server list.
	p.NominalX(servers...)
	p.NominalY(servers...)

	// 6) Save the plot to a file (PNG, PDF, etc.).
	return p.Save(6*vg.Inch, 6*vg.Inch, filename+".png")
}

// customPalette implements the plot.Palette interface.
// It produces a gradient from white (low values) to dark blue (high values).
type customPalette struct{}

// Colors returns a slice of colors forming a gradient from white to dark blue.
func (cp customPalette) Colors() []color.Color {
	return createWhiteToDarkBluePalette(256)
}

// createWhiteToDarkBluePalette returns a palette with a linear gradient
// starting from white (RGB 255,255,255) at t=0 to dark blue (RGB 0,0,128) at t=1.
func createWhiteToDarkBluePalette(steps int) []color.Color {
	palette := make([]color.Color, steps)
	for i := 0; i < steps; i++ {
		// Logarithmic interpolation parameter:
		t := math.Log(1+float64(i)) / math.Log(float64(steps))
		// Interpolate red and green channels from 255 to 0.
		r := uint8(255 * (1 - t))
		g := uint8(255 * (1 - t))
		// Interpolate blue from 255 to 128.
		b := uint8(255*(1-t) + 128*t)
		palette[i] = color.RGBA{R: r, G: g, B: b, A: 255}
	}
	return palette
}

// matrixGrid implements the Gonum/plot GridXYZ interface.
// It uses a unified slice for both rows (sources) and columns (destinations).
type matrixGrid struct {
	nodes []string                  // The unified sorted list of servers.
	data  map[string]map[string]int // Outer key = source, inner key = destination.
}

// Dims returns the width and height of the grid: N×N if there are N servers.
func (mg *matrixGrid) Dims() (int, int) {
	n := len(mg.nodes)
	return n, n
}

// Z returns the traffic from mg.nodes[r] to mg.nodes[c], or 0 if missing.
func (mg *matrixGrid) Z(c, r int) float64 {
	src := mg.nodes[r]
	dst := mg.nodes[c]
	val, ok := mg.data[src][dst]
	if !ok {
		return 0 // Missing data defaults to zero.
	}
	return float64(val)
}

// X returns the X coordinate for column c.
func (mg *matrixGrid) X(c int) float64 { return float64(c) }

// Y returns the Y coordinate for row r.
func (mg *matrixGrid) Y(r int) float64 { return float64(r) }
