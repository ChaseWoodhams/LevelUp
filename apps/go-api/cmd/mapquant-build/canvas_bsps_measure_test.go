package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/himap"
)

// canvas_bsps_measure_test.go — EVERY sbsp TAG OF A MODULE, NOT JUST THE LARGEST.
//
// mapquant-build keeps the largest sbsp tag as the map's box. For Forge canvases that is not always
// right: fo11_blank and fo13_frost both return a 3.9 km box at 18 bits, and films decoded with it
// spread their players over ~250-390 m, about 8x an arena. This bench lists each tag's bounds and
// widths so another tag of the same module can be examined.
//
//	MAPQUANT_LEVELS="C:/…/deploy/ds/levels/multi" MAPQUANT_MODULES=fo11_blank,fo08_wetland \
//	  go test ./cmd/mapquant-build/ -run TestListModuleBSPs -v
func TestListModuleBSPs(t *testing.T) {
	levels, modules := os.Getenv("MAPQUANT_LEVELS"), os.Getenv("MAPQUANT_MODULES")
	if levels == "" || modules == "" {
		t.Skip("MAPQUANT_LEVELS / MAPQUANT_MODULES not set: game files needed, not CI")
	}
	for _, mod := range strings.Split(modules, ",") {
		files, _ := filepath.Glob(filepath.Join(levels, mod, "*.module"))
		if len(files) == 0 {
			t.Errorf("%s: no module", mod)
			continue
		}
		bsps, err := himap.ReadModuleBSPBounds(files[0])
		if err != nil {
			t.Errorf("%s: %v", mod, err)
			continue
		}
		for i, b := range bsps {
			w := b.Bounds.AxisWidths()
			t.Logf("%s sbsp #%d (file %d, %d bytes): x %.1f..%.1f y %.1f..%.1f z %.1f..%.1f extent %.0f/%.0f/%.0f W %d/%d/%d",
				mod, i, b.FileIndex, b.UncompSize, b.Bounds.Min[0], b.Bounds.Max[0], b.Bounds.Min[1], b.Bounds.Max[1],
				b.Bounds.Min[2], b.Bounds.Max[2], b.Bounds.Extent(0), b.Bounds.Extent(1), b.Bounds.Extent(2), w[0], w[1], w[2])
		}
	}
}
