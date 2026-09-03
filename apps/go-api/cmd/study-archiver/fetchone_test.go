package main

// fetchone_test.go — END TO END OVER A FAKE HALO SERVER.
//
// The archiver's own seam is `fetchOne`, driven through the REAL
// haloclient.HaloAPIClient (manifest read, parallel blob download, zlib inflate, stats
// call) pointed at an httptest server. Only the BUILD is stubbed: replay.BuildFromFilm
// needs a whole film, the decoder has its own suites (golden assembly, mini-reel), and
// re-testing it here would test somebody else's code. The real decoder is exercised by
// the fixture-gated test in fetchone_realfilm_test.go.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/sync/haloclient"
)

// testMatchID is a well-formed match id — the client rejects anything else — whose
// short form is the film cache key 000d5950.
const testMatchID = "000d5950-1234-4abc-9def-0123456789ab"

// fakeHalo serves the three routes fetch-one touches: the film manifest, the film
// blobs, and the match stats.
type fakeHalo struct {
	*httptest.Server
	// chunks is the film, keyed by manifest index; the payloads are served zlib-encoded
	// because that is what the Halo blob CDN returns and what downloadBlob inflates.
	chunks map[int][]byte
	// manifestStatus, when non-zero, replaces the manifest response (410 = film expired).
	manifestStatus int
	stats          map[string]any
	// statsCalls / manifestCalls count what the archiver actually asked the API for.
	// Idempotency is a claim about NOT fetching, and only a call count can prove it —
	// row counts alone would still pass if the tool re-downloaded the whole film.
	statsCalls    atomic.Int32
	manifestCalls atomic.Int32
}

