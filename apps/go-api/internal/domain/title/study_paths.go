package title

import "path/filepath"

// study_paths.go — THE STUDY TOOL'S OWN DATA ROOT.
//
// The study tool (film archiver, replay viewer, heat maps) keeps data that is neither
// a per-title warehouse nor a regenerable cache: an archive of matches deliberately
// captured before their film CDN links expire. It gets its own root so that one
// directory can be backed up or deleted as a unit, without picking through
// `data/titles/**`.
//
// Single writer by construction: the archiver is a scheduled batch job and is the only
// process that writes the archive DB; readers open it read-only (the same discipline
// the rest of the repo applies to DuckDB files).

// StudyDataDir returns the root of the study tool's data.
// Ex: data/study/
func (p *PathResolver) StudyDataDir() string {
	return filepath.Join(p.repoRoot, "data", "study")
}

// StudyArchiveDBPath returns the study archive database (match/participant records
// of everything the archiver has captured).
// Ex: data/study/archive.duckdb
//
// DuckDB, not SQLite — the repo-wide rule holds here too.
func (p *PathResolver) StudyArchiveDBPath() string {
	return filepath.Join(p.StudyDataDir(), "archive.duckdb")
}

// StudyRastersDir returns the directory holding cached occupancy/heat rasters derived
// from archived matches. Regenerable from the archive, so safe to delete.
// Ex: data/study/rasters/
func (p *PathResolver) StudyRastersDir() string {
	return filepath.Join(p.StudyDataDir(), "rasters")
}

// StudyMapImagesDir returns the directory holding map floor images the study viewer
// draws underneath replays and heat maps.
// Ex: data/study/map_images/
func (p *PathResolver) StudyMapImagesDir() string {
	return filepath.Join(p.StudyDataDir(), "map_images")
}
