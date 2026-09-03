// Package handlers_test — leaderboard_test.go : contrat HTTP de la page Classement.
//
// Lot 4.1 (chantier « classement mondial : reprise du scrape ») : un GET sans
// saison ni playlist rendait un 500 `leaderboard_error` (l'erreur du repo
// remontait telle quelle). Un paramètre manquant n'est pas une panne serveur :
// le contrat est 200 + corps vide structuré. Test de bout en bout (routeur chi +
// Huma + VRAI LeaderboardService), pas un stub de service : c'est le câblage
// handler↔service qui produisait le 500.
package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service"
)

// strictLeaderboardRepo reproduit le contrat INTERNE du repo DuckDB : le couple
// (saison, playlist) est obligatoire côté lecture, sinon erreur. C'est justement
// cette erreur qui ne doit plus atteindre la couche HTTP.
//
// Its ZERO value doubles as the "repo with no rows at all" fixture: nil entries
// and a zero-value catalog, which is exactly what an empty DuckDB scan produces
// (see TestLeaderboardPage_EmptyCollectionsOnTheWire).
type strictLeaderboardRepo struct {
	entries []domain.LeaderboardEntry
	calls   int
}

func (r *strictLeaderboardRepo) GetLocalLeaderboard(_ context.Context, _, _, _ string) ([]domain.LeaderboardEntry, error) {
	return nil, nil
}

func (r *strictLeaderboardRepo) GetCSRWorldLeaderboard(_ context.Context, _, season, playlist string, _ int) ([]domain.LeaderboardEntry, error) {
	r.calls++
	if season == "" || playlist == "" {
		return nil, errors.New("GetCSRWorldLeaderboard: season et playlist requis")
	}
	return r.entries, nil
}

func (r *strictLeaderboardRepo) GetStatLeaderboard(_ context.Context, _ string, _ domain.LeaderboardCategory, _, _ string, _ int) ([]domain.LeaderboardEntry, error) {
	return nil, nil
}

func (r *strictLeaderboardRepo) GetWorldLeaderboardCatalog(_ context.Context, _ string) (domain.LeaderboardCatalog, error) {
	return domain.LeaderboardCatalog{}, nil
}

// newLeaderboardRouter mounts the leaderboard routes. Optional middlewares are
// installed before the routes (a chi constraint); they exist to reproduce what
// middleware.TitleExtractor does in production, see withTitleSlug.
func newLeaderboardRouter(repo port.LeaderboardRepository, mw ...func(http.Handler) http.Handler) *chi.Mux {
	factory := func(_ context.Context, _ string) (port.LeaderboardService, string, string, error) {
		return service.NewLeaderboardService(repo), testXUID1, testGamertag, nil
	}
	r := chi.NewRouter()
	for _, m := range mw {
		r.Use(m)
	}
	h := handlers.NewLeaderboardHandler(factory)
	r.Route("/players/{player_slug}", func(r chi.Router) {
		h.Mount(r)
	})
	return r
}

// withTitleSlug reproduces what middleware.TitleExtractor injects. Without it,
// ctxkeys.TitleSlug falls back to "halo_infinite" and GetCatalog's "title without
// capability" path — it reads the title from the CONTEXT, not from a query param
// the way GetPage does — stays unreachable from an HTTP test.
func withTitleSlug(slug string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(ctxkeys.WithTitleSlug(r.Context(), slug)))
		})
	}
}

func getLeaderboard(t *testing.T, r *chi.Mux, query string) (int, domain.LeaderboardResponse) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/players/test-player/pages/leaderboard"+query, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var body domain.LeaderboardResponse
	if w.Code == http.StatusOK {
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("corps non décodable (%s): %v", w.Body.String(), err)
		}
	}
	return w.Code, body
}

// TestLeaderboardPage_MissingParams_200Empty : sans saison/playlist → 200 + vide,
// jamais 500. Les 3 combinaisons incomplètes sont couvertes.
func TestLeaderboardPage_MissingParams_200Empty(t *testing.T) {
	cases := []struct{ name, query string }{
		{"aucun paramètre", ""},
		{"saison seule", "?season=csrseason13-3"},
		{"playlist seule", "?playlist=edfef3ac-9cbe-4fa2-b949-8f29deafd483"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &strictLeaderboardRepo{}
			code, body := getLeaderboard(t, newLeaderboardRouter(repo), tc.query)
			if code != http.StatusOK {
				t.Fatalf("status = %d, want 200", code)
			}
			if len(body.Entries) != 0 {
				t.Errorf("entries = %d, want 0", len(body.Entries))
			}
			if body.TotalLocal != 0 {
				t.Errorf("total = %d, want 0", body.TotalLocal)
			}
			if body.Category != string(domain.LeaderboardCSRWorld) {
				t.Errorf("category = %q, want csr-world", body.Category)
			}
			if repo.calls != 0 {
				t.Errorf("repo appelé %d fois alors que le couple est incomplet", repo.calls)
			}
		})
	}
}

