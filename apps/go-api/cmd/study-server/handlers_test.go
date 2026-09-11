package main

// handlers_test.go — the four routes, against a real archive and a real artifact on disk.
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
	"path/filepath"
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
	h := &studyHandler{archive: newTestSource(t), artifacts: art}
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

// TestListMatchesRoute_CarriesItsRosters — the browser's rows say who played.
//
// Without them the table cannot colour a player chip or tell two matches on the same map apart,
// and the alternative — a roster request per row — would be two hundred round trips to a
// database this server holds for as short a time as it can.
func TestListMatchesRoute_CarriesItsRosters(t *testing.T) {
	w := get(t, newTestServer(t), "/matches")
	expectStatus(t, w, http.StatusOK, "")
	page := decode[matchPage](t, w)

	byMap := map[string][]participantRow{}
	for _, m := range page.Matches {
		byMap[m.MapName] = m.Participants
	}
	if got := len(byMap["Cliffhanger"]); got != 2 {
		t.Fatalf("Cliffhanger roster = %d players, want 2", got)
	}
	// Each row got ITS OWN roster: one query for the page must not smear players across matches.
	if got := byMap["Cliffhanger"][0].Gamertag; got != "JGtm" {
		t.Errorf("first Cliffhanger player = %q, want JGtm", got)
	}
	if got := len(byMap["Aquarius"]); got != 2 {
		t.Errorf("Aquarius roster = %d players, want 2", got)
	}
	// And the row the shape hangs on survives the page read too: a player the stats named no
	// team for keeps a null team side rather than being folded into one.
	for _, p := range byMap["Aquarius"] {
		if p.Gamertag == "Inconnu" && p.TeamSide != nil {
			t.Errorf("the team-less player came back with team_side %q", *p.TeamSide)
		}
	}
}

