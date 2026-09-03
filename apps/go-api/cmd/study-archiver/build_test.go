package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain/title"
)

// busyBuildDelay is how long the overlap probe stays inside the build. Long enough that
// unsynchronised callers would certainly be caught overlapping, short enough not to slow
// the suite. A sleep rather than a spin loop: a spin's duration depends on the machine.
const busyBuildDelay = 3 * time.Millisecond

// overlapProbe is a build function that records the highest number of builds ever in
// flight at the same time.
type overlapProbe struct {
	inFlight atomic.Int32
	peak     atomic.Int32
}

func (p *overlapProbe) build(matchID, titleSlug, filmDir string, opt replay.Options) (replay.ReplayDocument, error) {
	n := p.inFlight.Add(1)
	for {
		peak := p.peak.Load()
		if n <= peak || p.peak.CompareAndSwap(peak, n) {
			break
		}
	}
	time.Sleep(busyBuildDelay)
	p.inFlight.Add(-1)
	return replay.ReplayDocument{
		Tracks: []replay.Track{{Slot: 1, Team: -1, Points: []replay.Point{{T: 0}, {T: 1}}}},
	}, nil
}

// TestFetchOne_BuildsNeverOverlap — the guard behind the answer this ticket owed:
// internal/analysis/filmdec does NOT serialise itself (cf. build.go), so the archiver
// must. This drives the REAL fetch-one path concurrently, not the lock in isolation: the
// point of the finding is that serialisation must belong to the call site rather than to
// one wiring of deps, so the test has to go through the call site to prove it.
//
// The probe is a build function, not the real decoder: what is under test is the
// archiver's lock, not filmdec.
func TestFetchOne_BuildsNeverOverlap(t *testing.T) {
	const archivers = 8
	srv := newFakeHalo(t, map[int][]byte{0: []byte("header"), 1: []byte("replication")},
		statsWithMap("Cliffhanger"))
	var probe overlapProbe

	var wg sync.WaitGroup
	for i := 0; i < archivers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// A path resolver each, so the concurrency under test is the BUILD and not a
			// race to write the same chunk file.
			d := srv.deps(t, probe.build)
			if _, err := fetchOne(context.Background(), d, testMatchID); err != nil {
				t.Errorf("fetchOne: %v", err)
			}
		}()
	}
	wg.Wait()

	if got := probe.peak.Load(); got != 1 {
		t.Errorf("%d concurrent builds observed, want 1 - the decoder's package-level "+
			"state would have been shared between them", got)
	}
}

func TestWriteArtifact_CreatesParentsAndValidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "replays", title.DefaultSlug, "000d5950.json")
	doc := replay.ReplayDocument{MatchID: "000d5950", TitleSlug: title.DefaultSlug}

	size, err := writeArtifact(path, doc)
	if err != nil {
		t.Fatalf("writeArtifact: %v", err)
	}
	blob, err := os.ReadFile(path) //nolint:gosec // path built by the test
	if err != nil {
		t.Fatalf("artifact unreadable: %v", err)
	}
	if len(blob) != size {
		t.Errorf("reported size %d, file holds %d bytes", size, len(blob))
	}
	var back replay.ReplayDocument
	if err := json.Unmarshal(blob, &back); err != nil {
		t.Fatalf("artifact is not valid JSON: %v", err)
	}
	if back.MatchID != "000d5950" {
		t.Errorf("match id = %q, want 000d5950", back.MatchID)
	}
}
