package main

// fetchone_expiry_test.go — THE TWO WAYS ARCHIVING FAILS, AND THE THIRD THAT IS NOT A
// FAILURE AT ALL (ticket #7).
//
// `expired` and `failed` mean OPPOSITE things to a later run, and the point of these
// tests is that the archiver cannot quietly collapse them:
//
//   - expired: the CDN blobs are gone for good. Retrying costs a request an hour against
//     a link that will never come back, so no later run may touch the match again.
//   - failed: the film arrived and the decoder refused it. The chunks are on disk, so a
//     decoder fix rescues the match — every later run must remain free to try.
//   - transient (5xx, timeouts): neither. A run that ends this way must leave NO terminal
//     row behind, or one bad afternoon would permanently bury a perfectly good film.
//
// The distinction is asserted on API CALL COUNTS, not just on rows: "never re-attempted"
// is a claim about not fetching, and a row assertion alone would pass on a tool that
// re-downloaded the whole film every hour and then rewrote the same verdict.

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain/title"
)

// runFetch runs one pass and returns the outcome, the archive row it left (if any) and
// the error. Unlike `archived`, it tolerates a failure — half these tests are about what
// a FAILING pass leaves behind.
func runFetch(t *testing.T, d deps) (outcome, matchRecord, bool, error) {
	t.Helper()
	out, err := fetchOne(context.Background(), d, testMatchID)
	rec, found, readErr := d.Archive.recorded(context.Background(), testMatchID)
	if readErr != nil {
		t.Fatalf("reading the archive row: %v", readErr)
	}
	return out, rec, found, err
}

// countingBuild records how many builds ran and delegates to `build`.
func countingBuild(runs *int, build buildFilm) buildFilm {
	return func(matchID, titleSlug, filmDir string, opt replay.Options) (replay.ReplayDocument, error) {
		*runs++
		return build(matchID, titleSlug, filmDir, opt)
	}
}

// THE TICKET'S OWN FIXTURE: the manifest still resolves, the chunk route answers 404. The
// film is gone even though Halo still describes it — which is precisely the case the
// manifest-level 410 test cannot reach, and the one that would otherwise be retried
// forever as a transport error.
func TestFetchOne_ExpiredChunksAreRecordedAndNeverRetried(t *testing.T) {
	srv := newFakeHalo(t, map[int][]byte{0: []byte("header"), 1: []byte("replication")},
		statsWithMap("Cliffhanger"))
	srv.chunkStatus = http.StatusNotFound

	builds := 0
	d := srv.deps(t, countingBuild(&builds, stubBuild(nil)))

	first, rec, found, err := runFetch(t, d)
	if err != nil {
		t.Fatalf("an expired film is a recorded outcome, not an error: %v", err)
	}
	if !found {
		t.Fatal("an expired film left no row: the next run would fetch it again")
	}
	if rec.State != stateExpired || rec.SkipReason != skipFilmAbsent {
		t.Errorf("state/reason = %q/%q, want %q/%q",
			rec.State, rec.SkipReason, stateExpired, skipFilmAbsent)
	}
	if first.ChunksWritten != 0 || builds != 0 {
		t.Errorf("wrote %d chunks and ran %d builds on a film whose blobs are gone", first.ChunksWritten, builds)
	}

	blobs, manifests, stats := srv.blobCalls.Load(), srv.manifestCalls.Load(), srv.statsCalls.Load()

	second, rec2, _, err := runFetch(t, d)
	if err != nil {
		t.Fatalf("second pass: %v", err)
	}
	if !second.Settled {
		t.Error("the second pass did not recognise the match as permanently settled")
	}
	if second.SkipReason != skipFilmAbsent {
		t.Errorf("second pass skip reason = %q, want %q", second.SkipReason, skipFilmAbsent)
	}
	if got := srv.blobCalls.Load(); got != blobs {
		t.Errorf("blobs fetched %d times, want %d - the expired film was re-downloaded", got, blobs)
	}
	if got := srv.manifestCalls.Load(); got != manifests {
		t.Errorf("manifest fetched %d times, want %d - the expired film was re-attempted", got, manifests)
	}
	if got := srv.statsCalls.Load(); got != stats {
		t.Errorf("stats fetched %d times, want %d - the expired match was re-read", got, stats)
	}
	if rec2.State != stateExpired {
		t.Errorf("state after the second pass = %q, want %q", rec2.State, stateExpired)
	}

	var rows int
	if err := d.Archive.db.QueryRow(context.Background(), `SELECT count(*) FROM matches`).Scan(&rows); err != nil {
		t.Fatalf("counting matches: %v", err)
	}
	if rows != 1 {
		t.Errorf("matches = %d rows, want 1", rows)
	}
}

