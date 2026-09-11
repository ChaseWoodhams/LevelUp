package main

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/domain/title"
)

const (
	unbuiltOlderID  = "10000000-0000-4000-8000-000000000001"
	unbuiltNewerID  = "11000000-0000-4000-8000-000000000011"
	recapExpiredID  = "20000000-0000-4000-8000-000000000002"
	recapFailedID   = "30000000-0000-4000-8000-000000000003"
	recapArchivedID = "40000000-0000-4000-8000-000000000004"
)

// seedRecaptureRow records one match in a given film state, played at a given instant.
func seedRecaptureRow(t *testing.T, d deps, id string, state filmState, skip reason, artifact string, played time.Time) {
	t.Helper()
	rec, roster := sampleRecord()
	rec.MatchID, rec.ShortID, rec.PlayedAt = id, title.FilmShortMatchID(id), &played
	rec.State, rec.SkipReason, rec.ArtifactPath = state, skip, artifact
	if artifact == "" {
		rec.BuiltAt, rec.DecoderRev = nil, ""
	}
	if err := d.Archive.recordMatch(context.Background(), rec, roster); err != nil {
		t.Fatalf("seeding %s: %v", id, err)
	}
}

func recaptureFixture(t *testing.T, ids ...string) (*fakeHalo, deps) {
	t.Helper()
	f := newFakeHalo(t, map[int][]byte{0: []byte("header"), 1: []byte("replication"), 2: []byte("highlights")}, nil)
	f.statsByMatch = map[string]map[string]any{}
	for _, id := range ids {
		f.statsByMatch[id] = statsFor(id, "Cliffhanger", twoPlayerRoster())
	}
	return f, f.deps(t, stubBuild(nil))
}

// Only a recorded match with no artifact, whose film is neither expired nor failed, is fetched.
func TestRecapture_FetchesOnlyWhatIsRecordedButUnbuilt(t *testing.T) {
	f, d := recaptureFixture(t, unbuiltOlderID)
	played := time.Date(2026, 5, 19, 20, 15, 0, 0, time.UTC)
	seedRecaptureRow(t, d, unbuiltOlderID, stateDownloaded, skipUnsupportedMap, "", played)
	seedRecaptureRow(t, d, recapExpiredID, stateExpired, skipFilmAbsent, "", played)
	seedRecaptureRow(t, d, recapFailedID, stateFailed, skipBuildFailed, "", played)
	seedRecaptureRow(t, d, recapArchivedID, stateDownloaded, "", "/archive/4.json", played)

	sum := recapturePass(context.Background(), d, 0)

	if sum.Candidates != 1 || sum.Archived != 1 || sum.Failed != 0 || sum.Skipped != 0 {
		t.Errorf("summary = %+v, want one candidate, archived", sum)
	}
	if n := f.statsCalls.Load(); n != 1 {
		t.Errorf("stats calls = %d, want 1 - expired, failed and archived matches are not recaptured", n)
	}
	rec, found, err := d.Archive.recorded(context.Background(), unbuiltOlderID)
	if err != nil || !found || rec.ArtifactPath == "" || rec.SkipReason != "" {
		t.Errorf("recaptured row = %+v (found=%v err=%v), want an artifact and no skip reason", rec, found, err)
	}
}

// A limited run takes the OLDEST films first: they are the closest to expiring.
func TestRecapture_LimitTakesTheOldestFirst(t *testing.T) {
	f, d := recaptureFixture(t, unbuiltOlderID, unbuiltNewerID)
	seedRecaptureRow(t, d, unbuiltNewerID, stateDownloaded, skipUnsupportedMap, "",
		time.Date(2026, 6, 1, 20, 0, 0, 0, time.UTC))
	seedRecaptureRow(t, d, unbuiltOlderID, stateDownloaded, skipUnsupportedMap, "",
		time.Date(2026, 5, 1, 20, 0, 0, 0, time.UTC))

	sum := recapturePass(context.Background(), d, 1)

	if sum.Candidates != 1 || sum.Archived != 1 || f.statsCalls.Load() != 1 {
		t.Fatalf("summary = %+v, stats calls = %d, want exactly one match recaptured", sum, f.statsCalls.Load())
	}
	older, _, _ := d.Archive.recorded(context.Background(), unbuiltOlderID)
	newer, _, _ := d.Archive.recorded(context.Background(), unbuiltNewerID)
	if older.ArtifactPath == "" || newer.ArtifactPath != "" {
		t.Errorf("older artifact %q / newer artifact %q, want the older match taken first",
			older.ArtifactPath, newer.ArtifactPath)
	}
}

// Five failures in a row are not five bad matches: the run stops instead of burning the list.
func TestRecapture_StopsAfterConsecutiveFailures(t *testing.T) {
	f, d := recaptureFixture(t)
	// 401, not 500: the Halo client retries a 5xx with backoff, which would make this test
	// spend a minute proving the same thing.
	f.statsStatus = 401
	base := time.Date(2026, 5, 1, 20, 0, 0, 0, time.UTC)
	for i := 0; i < maxConsecutiveRecaptureFailures+3; i++ {
		id := "5000000" + string(rune('0'+i)) + "-0000-4000-8000-00000000000" + string(rune('0'+i))
		seedRecaptureRow(t, d, id, stateDownloaded, skipUnsupportedMap, "", base.Add(time.Duration(i)*time.Hour))
	}

	sum := recapturePass(context.Background(), d, 0)

	if !sum.Stopped || sum.Failed != maxConsecutiveRecaptureFailures {
		t.Errorf("summary = %+v, want stopped after %d failures", sum, maxConsecutiveRecaptureFailures)
	}
}
