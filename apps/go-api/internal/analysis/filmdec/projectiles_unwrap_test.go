package filmdec

import (
	"math"
	"testing"
)

// streetsRange is the quantisation box of sgh_streets, where the wrap was measured.
var streetsRange = Vec3Range{
	{Min: -24.32224, Max: 27.407486},
	{Min: -23.018236, Max: 29.866623},
	{Min: -51.375183, Max: 16.546118},
}

func approx(a, b float32) bool { return math.Abs(float64(a-b)) < 1e-3 }

// The recorded defect: a flight crossing Y = -23.02 reappears at Y = +29.84 in one step.
// Unwrapping restores the path below the bound, and keeps following it.
func TestUnwrapLife_RestoresAFlightThatLeftTheBox(t *testing.T) {
	ext := streetsRange[1].Max - streetsRange[1].Min
	pts := []ProjectileSample{
		{TimestampUS: 0, X: 14.37, Y: -22.40, Z: 1},
		{TimestampUS: 16_000, X: 14.47, Y: -23.00, Z: 1},
		{TimestampUS: 32_000, X: 14.57, Y: -23.60 + ext, Z: 1}, // wrapped to the top
		{TimestampUS: 48_000, X: 14.67, Y: -24.20 + ext, Z: 1},
	}

	unwrapLife(pts, &streetsRange)

	want := []float32{-22.40, -23.00, -23.60, -24.20}
	for i, w := range want {
		if !approx(pts[i].Y, w) {
			t.Errorf("pt %d Y = %.3f, want %.3f", i, pts[i].Y, w)
		}
		if !approx(pts[i].X, 14.37+0.1*float32(i)) || pts[i].Z != 1 {
			t.Errorf("pt %d X/Z = %.3f/%.3f, the other axes must not move", i, pts[i].X, pts[i].Z)
		}
	}
}

// A flight that leaves and comes back crosses the bound twice: the offset returns to zero.
func TestUnwrapLife_FlightThatComesBackIsUnchangedAfterReturn(t *testing.T) {
	ext := streetsRange[0].Max - streetsRange[0].Min
	// Out past X max (wrapped to the low side), then back inside where the raw value is right.
	pts := []ProjectileSample{{X: 27.0}, {X: 27.8 - ext}, {X: 27.3}, {X: 26.5}}

	unwrapLife(pts, &streetsRange)

	want := []float32{27.0, 27.8, 27.3, 26.5}
	for i, w := range want {
		if !approx(pts[i].X, w) {
			t.Errorf("pt %d X = %.3f, want %.3f", i, pts[i].X, w)
		}
	}
}

// Ordinary motion, even fast, is never a wrap: nothing moves by half an extent in one sample.
func TestUnwrapLife_LeavesOrdinaryFlightsAlone(t *testing.T) {
	pts := []ProjectileSample{
		{X: -10, Y: 0, Z: -5}, {X: -4, Y: 5, Z: 0}, {X: 2, Y: 10, Z: 5}, {X: 8, Y: 15, Z: 10},
	}
	orig := append([]ProjectileSample(nil), pts...)

	unwrapLife(pts, &streetsRange)

	for i := range pts {
		if pts[i] != orig[i] {
			t.Errorf("pt %d = %+v, want unchanged %+v", i, pts[i], orig[i])
		}
	}
}