// A blob that answers 503 is the SAME transport, the OPPOSITE verdict: the film may well
// be there in ten minutes. Recording it as expired would bury it forever.
func TestFetchOne_TransientChunkFailureIsNotTerminal(t *testing.T) {
	film := map[int][]byte{0: []byte("header"), 1: []byte("replication")}
	srv := newFakeHalo(t, film, statsWithMap("Cliffhanger"))
	srv.chunkStatus = http.StatusServiceUnavailable

	builds := 0
	d := srv.deps(t, countingBuild(&builds, stubBuild(nil)))

	_, _, found, err := runFetch(t, d)
	if err == nil {
		t.Fatal("a 503 on the film blobs was reported as success")
	}
	if found {
		t.Fatal("a transient blob failure left a row behind - the film is now unreachable forever")
	}

	// The CDN recovers, and the next run archives the match normally.
	srv.chunkStatus = 0
	out, rec, found, err := runFetch(t, d)
	if err != nil {
		t.Fatalf("second pass after the CDN recovered: %v", err)
	}
	if !found || rec.State != stateDownloaded || rec.ArtifactPath == "" {
		t.Errorf("state/artifact = %q/%q, want %q and a built artifact",
			rec.State, rec.ArtifactPath, stateDownloaded)
	}
	if out.ChunksWritten != len(film) || builds != 1 {
		t.Errorf("wrote %d chunks over %d builds, want %d/1", out.ChunksWritten, builds, len(film))
	}
}

// Match stats that fail are a transport problem too: no row at all, so the next run
// starts from scratch. 429 stands in for the whole family because the client abandons it
// immediately - a 503 would only add its own retry ladder to the test's runtime.
func TestFetchOne_TransientStatsFailureRecordsNothing(t *testing.T) {
	srv := newFakeHalo(t, map[int][]byte{0: []byte("header")}, statsWithMap("Cliffhanger"))
	srv.statsStatus = http.StatusTooManyRequests

	d := srv.deps(t, stubBuild(nil))
	_, _, found, err := runFetch(t, d)
	if err == nil {
		t.Fatal("fetchOne reported success on a failing stats call")
	}
	if found {
		t.Error("a transient stats failure left a row behind")
	}
}

// A decoder that ERRORS is not a transport problem and not an expiry: the chunks are on
// disk and the failure will repeat identically until the decoder changes. It is recorded
// as `failed`, with its own reason - and it is STILL an error for the caller, so a
// decoder regression never passes for an ordinary archiving outcome.
func TestFetchOne_BuildFailureIsRecordedAsFailed(t *testing.T) {
	film := map[int][]byte{0: []byte("header"), 1: []byte("replication")}
	srv := newFakeHalo(t, film, statsWithMap("Cliffhanger"))
	boom := errors.New("decoder exploded")
	d := srv.deps(t, func(string, string, string, replay.Options) (replay.ReplayDocument, error) {
		return replay.ReplayDocument{}, boom
	})

	out, rec, found, err := runFetch(t, d)
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want the decoder's error", err)
	}
	if !found {
		t.Fatal("a build failure left no row: the archive cannot tell it from a match never seen")
	}
	if rec.State != stateFailed || rec.SkipReason != skipBuildFailed {
		t.Errorf("state/reason = %q/%q, want %q/%q",
			rec.State, rec.SkipReason, stateFailed, skipBuildFailed)
	}
	if rec.ArtifactPath != "" {
		t.Errorf("artifact_path = %q on a build that failed", rec.ArtifactPath)
	}
	// The film is kept: that is what makes the match rebuildable at all.
	if out.ChunksWritten != len(film) {
		t.Errorf("wrote %d chunks, want %d - the film must survive a failed build",
			out.ChunksWritten, len(film))
	}
	for idx := range film {
		if _, statErr := os.Stat(d.Paths.FilmChunkPath(testMatchID, idx)); statErr != nil {
			t.Errorf("chunk %d not kept: %v", idx, statErr)
		}
	}
}

