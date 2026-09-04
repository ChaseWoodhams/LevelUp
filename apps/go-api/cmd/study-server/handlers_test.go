package main

// handlers_test.go — the three routes, against a real archive and a real artifact on disk.
//
// The shape follows internal/api/handlers/replay_test.go, the closest existing handler test:
// a chi router built the way the server builds it, httptest requests through the whole Huma
// chain, and assertions on the status and on the error CODE rather than on the message.
// Where it differs is the fixture — a temp archive database and a temp artifact rather than a
// mocked service — because what the ticket asks to prove (a read-only archive, a resolved
// path, a clean not-found) lives in the seams a mock would replace.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/domain/title"
)

// newTestServer wires the handler over a seeded archive and a seeded artifact directory.
func newTestServer(t *testing.T) *chi.Mux {
	r, _ := newTestServerWithArtifacts(t)
	return r
}

// newTestServerWithArtifacts also hands back the artifact directory, for the tests that need
// to put a file where the archive does not expect one.
func newTestServerWithArtifacts(t *testing.T) (*chi.Mux, artifacts) {
	t.Helper()
	art := newTestArtifacts(t)
	h := &studyHandler{archive: newTestArchive(t), artifacts: art}
	r := chi.NewRouter()
	h.mount(r)
	return r, art
}

func get(t *testing.T, r *chi.Mux, path string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	return w
}

// decode reads a JSON body, failing the test with the body itself when it will not parse —
// which is what a 500 rendered as an error object looks like from here.
func decode[T any](t *testing.T, w *httptest.ResponseRecorder) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("unreadable body (%d): %s", w.Code, w.Body.String())
	}
	return out
}

func expectStatus(t *testing.T, w *httptest.ResponseRecorder, want int, code string) {
	t.Helper()
	if w.Code != want {
		t.Fatalf("status = %d, want %d: %s", w.Code, want, w.Body.String())
	}
	if code != "" && !strings.Contains(w.Body.String(), code) {
		t.Errorf("body must carry the error code %q, got %s", code, w.Body.String())
	}
}

func TestListMatchesRoute(t *testing.T) {
	w := get(t, newTestServer(t), "/matches")
	expectStatus(t, w, http.StatusOK, "")

	page := decode[matchPage](t, w)
	if page.Total != 3 || len(page.Matches) != 3 {
		t.Fatalf("page = %d rows of %d total, want 3 of 3", len(page.Matches), page.Total)
	}
	if page.Matches[0].MapName != "Aquarius" {
		t.Errorf("first row = %s, want the most recent match", page.Matches[0].MapName)
	}
}

func TestListMatchesRoute_Filters(t *testing.T) {
	r := newTestServer(t)
	cases := []struct {
		query string
		want  int
	}{
		{"?map=Cliffhanger", 1},
		{"?mode=Slayer", 2},
		{"?player=Rival", 2},
		{"?from=2026-05-19&to=2026-05-19", 1},
		{"?min_coverage=0.5", 1},
		{"?limit=1", 1},
	}
	for _, c := range cases {
		t.Run(c.query, func(t *testing.T) {
			w := get(t, r, "/matches"+c.query)
			expectStatus(t, w, http.StatusOK, "")
			if got := len(decode[matchPage](t, w).Matches); got != c.want {
				t.Errorf("rows = %d, want %d", got, c.want)
			}
		})
	}
}

// TestListMatchesRoute_BadFilter — a query string that cannot be honoured is the CALLER's
// mistake and says so, rather than arriving as an empty table that reads like an empty
// archive.
func TestListMatchesRoute_BadFilter(t *testing.T) {
	r := newTestServer(t)
	for _, query := range []string{"?from=hier", "?min_coverage=85", "?limit=999999"} {
		t.Run(query, func(t *testing.T) {
			expectStatus(t, get(t, r, "/matches"+query), http.StatusBadRequest, "invalid_filter")
		})
	}
}

func TestReplayRoute(t *testing.T) {
	r := newTestServer(t)
	// Either form of the identifier: the archive browser links with what it was given.
	for _, id := range []string{cliffhangerID, "000d5950"} {
		w := get(t, r, "/matches/"+id+"/replay")
		expectStatus(t, w, http.StatusOK, "")
		doc := decode[map[string]any](t, w)
		if doc["matchId"] != "000d5950" {
			t.Errorf("document = %v, want the Cliffhanger artifact", doc["matchId"])
		}
		if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
	}
}