// TestLeaderboardPage_CompleteParams_ServesEntries : le couple complet passe bien
// au repo — la garde 4.1 ne coupe pas le chemin nominal.
func TestLeaderboardPage_CompleteParams_ServesEntries(t *testing.T) {
	repo := &strictLeaderboardRepo{entries: []domain.LeaderboardEntry{
		{Rank: 1, Gamertag: "Twissted Mindss", CSRValue: 2180},
	}}
	code, body := getLeaderboard(t, newLeaderboardRouter(repo),
		"?season=csrseason13-3&playlist=edfef3ac-9cbe-4fa2-b949-8f29deafd483")
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200", code)
	}
	if len(body.Entries) != 1 || body.Entries[0].Gamertag != "Twissted Mindss" {
		t.Fatalf("entries inattendues: %+v", body.Entries)
	}
	if body.TotalLocal != 1 {
		t.Errorf("total = %d, want 1", body.TotalLocal)
	}
	if repo.calls != 1 {
		t.Errorf("repo appelé %d fois, want 1", repo.calls)
	}
}

// TestLeaderboardPage_TotalFieldNameOnTheWire (Lot 4.4) : le compteur rempli par
// le service (domain.LeaderboardResponse.TotalLocal) voyage sous le nom `total` —
// il n'y a jamais eu de champ `total_local` sur le fil, le tag JSON est présent et
// sans omitempty. Ce test fige le nom : le contrat OpenAPI et generated.ts en
// dérivent, un renommage silencieux casserait le front sans erreur de compilation.
func TestLeaderboardPage_TotalFieldNameOnTheWire(t *testing.T) {
	repo := &strictLeaderboardRepo{entries: []domain.LeaderboardEntry{{Rank: 1, Gamertag: "A", CSRValue: 1500}}}
	req := httptest.NewRequest(http.MethodGet,
		"/players/test-player/pages/leaderboard?season=csrseason13-3&playlist=pl-a", nil)
	w := httptest.NewRecorder()
	newLeaderboardRouter(repo).ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var raw map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatalf("corps non décodable: %v", err)
	}
	if _, unexpected := raw["total_local"]; unexpected {
		t.Errorf("le corps expose `total_local` — le nom de fil attendu est `total`: %s", w.Body.String())
	}
	total, ok := raw["total"]
	if !ok {
		t.Fatalf("champ `total` absent du corps: %s", w.Body.String())
	}
	if total != float64(1) {
		t.Errorf("total = %v, want 1", total)
	}
}

// TestLeaderboardPage_EmptyCollectionsOnTheWire (Lot 4 finding): `entries`,
// `seasons` and `playlists` are fields WITHOUT omitempty — the contract promises
// the field is present, so `[]` and never `null`. A DuckDB scan with no rows
// returns a NIL Go slice: the service ratchet (TestDTOs_NoNilSlicesOnEmptyInput)
// only exercised GetPage's "incomplete couple" early return, leaving the NOMINAL
// path unguarded (the service assigned the repo's nil slice over its
// construction-time guarantee) along with the whole of GetCatalog.
//
// This test reads the JSON ACTUALLY emitted, not the struct: marshalling is what
// turns a nil into `null`, and that `null` is what reaches the frontend.
func TestLeaderboardPage_EmptyCollectionsOnTheWire(t *testing.T) {
	const pageWithCouple = "/players/test-player/pages/leaderboard?season=csrseason13-3&playlist=pl-a"
	const catalogPath = "/players/test-player/pages/leaderboard/catalog"

	cases := []struct {
		name string
		// ctxTitleSlug: title injected into the CONTEXT (empty = no middleware, so
		// ctxkeys falls back to halo_infinite). Only GetCatalog reads it; GetPage
		// takes its own from the `title_slug` query param.
		ctxTitleSlug string
		path         string
		fields       []string
	}{
		{
			name:   "served page, repo with no rows",
			path:   pageWithCouple,
			fields: []string{"entries"},
		},
		{
			name:   "catalog with no snapshot",
			path:   catalogPath,
			fields: []string{"seasons", "playlists"},
		},
		{
			// The scenario the Lot 4 finding NAMES, exercised on the wire.
			name:   "page, title without capability",
			path:   pageWithCouple + "&title_slug=unknown_title_no_cap",
			fields: []string{"entries"},
		},
		{
			name:         "catalog, title without capability",
			ctxTitleSlug: "unknown_title_no_cap",
			path:         catalogPath,
			fields:       []string{"seasons", "playlists"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Repo with no data: nil entries AND a zero-value catalog (nil slices).
			var mw []func(http.Handler) http.Handler
			if tc.ctxTitleSlug != "" {
				mw = append(mw, withTitleSlug(tc.ctxTitleSlug))
			}
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			w := httptest.NewRecorder()
			newLeaderboardRouter(&strictLeaderboardRepo{}, mw...).ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 (body: %s)", w.Code, w.Body.String())
			}
			var raw map[string]json.RawMessage
			if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
				t.Fatalf("body not decodable: %v", err)
			}
			for _, field := range tc.fields {
				v, ok := raw[field]
				if !ok {
					t.Errorf("field %q missing from body although it has no omitempty: %s", field, w.Body.String())
					continue
				}
				if string(v) != "[]" {
					t.Errorf("%s = %s, want [] (a `null` breaks the typed consumer)", field, v)
				}
			}
		})
	}
}