// THE OTHER HALF OF A FAILED BUILD: the decoder succeeded and the DISK refused. That says
// nothing about the match, so it must leave no state behind — recording `failed` here
// would blame the decoder for a full disk, and the match would be reported as needing a
// decoder fix it does not need.
func TestFetchOne_ArtifactWriteFailureIsNotRecorded(t *testing.T) {
	srv := newFakeHalo(t, map[int][]byte{0: []byte("header"), 1: []byte("replication")},
		statsWithMap("Cliffhanger"))
	d := srv.deps(t, stubBuild(nil))

	// A plain FILE where the artifact's directory has to be: MkdirAll cannot pass through
	// it, so the write fails after a decode that went perfectly well.
	artifactDir := filepath.Dir(d.Paths.ReplayArtifactPath(title.DefaultSlug, testMatchID))
	if err := os.MkdirAll(filepath.Dir(artifactDir), cacheDirPerm); err != nil {
		t.Fatalf("preparing the artifact directory: %v", err)
	}
	if err := os.WriteFile(artifactDir, []byte("not a directory"), cacheFilePerm); err != nil {
		t.Fatalf("blocking the artifact directory: %v", err)
	}

	_, _, found, err := runFetch(t, d)
	if err == nil {
		t.Fatal("fetchOne reported success although the artifact could not be written")
	}
	if found {
		t.Error("a disk failure was recorded - the next run would read it as a decoder problem")
	}
}

// `failed` is not terminal: a later decoder rescues the match. Both flavours are checked,
// because they arrive by different paths - a decoder that errors, and one that returns a
// document with no trajectory at all.
//
// AND THE CDN IS KILLED BETWEEN THE TWO PASSES, which is the whole point. A decoder fix is
// months away; by the time it lands the film link is certainly dead. If the rebuild went
// back to the CDN it would collect a 404 and record `expired`, burying a match whose film
// is on disk intact — so this asserts the rebuild happens with the film route answering
// 404/410, and that the recorded state does not decay.
func TestFetchOne_FailedMatchesStayRebuildable(t *testing.T) {
	film := map[int][]byte{0: []byte("header"), 1: []byte("replication")}
	empty := replay.ReplayDocument{SchemaVersion: replay.SchemaVersion}

	cases := map[string]buildFilm{
		"the decoder errored": func(string, string, string, replay.Options) (replay.ReplayDocument, error) {
			return replay.ReplayDocument{}, errors.New("decoder exploded")
		},
		"the decoder found no trajectory": stubBuild(&empty),
	}
	for name, broken := range cases {
		t.Run(name, func(t *testing.T) {
			srv := newFakeHalo(t, film, statsWithMap("Cliffhanger"))
			d := srv.deps(t, broken)

			_, rec, found, _ := runFetch(t, d)
			if !found || rec.State != stateFailed {
				t.Fatalf("first pass state = %q (found=%v), want %q", rec.State, found, stateFailed)
			}

			// Months pass: the decoder is fixed and the film link dies.
			srv.manifestStatus = http.StatusGone
			srv.chunkStatus = http.StatusNotFound
			blobs := srv.blobCalls.Load()

			builds := 0
			d.Build = countingBuild(&builds, stubBuild(nil))
			second, rec2, _, err := runFetch(t, d)
			if err != nil {
				t.Fatalf("rebuild after a decoder fix: %v", err)
			}
			if got := srv.blobCalls.Load(); got != blobs {
				t.Errorf("the rebuild went back to the CDN (%d blob calls, want %d) - "+
					"a captured film must be rebuilt from disk", got, blobs)
			}
			if second.Settled {
				t.Error("a failed match was treated as settled - a decoder fix could never rescue it")
			}
			if builds != 1 {
				t.Errorf("%d builds on the second pass, want 1", builds)
			}
			if rec2.State != stateDownloaded || rec2.SkipReason != "" {
				t.Errorf("state/reason after the rebuild = %q/%q, want %q and no reason",
					rec2.State, rec2.SkipReason, stateDownloaded)
			}
			if rec2.ArtifactPath == "" {
				t.Error("the rebuilt artifact was not recorded")
			}
			if _, statErr := os.Stat(d.Paths.ReplayArtifactPath(title.DefaultSlug, testMatchID)); statErr != nil {
				t.Errorf("the artifact was not written: %v", statErr)
			}
		})
	}
}

