package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain/title"
)

// newTestArtifacts writes one artifact under a throwaway repo root and returns the reader.
func newTestArtifacts(t *testing.T) artifacts {
	t.Helper()
	root := t.TempDir()
	a := artifacts{paths: title.NewPathResolver(root), titleSlug: title.DefaultSlug}
	writeTestArtifact(t, a, cliffhangerID)
	return a
}

func writeTestArtifact(t *testing.T, a artifacts, matchID string) {
	t.Helper()
	path := a.paths.ReplayArtifactPath(a.titleSlug, matchID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("artifact directory: %v", err)
	}
	blob, err := json.Marshal(testDocument(matchID))
	if err != nil {
		t.Fatalf("encoding the fixture artifact: %v", err)
	}
	if err := os.WriteFile(path, blob, 0o644); err != nil {
		t.Fatalf("writing the fixture artifact: %v", err)
	}
}

func testDocument(matchID string) replay.ReplayDocument {
	return replay.ReplayDocument{
		SchemaVersion: replay.SchemaVersion,
		MatchID:       title.FilmShortMatchID(matchID),
		TitleSlug:     title.DefaultSlug,
		FrameCount:    2,
		Tracks: []replay.Track{{
			Slot: 665, Team: -1, XUID: "2533274823110022",
			Points: []replay.Point{{T: 0, X: 1, Y: 2}},
		}},
	}
}

// TestArtifacts_ReadResolvesTheShortForm — the artifact is filed under the short film id, and
// handing the reader a full match id must reach the same file. This is the crossing that once
// produced a 404 on a file that was present.
func TestArtifacts_ReadResolvesTheShortForm(t *testing.T) {
	a := newTestArtifacts(t)
	full, err := a.read(cliffhangerID)
	if err != nil {
		t.Fatalf("read(full id): %v", err)
	}
	short, err := a.read("000d5950")
	if err != nil {
		t.Fatalf("read(short id): %v", err)
	}
	if string(full) != string(short) {
		t.Error("both forms of the identifier must resolve to the same artifact")
	}
}

// TestArtifacts_ReadIsVerbatim — the response is the file, not a re-encoding of it. A field
// this repo's ReplayDocument does not model must survive the trip, because the viewer's
// schema-version guard is what is meant to catch it.
func TestArtifacts_ReadIsVerbatim(t *testing.T) {
	root := t.TempDir()
	a := artifacts{paths: title.NewPathResolver(root), titleSlug: title.DefaultSlug}
	path := a.paths.ReplayArtifactPath(a.titleSlug, cliffhangerID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("artifact directory: %v", err)
	}
	onDisk := `{"schemaVersion":99,"matchId":"000d5950","somethingNewer":[1,2,3]}`
	if err := os.WriteFile(path, []byte(onDisk), 0o644); err != nil {
		t.Fatalf("writing the artifact: %v", err)
	}

	got, err := a.read(cliffhangerID)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != onDisk {
		t.Errorf("read = %s, want the file unchanged: %s", got, onDisk)
	}
}

func TestArtifacts_ReadMissing(t *testing.T) {
	a := newTestArtifacts(t)
	if _, err := a.read(streetsID); !errors.Is(err, errArtifactMissing) {
		t.Errorf("err = %v, want errArtifactMissing", err)
	}
}