// TestGetMatchRoute_HasNoRoster — the single-match route deliberately leaves the roster out:
// the replay screen reads `/participants`, whose failure must fail that screen, while this
// summary is allowed to be missing.
func TestGetMatchRoute_HasNoRoster(t *testing.T) {
	w := get(t, newTestServer(t), "/matches/000d5950")
	expectStatus(t, w, http.StatusOK, "")
	if got := decode[matchSummary](t, w).Participants; got != nil {
		t.Errorf("summary carries %d participants, want none", len(got))
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

// TestGetMatchRoute — the single-match summary, under either form of the identifier.
//
// The field the viewer actually came for is `map_module`: a replay artifact carries no map, so
// without this route the floor's calibrated-image fallback has nothing to look a map up by.
func TestGetMatchRoute(t *testing.T) {
	r := newTestServer(t)
	for _, id := range []string{cliffhangerID, "000d5950"} {
		t.Run(id, func(t *testing.T) {
			w := get(t, r, "/matches/"+id)
			expectStatus(t, w, http.StatusOK, "")
			m := decode[matchSummary](t, w)
			if m.MatchID != cliffhangerID || m.ShortID != "000d5950" {
				t.Errorf("summary = %+v, want the Cliffhanger row", m)
			}
			if m.MapName != "Cliffhanger" || m.MapModule != "olympus" {
				t.Errorf("map = %q / module %q, want Cliffhanger / olympus", m.MapName, m.MapModule)
			}
			if m.Coverage == nil || *m.Coverage < 0.85 || *m.Coverage > 0.86 {
				t.Errorf("coverage = %v, want 90/105", m.Coverage)
			}
		})
	}
}

// TestGetMatchRoute_ServesARecordedButUnbuiltMatch — the map is a fact about the MATCH, not
// about whether its film could be decoded, so it is served like the roster and unlike the
// artifact.
func TestGetMatchRoute_ServesARecordedButUnbuiltMatch(t *testing.T) {
	w := get(t, newTestServer(t), "/matches/"+unbuiltID)
	expectStatus(t, w, http.StatusOK, "")
	if m := decode[matchSummary](t, w); m.MapName != "Cliffhanger" {
		t.Errorf("summary = %+v, want the unbuilt match's recorded row", m)
	}
}

// TestGetMatchRoute_UnknownMatch — a clean 404 with the same code the other routes use, so one
// front end handles one contract.
func TestGetMatchRoute_UnknownMatch(t *testing.T) {
	expectStatus(t, get(t, newTestServer(t), "/matches/deadbeef"), http.StatusNotFound, "match_not_found")
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

// TestParticipantsRoute_UnknownValuesAreNullNotMissing is the case the shape test above cannot
// see on its own, because every player in the Cliffhanger fixture has a team and counters.
//
// WHY IT MATTERS. The consumer's type declares these as `T | null`
// (`apps/web/src/lib/api/types.ts`: `team_side: string | null`, `kills: number | null`), so a
// null is a value it already models — while a MISSING KEY is not. `omitempty` on those fields
// made the key vanish for exactly the player this endpoint has to represent honestly: one the
// match stats named no team for.
func TestParticipantsRoute_UnknownValuesAreNullNotMissing(t *testing.T) {
	// The Aquarius fixture carries a player with no team and no counters at all.
	w := get(t, newTestServer(t), "/matches/333c4d5e/participants")
	expectStatus(t, w, http.StatusOK, "")

	body := decode[struct {
		Participants []map[string]any `json:"participants"`
	}](t, w)

	var unknown map[string]any
	for _, p := range body.Participants {
		if p["gamertag"] == "Inconnu" {
			unknown = p
		}
	}
	if unknown == nil {
		t.Fatalf("the fixture's team-less player is missing from %v", body.Participants)
	}
	for _, key := range []string{"team_side", "kills", "deaths", "assists"} {
		value, present := unknown[key]
		if !present {
			t.Errorf("key %q dropped instead of published as null: %v", key, unknown)
			continue
		}
		if value != nil {
			t.Errorf("%q = %v, want null", key, value)
		}
	}
}

// TestParticipantsRoute_ServesARecordedButUnbuiltMatch — the roster does not depend on the
// artifact. Participants come from the match stats; the film carries no team information at
// all, so a build that never happened says nothing about who played.
func TestParticipantsRoute_ServesARecordedButUnbuiltMatch(t *testing.T) {
	r := newTestServer(t)

	w := get(t, r, "/matches/"+unbuiltID+"/participants")
	expectStatus(t, w, http.StatusOK, "")
	body := decode[struct {
		Participants []participantRow `json:"participants"`
	}](t, w)
	if len(body.Participants) != 1 || body.Participants[0].Gamertag != "JGtm" {
		t.Errorf("participants = %+v, want the unbuilt match's recorded roster", body.Participants)
	}

	// The replay of that same match is still a 404: there is nothing built to serve.
	expectStatus(t, get(t, r, "/matches/"+unbuiltID+"/replay"), http.StatusNotFound, "match_not_found")
}

// TestRoutes_BusyArchiveIs503 — while a capture holds the archive, every route says "busy"
// rather than "broken". A 503 is retryable in humacore's error contract; a 500 would send the
// operator hunting a fault that is not there.
func TestRoutes_BusyArchiveIs503(t *testing.T) {
	art := newTestArtifacts(t)
	path := filepath.Join(t.TempDir(), "archive.duckdb")
	seedArchive(t, path)
	src, err := newArchiveSource(path)
	if err != nil {
		t.Fatalf("newArchiveSource: %v", err)
	}
	holdArchiveInAnotherProcess(t, path, "rw")

	r := chi.NewRouter()
	(&studyHandler{archive: src, artifacts: art}).mount(r)

	for _, route := range []string{
		"/matches",
		"/matches/000d5950/replay",
		"/matches/000d5950/participants",
	} {
		t.Run(route, func(t *testing.T) {
			expectStatus(t, get(t, r, route), http.StatusServiceUnavailable, "archive_busy")
		})
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
