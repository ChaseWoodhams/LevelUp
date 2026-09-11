package main

// rebuild_test.go — RE-ASSEMBLING FROM DISK, OFFLINE (#10).
//
// Every test here wires deps with a NIL Halo client. That is the assertion, not an
// omission: `rebuild` exists for the day the decoder improves, when every match worth
// rebuilding has a CDN link that died months ago. If any path reached for the network the
// nil client would panic, and the test would say so.

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain/title"
)

// offlineDeps wires the archiver with no client at all and a throwaway root.
func offlineDeps(t *testing.T, build buildFilm) deps {
	t.Helper()
	return deps{
		Client:  nil, // rebuild must never reach for it
		Paths:   title.NewPathResolver(t.TempDir()),
		Title:   title.DefaultSlug,
		Catalog: testCatalog(),
		Archive: testArchive(t),
		Build:   build,
	}
}

// captured seeds a match whose chunks are on disk and whose row says so.
func captured(t *testing.T, d deps, state filmState, skip reason) {
	t.Helper()
	if _, err := writeFilmChunks(d.Paths, testMatchID, testChunks()); err != nil {
		t.Fatalf("seeding the chunk cache: %v", err)
	}
	rec, roster := sampleRecord()
	rec.ArtifactPath, rec.BuiltAt, rec.DecoderRev = "", nil, ""
	rec.State, rec.SkipReason = state, skip
	rec.Tracks, rec.Points = 0, 0
	if err := d.Archive.recordMatch(context.Background(), rec, roster); err != nil {
		t.Fatalf("seeding the archive: %v", err)
	}
}

// THE TICKET'S OWN CRITERION: a match that failed to build, rebuilt by a decoder that now
// works, leaves the failed state and carries the new counts.
func TestRebuild_AFailedMatchLeavesTheFailedState(t *testing.T) {
	rebuilt := replay.ReplayDocument{
		SchemaVersion: replay.SchemaVersion,
		Tracks: []replay.Track{
			{Slot: 1, Team: -1, Points: []replay.Point{{T: 0}, {T: 1}, {T: 2}}},
			{Slot: 2, Team: -1, Points: []replay.Point{{T: 0}, {T: 1}}},
		},
		Coverage: &replay.Coverage{Bridge: replay.BridgeHealth{LivesNamed: 12, LivesTotal: 14}},
	}
	d := offlineDeps(t, stubBuild(&rebuilt))
	captured(t, d, stateFailed, skipNoTracks)

	out, err := rebuildOne(context.Background(), d, testMatchID)
	if err != nil {
		t.Fatalf("rebuildOne: %v", err)
	}
	if out.SkipReason != "" {
		t.Fatalf("skipped for %q, want a rebuild", out.SkipReason)
	}

	rec, found, err := d.Archive.recorded(context.Background(), testMatchID)
	if err != nil || !found {
		t.Fatalf("reading back: found=%v err=%v", found, err)
	}
	if rec.State != stateDownloaded || rec.SkipReason != "" {
		t.Errorf("state/reason = %q/%q, want %q and no reason - the match never left `failed`",
			rec.State, rec.SkipReason, stateDownloaded)
	}
	// The recorded counts are the NEW ones: that is how a coverage change is spotted.
	if rec.Tracks != 2 || rec.Points != 5 || rec.NamedLives != 12 || rec.TotalLives != 14 {
		t.Errorf("counts = %d tracks / %d points / %d of %d lives, want 2/5/12/14",
			rec.Tracks, rec.Points, rec.NamedLives, rec.TotalLives)
	}
	if rec.ArtifactPath == "" {
		t.Error("no artifact path recorded")
	}
	if _, statErr := os.Stat(d.Paths.ReplayArtifactPath(title.DefaultSlug, testMatchID)); statErr != nil {
		t.Errorf("the artifact was not written: %v", statErr)
	}
}

// A rebuild must not blank the facts that belong to the MATCH rather than to the build.
// `archive.recorded` is a partial reader, so round-tripping it through recordMatch would
// wipe mode, playlist, played-at and source_gamertag on every rebuild.
func TestRebuild_KeepsTheFactsThatBelongToTheMatch(t *testing.T) {
	d := offlineDeps(t, stubBuild(nil))
	captured(t, d, stateFailed, skipNoTracks)

	if _, err := rebuildOne(context.Background(), d, testMatchID); err != nil {
		t.Fatalf("rebuildOne: %v", err)
	}

	var mode, playlist, source string
	var players int
	ctx := context.Background()
	if err := d.Archive.db.QueryRow(ctx, `
        SELECT mode, playlist, source_gamertag FROM matches WHERE match_id = ?`, testMatchID).
		Scan(&mode, &playlist, &source); err != nil {
		t.Fatalf("reading the preserved facts: %v", err)
	}
	if mode != "Slayer" || playlist != "Ranked Arena" || source != "JGtm" {
		t.Errorf("mode/playlist/source = %q/%q/%q, want them untouched by a rebuild",
			mode, playlist, source)
	}
	if err := d.Archive.db.QueryRow(ctx,
		`SELECT count(*) FROM participants WHERE match_id = ?`, testMatchID).Scan(&players); err != nil {
		t.Fatalf("counting the roster: %v", err)
	}
	if players != 2 {
		t.Errorf("roster = %d rows, want 2 - a rebuild cannot have changed who played", players)
	}
}

