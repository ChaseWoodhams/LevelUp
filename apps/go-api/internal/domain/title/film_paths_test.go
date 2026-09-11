package title

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestFilmChunksDir_SamePathForBothIDForms — the invariant that makes the chunk
// directory reachable whichever form of the match ID a caller holds.
//
// This is the same defect ReplayArtifactPath already guards against, one directory
// over: the film cache writes under the SHORT form, and a caller holding the full
// match ID used to build a path that was never written.
func TestFilmChunksDir_SamePathForBothIDForms(t *testing.T) {
	p := NewPathResolver("/depot")
	short := p.FilmChunksDir("000d5950")
	full := p.FilmChunksDir("000d5950-1234-4abc-9def-0123456789ab")
	if short != full {
		t.Errorf("the two match_id forms give two paths:\n  short %s\n  full  %s", short, full)
	}
	if !strings.HasSuffix(filepath.ToSlash(short), "data/cache/film_chunks/000d5950") {
		t.Errorf("unexpected path: %s", short)
	}
}

// TestFilmChunkPath_NumberedChunkFile — the on-disk layout is
// film_chunks/<short>/chunk_NN.bin, the naming the offline tools already read.
func TestFilmChunkPath_NumberedChunkFile(t *testing.T) {
	p := NewPathResolver("/depot")
	got := filepath.ToSlash(p.FilmChunkPath("000d5950-1234-4abc-9def-0123456789ab", 7))
	if !strings.HasSuffix(got, "data/cache/film_chunks/000d5950/chunk_07.bin") {
		t.Errorf("unexpected path: %s", got)
	}
}

// TestUneSeuleJointureFilmChunks — GUARD-RAIL (repo rule 6).
//
// The film-chunk join reached three copies: cmd/replay-build (which hand-built it
// from the FULL match ID, so it pointed at a directory nothing ever wrote) and two
// in cmd/diag_weapons_v3. Centralising without a guard lets the fourth copy back in
// silently — and this particular duplication is invisible at compile time, because a
// wrong path is a runtime "file not found", not a type error.
//
// ALLOWLIST (2026-09-03, explicit and dated per repo rule 3). Every entry below
// resolves chunks under an ARBITRARY cache root — a `--cache-dir` flag, or the legacy
// Python cache directory — not under repoRoot. They are therefore not PathResolver
// callers: pointing them at PathResolver would not rename a path, it would change which
// directory they read. Migrating them is a separate decision with its own risk, recorded
// as a finding rather than smuggled into this prefactor.
//
// The allowlist is per-DIRECTORY, so a brand-new file inside one of these tools is
// covered, but any new tool or internal package is not — which is the copy this guard
// exists to stop. If one of these ever becomes repoRoot-rooted, move it to
// FilmChunkPath and take it off this list.
func TestUneSeuleJointureFilmChunks(t *testing.T) {
	const owner = "film_paths.go"
	allowed := map[string]bool{
		"diag_weapons_v3":    true, // --cache-dir, may point at the legacy Python cache
		"haloclient":         true, // LocalFilmCache.rootDir: the legacy Python cache root
		"fetch_film_chunks":  true, // --cache-dir
		"frontb_coverage":    true, // --cache-dir
		"killsource":         true, // --cache-dir
		"probe_pi_reconcile": true, // --cache-dir
	}

	motif := regexp.MustCompile(`"film_chunks"`)
	seenAtOwner := false
	var offenders []string

	for _, root := range []string{
		filepath.Join("..", "..", "..", "internal"),
		filepath.Join("..", "..", "..", "cmd"),
	} {
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") ||
				strings.HasSuffix(path, "_test.go") {
				return nil
			}
			raw, rErr := os.ReadFile(filepath.Clean(path))
			if rErr != nil || !motif.Match(raw) {
				return nil
			}
			if filepath.Base(path) == owner {
				seenAtOwner = true
				return nil
			}
			if allowed[filepath.Base(filepath.Dir(path))] {
				return nil
			}
			offenders = append(offenders, path)
			return nil
		})
		if err != nil {
			t.Fatalf("walking %s: %v", root, err)
		}
	}

	// The guard must be able to FAIL: if the literal is gone from its owner too, the
	// test guards nothing any more (lesson J4.0, cf. TestUneSeuleTroncatureDeMatchID).
	if !seenAtOwner {
		t.Fatalf("literal not found in %s: the guard-rail no longer checks anything "+
			"(was the method renamed or moved?)", owner)
	}
	if len(offenders) > 0 {
		t.Errorf("film_chunks join copied outside %s: %v — call PathResolver.FilmChunksDir "+
			"/ FilmChunkPath", owner, offenders)
	}
}
