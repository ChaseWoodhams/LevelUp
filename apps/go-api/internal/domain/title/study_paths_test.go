package title

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestStudyPaths — the study tool's own data root, resolved in one place so the
// archiver (writer) and the study server (reader) cannot disagree about where the
// archive lives. Everything sits under a single `data/study/` root: one directory to
// back up, one to delete, and nothing interleaved with the per-title warehouses.
func TestStudyPaths(t *testing.T) {
	p := NewPathResolver("/depot")
	cas := []struct {
		nom  string
		got  string
		want string
	}{
		{"root", p.StudyDataDir(), "data/study"},
		{"archive db", p.StudyArchiveDBPath(), "data/study/archive.duckdb"},
		{"rasters", p.StudyRastersDir(), "data/study/rasters"},
		{"map images", p.StudyMapImagesDir(), "data/study/map_images"},
	}
	for _, c := range cas {
		if !strings.HasSuffix(filepath.ToSlash(c.got), c.want) {
			t.Errorf("%s = %s, want suffix %s", c.nom, c.got, c.want)
		}
	}
}

// TestStudyPaths_UnderTheStudyRoot — every study path is a child of StudyDataDir.
// Pins the "one directory" property the methods exist to provide: a later path that
// escapes the root (a sibling `data/rasters/`, say) breaks backup and teardown
// without breaking anything at compile time.
func TestStudyPaths_UnderTheStudyRoot(t *testing.T) {
	p := NewPathResolver("/depot")
	root := filepath.ToSlash(p.StudyDataDir()) + "/"
	for nom, got := range map[string]string{
		"archive db": p.StudyArchiveDBPath(),
		"rasters":    p.StudyRastersDir(),
		"map images": p.StudyMapImagesDir(),
	} {
		if !strings.HasPrefix(filepath.ToSlash(got), root) {
			t.Errorf("%s (%s) is outside the study root %s", nom, got, root)
		}
	}
}