// No chunks on disk: a clear refusal, never a silent re-download. This is the case that
// makes the "offline" promise real rather than incidental.
func TestRebuild_WithoutChunksFailsClearly(t *testing.T) {
	d := offlineDeps(t, stubBuild(nil))
	rec, roster := sampleRecord()
	rec.ArtifactPath, rec.BuiltAt, rec.DecoderRev = "", nil, ""
	rec.State, rec.SkipReason = stateFailed, skipNoTracks
	if err := d.Archive.recordMatch(context.Background(), rec, roster); err != nil {
		t.Fatalf("seeding the archive: %v", err)
	}

	_, err := rebuildOne(context.Background(), d, testMatchID)
	if err == nil {
		t.Fatal("a rebuild with no chunks on disk reported success")
	}
	if !strings.Contains(err.Error(), "never downloads") {
		t.Errorf("err = %v, want it to say the rebuild does not download", err)
	}
	// And the row is untouched: nothing happened, so nothing is recorded.
	if after, _, _ := d.Archive.recorded(context.Background(), testMatchID); after.State != stateFailed {
		t.Errorf("state = %q, want %q left alone", after.State, stateFailed)
	}
}

// A match nobody captured is not a rebuild candidate, and saying so is more useful than
// building an artifact for a match the archive has never heard of.
func TestRebuild_AnUnknownMatchIsRefused(t *testing.T) {
	d := offlineDeps(t, stubBuild(nil))
	_, err := rebuildOne(context.Background(), d, testMatchID)
	if err == nil || !strings.Contains(err.Error(), "not in the archive") {
		t.Errorf("err = %v, want a refusal naming the missing row", err)
	}
}

// A decoder that still refuses the film keeps the match in `failed` - with the reason
// updated to what happened THIS time, so a rebuild that changed the failure is visible.
func TestRebuild_AStillBrokenDecoderStaysFailed(t *testing.T) {
	boom := errors.New("decoder exploded")
	d := offlineDeps(t, func(string, string, string, replay.Options) (replay.ReplayDocument, error) {
		return replay.ReplayDocument{}, boom
	})
	captured(t, d, stateFailed, skipNoTracks)

	if _, err := rebuildOne(context.Background(), d, testMatchID); !errors.Is(err, boom) {
		t.Fatalf("err = %v, want the decoder's error", err)
	}
	rec, _, _ := d.Archive.recorded(context.Background(), testMatchID)
	if rec.State != stateFailed || rec.SkipReason != skipBuildFailed {
		t.Errorf("state/reason = %q/%q, want %q/%q",
			rec.State, rec.SkipReason, stateFailed, skipBuildFailed)
	}
}

// A rebuild that produces NO artifact must not clear a terminal verdict. The film is still
// gone from the CDN whatever the local cache holds, and writing `downloaded` would make the
// match retryable again — sending the hourly loop back to a link known to be dead, the
// exact waste #7 exists to prevent.
func TestRebuild_DoesNotClearAnExpiredVerdictWhenItBuildsNothing(t *testing.T) {
	d := offlineDeps(t, stubBuild(nil))
	captured(t, d, stateExpired, skipFilmAbsent)
	// The map's bounds have not arrived: the rebuild produces nothing, through the path
	// that is neither a decoder failure nor a disk failure.
	rec, roster := sampleRecord()
	rec.MapName, rec.MapModule = "Forbidden Sands", ""
	rec.ArtifactPath, rec.BuiltAt, rec.DecoderRev = "", nil, ""
	rec.State, rec.SkipReason = stateExpired, skipFilmAbsent
	if err := d.Archive.recordMatch(context.Background(), rec, roster); err != nil {
		t.Fatalf("seeding the archive: %v", err)
	}

	out, err := rebuildOne(context.Background(), d, testMatchID)
	if err != nil {
		t.Fatalf("rebuildOne: %v", err)
	}
	if out.SkipReason != skipUnsupportedMap {
		t.Fatalf("skip reason = %q, want %q", out.SkipReason, skipUnsupportedMap)
	}
	after, _, _ := d.Archive.recorded(context.Background(), testMatchID)
	if after.State != stateExpired || after.SkipReason != skipFilmAbsent {
		t.Errorf("state/reason = %q/%q, want %q/%q left standing - the terminal verdict was "+
			"cleared by a rebuild that built nothing",
			after.State, after.SkipReason, stateExpired, skipFilmAbsent)
	}
}

// An expired film whose chunks were captured before the link died is EXACTLY what this
// command is for: the archive says `expired`, and the bytes are still on disk. Here the
// rebuild SUCCEEDS, and an artifact is what settles the match - so the state moves.
func TestRebuild_RescuesAMatchWhoseFilmHasExpired(t *testing.T) {
	d := offlineDeps(t, stubBuild(nil))
	captured(t, d, stateExpired, skipFilmAbsent)

	out, err := rebuildOne(context.Background(), d, testMatchID)
	if err != nil {
		t.Fatalf("rebuildOne: %v", err)
	}
	if out.ArtifactPath == "" {
		t.Fatal("no artifact built from chunks that are on disk")
	}
	rec, _, _ := d.Archive.recorded(context.Background(), testMatchID)
	if rec.State != stateDownloaded {
		t.Errorf("state = %q, want %q - the film is on disk, whatever the CDN says",
			rec.State, stateDownloaded)
	}
}
