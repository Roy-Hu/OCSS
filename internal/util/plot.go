package util

import (
	"sort"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/palette/moreland"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

// createHeatmap converts the TrafficMatrix into a gonum/plot heatmap and saves to a file.
func CreateHeatmap(tm map[string]map[string]int, filename string) error {
	// Collect and sort all source & destination labels
	var sources, destinations []string
	for src := range tm {
		sources = append(sources, src)
	}
	sort.Strings(sources)

	// Collect destinations from the first source’s map (or union them all if needed).
	for dst := range tm[sources[0]] {
		destinations = append(destinations, dst)
	}
	sort.Strings(destinations)

	// Prepare the data in a GridXYZ form
	gridData := &matrixGrid{
		sources:      sources,
		destinations: destinations,
		data:         tm,
	}

	// Make a new plot
	p := plot.New()
	p.Title.Text = "Traffic Heatmap"
	p.X.Label.Text = "Destination Server"
	p.Y.Label.Text = "Source Server"

	// Reverse the Y axis visually if desired; otherwise, leave it as linear:
	p.Y.Scale = plot.LinearScale{}

	// Create a color palette (e.g. Moreland’s “Kindlmann”)
	palette := moreland.Kindlmann()
	palette.SetMax(12) // Adjust if needed to cap the color scale

	// Create a heatmap using our GridXYZ data
	heatmap := plotter.NewHeatMap(gridData, palette.Palette(12))
	p.Add(heatmap)

	// Set tick labels for X and Y axes based on our labels
	p.NominalX(destinations...)
	p.NominalY(sources...)

	// Save the heatmap to file
	return p.Save(4*vg.Inch, 4*vg.Inch, filename)
}

// matrixGrid implements the GridXYZ interface that gonum/plot uses for heatmaps.
type matrixGrid struct {
	sources, destinations []string
	data                  map[string]map[string]int
}

// Dims returns the dimensions of the grid: (columns, rows).
func (mg *matrixGrid) Dims() (c, r int) {
	return len(mg.destinations), len(mg.sources)
}

// Z returns the value at the given column/row.
func (mg *matrixGrid) Z(c, r int) float64 {
	src := mg.sources[r]
	dst := mg.destinations[c]
	return float64(mg.data[src][dst])
}

// X returns the coordinate along the X-axis for column c.
func (mg *matrixGrid) X(c int) float64 {
	return float64(c)
}

// Y returns the coordinate along the Y-axis for row r.
func (mg *matrixGrid) Y(r int) float64 {
	return float64(r)
}
