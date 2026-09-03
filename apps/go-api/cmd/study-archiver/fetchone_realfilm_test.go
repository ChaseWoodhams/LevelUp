package main

// fetchone_realfilm_test.go — THE SAME PATH, WITH A REAL FILM AND THE REAL DECODER.
//
// fetchone_test.go stubs the build so the orchestration can be pinned without a 20 MB
// film. This one stubs nothing: the repo's existing fake Halo server serves the JGtm
// full-match fixture, the real haloclient downloads it, and replay.BuildFromFilm decodes
// it into an artifact on disk.
//
// IT SKIPS WHEN THE FIXTURE IS ABSENT, which is the normal case: jgtm_full_match is
// gitignored (6 MB of binaries) and regenerated with
// `go run ./cmd/gen_test_fixtures download-full-match` (tokens required). That is the
// convention every other fixture-backed suite in this repo already follows.

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/replaylabels"
	"levelup/go-api/internal/sync/haloclient"
	"levelup/go-api/internal/sync/halotest"
	"levelup/go-api/internal/testfixtures"
)

// jgtmMatchID is the fixture's match (Arena, 8 players, ~9 min).
const jgtmMatchID = "b71d39db-e3af-40e4-b7f9-e7c34c367981"

// realRepoDeps wires the archiver with the REPO's own reference catalogues and the REAL
// decoder, but a throwaway output root: the test must never write into data/cache. That
// split is exactly why deps carries catalogues loaded rather than as paths.
func realRepoDeps(t *testing.T, serverURL string) deps {
	t.Helper()
	repoRoot := testfixtures.RepoRoot()
	repoPaths := title.NewPathResolver(repoRoot)
	catalog, err := filmdec.LoadMapQuantCatalog(repoPaths.MapQuantBoundsPath(title.DefaultSlug))
	if err != nil {
		t.Fatalf("map bounds catalogue: %v", err)
	}
	labels, err := replaylabels.Load(repoRoot, title.DefaultSlug)
	if err != nil {
		t.Fatalf("label catalogue: %v", err)
	}
	return deps{
		Client: haloclient.NewHaloAPIClient("spartan-test", "clearance-test", 1000).
			WithHTTPClient(&http.Client{Transport: redirectTo(serverURL)}),
		Paths:          title.NewPathResolver(t.TempDir()),
		Title:          title.DefaultSlug,
		Catalog:        catalog,
		Labels:         labels,
		Archive:        testArchive(t),
		Build:          replay.BuildFromFilm,
		SourceGamertag: "JGtm",
	}
}

func TestFetchOne_RealFilmEndToEnd(t *testing.T) {
	if !testfixtures.JGtmFullMatchAvailable() {
		t.Skip("jgtm_full_match fixture absent - regenerate via " +
			"`go run ./cmd/gen_test_fixtures download-full-match`")
	}
	fx := testfixtures.LoadJGtmFullMatch(t)
	if len(fx.MatchStatsRaw) == 0 {
		t.Skip("api_match_stats.json absent from the fixture - no map name to resolve")
	}
	srv := halotest.NewFakeServer(t, fx)
	d := realRepoDeps(t, srv.URL)

	// The fixture's own map decides whether this match is buildable at all; a fixture on
	// a map the catalogue does not carry would exercise the skip path, not this one.
	var stats map[string]any
	if err := json.Unmarshal(fx.MatchStatsRaw, &stats); err != nil {
		t.Fatalf("fixture stats: %v", err)
	}
	facts, err := readMatchFacts(stats, "fixture")
	if err != nil {
		t.Fatalf("fixture stats unreadable: %v", err)
	}
	if _, err := resolveMatchMap(facts.MapName, d.Catalog); err != nil {
		t.Skipf("fixture map not in the versioned catalogue: %v", err)
	}

	out, err := fetchOne(context.Background(), d, jgtmMatchID)
	if err != nil {
		t.Fatalf("fetchOne: %v", err)
	}
	if out.SkipReason != "" {
		t.Fatalf("skipped for %q, want an artifact", out.SkipReason)
	}
	if out.ChunksWritten != len(fx.Manifest.CustomData.Chunks) {
		t.Errorf("wrote %d chunks, want the manifest's %d",
			out.ChunksWritten, len(fx.Manifest.CustomData.Chunks))
	}
	// The header (type 1) and the highlight footer (type 3) are the two the older
	// callers dropped; the death feed in the footer is what names the lives.
	if idx := fx.HighlightChunkIndex(); idx >= 0 {
		if _, err := os.Stat(d.Paths.FilmChunkPath(jgtmMatchID, idx)); err != nil {
			t.Errorf("highlight chunk %d not written: %v", idx, err)
		}
	}
	if _, err := os.Stat(d.Paths.FilmChunkPath(jgtmMatchID, 0)); err != nil {
		t.Errorf("header chunk not written: %v", err)
	}

	blob, err := os.ReadFile(out.ArtifactPath) //nolint:gosec // path from the resolver, under t.TempDir()
	if err != nil {
		t.Fatalf("artifact not written: %v", err)
	}
	var doc replay.ReplayDocument
	if err := json.Unmarshal(blob, &doc); err != nil {
		t.Fatalf("artifact is not valid JSON: %v", err)
	}
	if len(doc.Tracks) == 0 || out.Tracks != len(doc.Tracks) {
		t.Errorf("artifact carries %d tracks, outcome reports %d", len(doc.Tracks), out.Tracks)
	}
	if out.Points == 0 {
		t.Error("no trajectory sample decoded from a real film")
	}
	t.Logf("archived %s on %s: %d tracks, %d points, %d shots, %d bytes",
		jgtmMatchID, out.MapName, out.Tracks, out.Points, out.Shots, len(blob))
}
