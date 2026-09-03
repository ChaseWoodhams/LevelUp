package main

// build.go — ASSEMBLING THE REPLAY ARTIFACT, ONE AT A TIME.
//
// THE SERIALISATION QUESTION, ANSWERED ON THE SOURCE (2026-09-03, ticket #5).
// killsource.Decode documents a package-level mutex because the bit decoder's
// replication parameters are package GLOBALS: two concurrent decodes read each other's
// tuning and the result depends on call order. The question this ticket had to settle
// was whether internal/analysis/filmdec — the decoder replay.BuildFromFilm drives — does
// the same for itself.
//
// IT DOES NOT. filmdec keeps the same kind of mutable package-level state
// (PositionFullPrecision, PositionDeltaHasHandleTail, PositionCalibratedSkip,
// DeltaQuantum, DeltaAxisWidth, MobilityActionBodyPorted, bipedDefaultState* and the
// frame-chain statistics) and imports "sync" NOWHERE: neither filmdec nor
// internal/analysis/replay contains a mutex. Nothing serialises those globals today —
// every existing caller happens to be a single-threaded offline tool
// (cmd/replay-build builds exactly one match per process).
//
// So the archiver owns the serialisation, and it is a PROCESS-WIDE lock rather than a
// per-Archiver field on purpose: the state being protected is package-level in filmdec,
// so two archiver instances in one process would collide exactly as two goroutines do.

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"levelup/go-api/internal/analysis/replay"
)

// buildFilm is the archiver's build seam: the signature of replay.BuildFromFilm.
//
// A named type rather than a direct call so the orchestration can be exercised without
// a 20 MB film on disk — the decoder has its own suites (golden assembly, mini-reel) and
// this binary must not re-test them.
type buildFilm func(matchID, titleSlug, filmDir string, opt replay.Options) (replay.ReplayDocument, error)

// buildMu serialises every replay build in this process. See the file header for why it
// is package-level and why it is the archiver's job at all.
var buildMu sync.Mutex

// runBuild runs one build under the process-wide lock.
//
// EVERY build goes through here, which is the point: the lock belongs to the CALL SITE,
// not to a wiring. Wrapping the build function at construction time would have made
// serialisation a property of `newDeps` alone — `watch` (#8), or any second wiring, could
// then drop it by simply assembling `deps` itself, and no test would fail.
func runBuild(build buildFilm, matchID, titleSlug, filmDir string, opt replay.Options) (replay.ReplayDocument, error) {
	buildMu.Lock()
	defer buildMu.Unlock()
	return build(matchID, titleSlug, filmDir, opt)
}

// buildOptions assembles the inputs of one match's build: the map's dequantisation
// bounds (mandatory), the title's label catalogue, and the optional map backgrounds.
func (d deps) buildOptions(ctx context.Context, m matchMap) replay.Options {
	return replay.Options{
		FrameIntervalMS: d.FrameIntervalMS,
		Geometry:        loadGeometry(ctx, d.Paths.MapGeometryDir(d.Title)),
		Structure:       loadStructure(ctx, d.Paths.MapStructurePath(d.Title, m.Module), m.Module),
		Labels:          d.Labels,
		WorldRange:      &m.Range,
	}
}

// loadStructure loads the map's frozen structural background (floors, platforms, ramps,
// walls). Absence is NOT fatal — not every map has a structure file yet, and a replay
// without a background is still readable — but it is logged.
func loadStructure(ctx context.Context, path, module string) []replay.Surface {
	ms, err := replay.LoadMapStructure(path)
	if err != nil {
		slog.WarnContext(ctx, "study-archiver: map structure unavailable - replay without structural background",
			"err", err, "path", path, "module", module)
		return nil
	}
	slog.InfoContext(ctx, "study-archiver: map structure loaded",
		"module", module, "surfaces", len(ms.Surfaces),
		"coverage_pct", ms.Stats.CoveragePct, "path", path)
	return ms.Surfaces
}

// loadGeometry loads the map's Forge props (contextual landmarks). Absence is NOT fatal
// either; it is logged.
func loadGeometry(ctx context.Context, dir string) []replay.MapObject {
	objs, skipped, err := replay.LoadGeometry(dir)
	if err != nil {
		slog.WarnContext(ctx, "study-archiver: map geometry unavailable - replay without props",
			"err", err, "dir", dir)
		return nil
	}
	slog.InfoContext(ctx, "study-archiver: map geometry loaded",
		"objects", len(objs), "without_footprint", skipped, "dir", dir)
	return objs
}

// Permissions of everything this tool writes under data/cache. They match what
// cmd/replay-build and cmd/fetch_film_chunks already write into the SAME directories: a
// tool that tightened them on its own would leave one cache tree with two answers, and
// the artifact has to stay readable by the study server that serves it (Spec 2).
const (
	cacheDirPerm  = 0o755
	cacheFilePerm = 0o644
)

// writeArtifact serialises the document, creating parent directories, and returns the
// size in bytes.
func writeArtifact(outPath string, doc replay.ReplayDocument) (int, error) {
	if err := os.MkdirAll(filepath.Dir(outPath), cacheDirPerm); err != nil {
		return 0, fmt.Errorf("artifact directory: %w", err)
	}
	blob, err := json.Marshal(doc)
	if err != nil {
		return 0, fmt.Errorf("artifact encoding: %w", err)
	}
	if err := os.WriteFile(outPath, blob, cacheFilePerm); err != nil {
		return 0, fmt.Errorf("artifact write: %w", err)
	}
	return len(blob), nil
}

// totalPoints counts the published trajectory samples across every track.
func totalPoints(doc replay.ReplayDocument) int {
	n := 0
	for _, tr := range doc.Tracks {
		n += len(tr.Points)
	}
	return n
}