func newFakeHalo(t *testing.T, chunks map[int][]byte, stats map[string]any) *fakeHalo {
	t.Helper()
	f := &fakeHalo{chunks: chunks, stats: stats}
	mux := http.NewServeMux()
	mux.HandleFunc("/hi/films/matches/", func(w http.ResponseWriter, r *http.Request) {
		f.manifestCalls.Add(1)
		if f.manifestStatus != 0 {
			w.WriteHeader(f.manifestStatus)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(f.manifestJSON())
	})
	mux.HandleFunc("/ugcstorage/", func(w http.ResponseWriter, r *http.Request) {
		idx, err := parseChunkIndex(filepath.Base(r.URL.Path))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		data, ok := f.chunks[idx]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(zlibBytes(t, data))
	})
	mux.HandleFunc("/hi/matches/", func(w http.ResponseWriter, r *http.Request) {
		f.statsCalls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		blob, err := json.Marshal(f.stats)
		if err != nil {
			t.Errorf("encoding stats: %v", err)
			return
		}
		_, _ = w.Write(blob)
	})
	f.Server = httptest.NewServer(mux)
	t.Cleanup(f.Server.Close)
	return f
}

// manifestJSON describes the film exactly as the Halo UGC endpoint does: chunk 0 is the
// header, the middle chunks are replication data, the last one is the highlight feed.
// All three types are declared so the test proves fetch-one downloads the WHOLE film and
// not just the replication chunks the weapon scanner needs.
func (f *fakeHalo) manifestJSON() []byte {
	type chunk struct {
		Index                            int    `json:"Index"`
		ChunkType                        int    `json:"ChunkType"`
		ChunkStartTimeOffsetMilliseconds int    `json:"ChunkStartTimeOffsetMilliseconds"`
		DurationMilliseconds             int    `json:"DurationMilliseconds"`
		FileRelativePath                 string `json:"FileRelativePath"`
	}
	var chunks []chunk
	for idx := 0; idx < len(f.chunks); idx++ {
		kind := haloclient.FilmChunkTypeReplicationData
		switch idx {
		case 0:
			kind = haloclient.FilmChunkTypeHeader
		case len(f.chunks) - 1:
			kind = haloclient.FilmChunkTypeHighlightEvents
		}
		chunks = append(chunks, chunk{
			Index:            idx,
			ChunkType:        kind,
			FileRelativePath: fmtChunkName(idx),
		})
	}
	body := map[string]any{
		"BlobStoragePathPrefix": f.Server.URL + "/ugcstorage/film/test/test/",
		"CustomData": map[string]any{
			"FilmMajorVersion": 40,
			"Chunks":           chunks,
		},
	}
	blob, _ := json.Marshal(body)
	return blob
}

// client builds a real Halo client whose every request lands on the fake server.
func (f *fakeHalo) client() *haloclient.HaloAPIClient {
	return haloclient.NewHaloAPIClient("spartan-test", "clearance-test", 1000).
		WithHTTPClient(&http.Client{Transport: redirectTo(f.Server.URL)})
}

// deps wires the archiver against the fake server, a throwaway repo root, a throwaway
// archive database and a build function the test controls.
func (f *fakeHalo) deps(t *testing.T, build buildFilm) deps {
	t.Helper()
	return deps{
		Client:         f.client(),
		Paths:          title.NewPathResolver(t.TempDir()),
		Title:          title.DefaultSlug,
		Catalog:        testCatalog(),
		Archive:        testArchive(t),
		Build:          build,
		SourceGamertag: "JGtm",
	}
}

// stubBuild returns a document with one two-point track — the smallest thing that is
// not "no tracks decoded".
func stubBuild(doc *replay.ReplayDocument) buildFilm {
	return func(matchID, titleSlug, filmDir string, opt replay.Options) (replay.ReplayDocument, error) {
		if doc != nil {
			return *doc, nil
		}
		return replay.ReplayDocument{
			SchemaVersion: replay.SchemaVersion,
			MatchID:       matchID,
			TitleSlug:     titleSlug,
			Tracks: []replay.Track{{
				Slot:   1,
				Team:   -1,
				Points: []replay.Point{{T: 0}, {T: 1}},
			}},
		}, nil
	}
}

func TestFetchOne_WritesEveryChunkAndTheArtifact(t *testing.T) {
	film := map[int][]byte{
		0: []byte("header-chunk"),
		1: []byte("replication-chunk-one"),
		2: []byte("replication-chunk-two"),
		3: []byte("highlight-chunk"),
	}
	srv := newFakeHalo(t, film, statsWithMap("Cliffhanger"))

	var seenFilmDir string
	d := srv.deps(t, func(matchID, titleSlug, filmDir string, opt replay.Options) (replay.ReplayDocument, error) {
		seenFilmDir = filmDir
		if opt.WorldRange == nil {
			t.Error("build called without the map's bounds")
		}
		return stubBuild(nil)(matchID, titleSlug, filmDir, opt)
	})

	out, err := fetchOne(context.Background(), d, testMatchID)
	if err != nil {
		t.Fatalf("fetchOne: %v", err)
	}

	if out.SkipReason != "" {
		t.Fatalf("skipped for %q, want a build", out.SkipReason)
	}
	if out.ChunksWritten != len(film) {
		t.Errorf("wrote %d chunks, want %d (header and highlight included)",
			out.ChunksWritten, len(film))
	}
	// Every chunk lands at the path the offline tools already read, with the bytes the
	// CDN served once inflated.
	for idx, want := range film {
		path := d.Paths.FilmChunkPath(testMatchID, idx)
		got, readErr := os.ReadFile(path) //nolint:gosec // path from the resolver, under t.TempDir()
		if readErr != nil {
			t.Errorf("chunk %d not written: %v", idx, readErr)
			continue
		}
		if !bytes.Equal(got, want) {
			t.Errorf("chunk %d = %q, want %q", idx, got, want)
		}
	}
	if seenFilmDir != d.Paths.FilmChunksDir(testMatchID) {
		t.Errorf("build read %q, want the resolved film directory %q",
			seenFilmDir, d.Paths.FilmChunksDir(testMatchID))
	}

	wantArtifact := d.Paths.ReplayArtifactPath(title.DefaultSlug, testMatchID)
	if out.ArtifactPath != wantArtifact {
		t.Errorf("artifact path = %q, want %q", out.ArtifactPath, wantArtifact)
	}
	blob, err := os.ReadFile(wantArtifact) //nolint:gosec // path from the resolver, under t.TempDir()
	if err != nil {
		t.Fatalf("artifact not written: %v", err)
	}
	var doc replay.ReplayDocument
	if err := json.Unmarshal(blob, &doc); err != nil {
		t.Fatalf("artifact is not valid JSON: %v", err)
	}
	if doc.MatchID != testMatchID {
		t.Errorf("artifact match id = %q, want %q", doc.MatchID, testMatchID)
	}
	if out.Tracks != 1 || out.Points != 2 {
		t.Errorf("counts reported = %d tracks / %d points, want 1/2", out.Tracks, out.Points)
	}
}

// An unsupported map must skip the BUILD only. The film is still downloaded and kept:
// the catalogue can grow later (cmd/mapquant-build), the CDN link cannot come back.
func TestFetchOne_UnsupportedMapKeepsTheFilmAndWritesNoArtifact(t *testing.T) {
	film := map[int][]byte{0: []byte("header"), 1: []byte("replication")}
	srv := newFakeHalo(t, film, statsWithMap("Forbidden Sands"))

	built := false
	d := srv.deps(t, func(string, string, string, replay.Options) (replay.ReplayDocument, error) {
		built = true
		return replay.ReplayDocument{}, nil
	})

	out, err := fetchOne(context.Background(), d, testMatchID)
	if err != nil {
		t.Fatalf("fetchOne: %v", err)
	}
	if built {
		t.Error("the replay was built with another map's bounds")
	}
	if out.SkipReason != skipUnsupportedMap {
		t.Errorf("skip reason = %q, want %q", out.SkipReason, skipUnsupportedMap)
	}
	if out.ArtifactPath != "" {
		t.Errorf("artifact path reported (%q) for a skipped match", out.ArtifactPath)
	}
	if _, err := os.Stat(d.Paths.ReplayArtifactPath(title.DefaultSlug, testMatchID)); !os.IsNotExist(err) {
		t.Error("an artifact was written for an unsupported map")
	}
	if out.ChunksWritten != len(film) {
		t.Errorf("wrote %d chunks, want %d — the film must be kept for a later rebuild",
			out.ChunksWritten, len(film))
	}
	for idx := range film {
		if _, err := os.Stat(d.Paths.FilmChunkPath(testMatchID, idx)); err != nil {
			t.Errorf("chunk %d not kept: %v", idx, err)
		}
	}
}

// An expired film (410 on the manifest) is a named skip, not a crash: this is the race
// the whole tool exists to run, and losing it is a normal outcome.
func TestFetchOne_ExpiredFilmIsANamedSkip(t *testing.T) {
	srv := newFakeHalo(t, map[int][]byte{0: []byte("header")}, statsWithMap("Cliffhanger"))
	srv.manifestStatus = http.StatusGone

	d := srv.deps(t, stubBuild(nil))
	out, err := fetchOne(context.Background(), d, testMatchID)
	if err != nil {
		t.Fatalf("fetchOne: %v", err)
	}
	if out.SkipReason != skipFilmAbsent {
		t.Errorf("skip reason = %q, want %q", out.SkipReason, skipFilmAbsent)
	}
	if out.ChunksWritten != 0 {
		t.Errorf("wrote %d chunks for an absent film", out.ChunksWritten)
	}
}

// A film that downloads but decodes to nothing is distinct from an absent film: the
// chunks are on disk, so a later decoder fix can rebuild without the CDN.
func TestFetchOne_NoTracksDecodedIsANamedSkip(t *testing.T) {
	film := map[int][]byte{0: []byte("header"), 1: []byte("replication")}
	srv := newFakeHalo(t, film, statsWithMap("Cliffhanger"))

	empty := replay.ReplayDocument{SchemaVersion: replay.SchemaVersion}
	d := srv.deps(t, stubBuild(&empty))

	out, err := fetchOne(context.Background(), d, testMatchID)
	if err != nil {
		t.Fatalf("fetchOne: %v", err)
	}
	if out.SkipReason != skipNoTracks {
		t.Errorf("skip reason = %q, want %q", out.SkipReason, skipNoTracks)
	}
	if _, err := os.Stat(d.Paths.ReplayArtifactPath(title.DefaultSlug, testMatchID)); !os.IsNotExist(err) {
		t.Error("an artifact was written for a film that decoded to nothing")
	}
	if out.ChunksWritten != len(film) {
		t.Errorf("wrote %d chunks, want %d", out.ChunksWritten, len(film))
	}
}

// Match stats that carry no map name at all: distinct from an unsupported map, because
// nothing here says a rebuild would ever succeed. The film is still kept.
func TestFetchOne_StatsWithoutAMapNameIsANamedSkip(t *testing.T) {
	film := map[int][]byte{0: []byte("header"), 1: []byte("replication")}
	srv := newFakeHalo(t, film, statsWithMap("")) // a payload naming no map at all

	built := false
	d := srv.deps(t, func(string, string, string, replay.Options) (replay.ReplayDocument, error) {
		built = true
		return replay.ReplayDocument{}, nil
	})

	out, err := fetchOne(context.Background(), d, testMatchID)
	if err != nil {
		t.Fatalf("fetchOne: %v", err)
	}
	if built {
		t.Error("the replay was built without knowing the map")
	}
	if out.SkipReason != skipNoMapInStats {
		t.Errorf("skip reason = %q, want %q", out.SkipReason, skipNoMapInStats)
	}
	if out.ChunksWritten != len(film) {
		t.Errorf("wrote %d chunks, want %d - the film must be kept", out.ChunksWritten, len(film))
	}
}

// A build that FAILS is not a skip: it is an error for the caller to handle, so a bug in
// the decoder never looks like an ordinary archiving outcome.
func TestFetchOne_BuildErrorIsReturned(t *testing.T) {
	srv := newFakeHalo(t, map[int][]byte{0: []byte("header")}, statsWithMap("Cliffhanger"))
	boom := errors.New("decoder exploded")
	d := srv.deps(t, func(string, string, string, replay.Options) (replay.ReplayDocument, error) {
		return replay.ReplayDocument{}, boom
	})

	if _, err := fetchOne(context.Background(), d, testMatchID); !errors.Is(err, boom) {
		t.Errorf("err = %v, want the decoder's error", err)
	}
}