// TestReplayRoute_UnknownMatch is the ticket's own criterion: an id nobody archived — and an
// id the archiver recorded but never managed to build — comes back as a clean not-found
// rather than as a 500.
func TestReplayRoute_UnknownMatch(t *testing.T) {
	r := newTestServer(t)
	for _, id := range []string{"deadbeef", unbuiltID} {
		t.Run(id, func(t *testing.T) {
			expectStatus(t, get(t, r, "/matches/"+id+"/replay"), http.StatusNotFound, "match_not_found")
		})
	}
}

// TestReplayRoute_PathComesFromTheArchive is the security property stated as a test: the
// filesystem path is built from a short_id READ OFF A ROW, never from the URL. An artifact
// sitting in the cache for a match the archive does not hold is therefore unreachable — and
// with it, any path a caller might try to name.
func TestReplayRoute_PathComesFromTheArchive(t *testing.T) {
	r, art := newTestServerWithArtifacts(t)
	const strangerID = "999f0e1d-2c3b-4a59-8768-5a4b3c2d1e0f"
	writeTestArtifact(t, art, strangerID)

	expectStatus(t, get(t, r, "/matches/"+strangerID+"/replay"), http.StatusNotFound, "match_not_found")
	expectStatus(t, get(t, r, "/matches/999f0e1d/replay"), http.StatusNotFound, "match_not_found")
}

// TestReplayRoute_ArchivedButFileGone — the archive says built, the file is not there. A torn
// state, and a DIFFERENT answer from "unknown match": the operator's next move is to look at
// the cache, not at the id they typed.
func TestReplayRoute_ArchivedButFileGone(t *testing.T) {
	// The fixture artifact directory holds Cliffhanger alone; Streets is archived in the
	// database and has no file.
	w := get(t, newTestServer(t), "/matches/"+streetsID+"/replay")
	expectStatus(t, w, http.StatusNotFound, "replay_not_available")
}

func TestParticipantsRoute(t *testing.T) {
	w := get(t, newTestServer(t), "/matches/000d5950/participants")
	expectStatus(t, w, http.StatusOK, "")

	body := decode[struct {
		Participants []participantRow `json:"participants"`
	}](t, w)
	if len(body.Participants) != 2 {
		t.Fatalf("participants = %d, want 2", len(body.Participants))
	}
	if body.Participants[0].Gamertag != "JGtm" || *body.Participants[0].TeamSide != "t0" {
		t.Errorf("first participant = %+v", body.Participants[0])
	}
}

// TestParticipantsRoute_ShapeIsTheScoreboardSubset pins the payload's keys. The study viewer
// reuses the app's roster logic unchanged, and that logic reads exactly these — a renamed
// key here is a roster that silently groups everybody into one nameless team.
func TestParticipantsRoute_ShapeIsTheScoreboardSubset(t *testing.T) {
	w := get(t, newTestServer(t), "/matches/000d5950/participants")
	expectStatus(t, w, http.StatusOK, "")

	body := decode[struct {
		Participants []map[string]any `json:"participants"`
	}](t, w)
	got := body.Participants[0]
	for _, key := range []string{"xuid", "gamertag", "team_side", "kills", "deaths", "assists"} {
		if _, ok := got[key]; !ok {
			t.Errorf("key %q missing from the participant payload: %v", key, got)
		}
	}
	if len(got) != 6 {
		t.Errorf("participant carries %d keys, want the 6 the roster logic reads: %v", len(got), got)
	}
}

func TestParticipantsRoute_UnknownMatch(t *testing.T) {
	expectStatus(t, get(t, newTestServer(t), "/matches/deadbeef/participants"),
		http.StatusNotFound, "match_not_found")
}

// TestServerDefaults documents the two choices the binary makes for the operator, in the one
// place a change to either of them would be noticed.
func TestServerDefaults(t *testing.T) {
	if !strings.HasPrefix(defaultAddr, "127.0.0.1:") {
		t.Errorf("defaultAddr = %q: the study server holds an archive of other people's "+
			"matches and must not listen on a public interface by default", defaultAddr)
	}
	if title.DefaultSlug == "" {
		t.Error("the default title slug must resolve")
	}
}