// THE REGRESSION THIS TICKET NEARLY SHIPPED: a `failed` match, re-run once its CDN link
// has died, must NOT be rewritten `expired`. The two states are not a severity ordering —
// `expired` means "the bytes are gone", and the bytes are right there on disk. Rewriting
// it would make the match terminal and bury a film a decoder fix could still rescue.
func TestFetchOne_ADeadCDNDoesNotExpireACapturedFilm(t *testing.T) {
	film := map[int][]byte{0: []byte("header"), 1: []byte("replication")}
	srv := newFakeHalo(t, film, statsWithMap("Cliffhanger"))
	empty := replay.ReplayDocument{SchemaVersion: replay.SchemaVersion}
	d := srv.deps(t, stubBuild(&empty))

	if _, rec, _, _ := runFetch(t, d); rec.State != stateFailed {
		t.Fatalf("first pass state = %q, want %q", rec.State, stateFailed)
	}

	// The film link dies, and the decoder is still broken: the pass fails the same way.
	srv.manifestStatus = http.StatusGone
	srv.chunkStatus = http.StatusNotFound

	_, rec, _, err := runFetch(t, d)
	if err != nil {
		t.Fatalf("second pass: %v", err)
	}
	if rec.State != stateFailed || rec.SkipReason != skipNoTracks {
		t.Fatalf("state/reason = %q/%q, want %q/%q - a dead link overwrote a captured film",
			rec.State, rec.SkipReason, stateFailed, skipNoTracks)
	}
	// And it is still not settled, so the decoder fix that lands next month still gets a go.
	third, _, _, _ := runFetch(t, d)
	if third.Settled {
		t.Error("the match became terminal: no later decoder fix could ever rescue it")
	}
	for idx := range film {
		if _, statErr := os.Stat(d.Paths.FilmChunkPath(testMatchID, idx)); statErr != nil {
			t.Errorf("chunk %d was lost: %v", idx, statErr)
		}
	}
}

// An unsupported map is neither terminal state: the catalogue grows, and the match must
// be re-attempted until it does. It is the third answer the two-state reading would lose.
func TestFetchOne_UnsupportedMapStaysRetryable(t *testing.T) {
	srv := newFakeHalo(t, map[int][]byte{0: []byte("header"), 1: []byte("replication")},
		statsWithMap("Forbidden Sands"))
	d := srv.deps(t, stubBuild(nil))

	if _, rec, found, _ := runFetch(t, d); !found || rec.State != stateDownloaded {
		t.Fatalf("first pass state = %q (found=%v), want %q", rec.State, found, stateDownloaded)
	}
	second, _, _, err := runFetch(t, d)
	if err != nil {
		t.Fatalf("second pass: %v", err)
	}
	if second.Settled {
		t.Error("an unsupported map was treated as settled - a catalogue update could never rescue it")
	}
}
