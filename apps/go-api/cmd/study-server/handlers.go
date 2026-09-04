package main

// handlers.go — THE THREE ROUTES.
//
//	GET /matches                              the archive browser's table
//	GET /matches/{match_id}/replay            the artifact, byte for byte
//	GET /matches/{match_id}/participants      who played, in the scoreboard's shape
//
// Chi + Huma through `internal/api/humacore`, the same pair every migrated route of the app
// uses. That package imports nothing from the project (only huma/humachi/chi and the
// standard library), so a separate binary can adopt it without dragging the app's DI in — and
// adopting it is what makes THIS server's errors byte-identical to the app's:
// `{code, message, retryable}`, generic on 5xx. A hand-rolled JSON error here would be a
// second error contract for one front end to handle.
//
// `{match_id}` ACCEPTS EITHER FORM of the identifier, full or short. The archive holds both,
// the browser links with whichever it was handed, and the resolution happens once, in
// `lookupMatch`.
//
// NO CORS, DELIBERATELY. The study app reaches this server through Vite's dev proxy, the way
// apps/web already reaches the Go API (`vite.config.ts`, `/api` -> the API's own port). Adding
// permissive CORS headers to a server that reads a local archive would widen it for no gain.
//
// NO LOCAL-ONLY MIDDLEWARE EITHER, unlike the app's own replay route: there, a transport
// guard was needed because the route hangs off a server that listens for the whole machine.
// Here the BINDING is the boundary — the listener is on the loopback address by default
// (cf. main.go) — so there is one rule rather than a rule and a guard that can disagree.

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
)

// studyHandler serves the archive: its database, and the artifacts it points at.
//
// It holds the archive's ADDRESS, not an open handle — see archive.go for why that is the one
// structural decision in this server. Every handler therefore goes through `withArchive`.
type studyHandler struct {
	archive   archiveSource
	artifacts artifacts
}

// withArchive borrows the archive for one request and gives it back.
//
// A free function rather than a method because it is generic in the answer, and one seam
// rather than three open/defer pairs because the giving-back is what keeps the hourly capture
// unblocked: a handler that forgot its `Close` would hold the file until the process died, and
// nothing else in the request would look wrong.
func withArchive[T any](ctx context.Context, src archiveSource, fn func(*archive) (T, error)) (T, error) {
	var zero T
	a, err := src.open(ctx)
	if err != nil {
		return zero, err
	}
	defer a.Close()
	return fn(a)
}

// mount registers the three routes on the router.
func (h *studyHandler) mount(r chi.Router) {
	api := humacore.NewAPI(r)
	huma.Get(api, "/matches", h.handleListMatches,
		humacore.Op("listArchivedMatches", "Matchs archivés de l'outil d'étude", "study"))
	huma.Get(api, "/matches/{match_id}/replay", h.handleGetReplay,
		humacore.Op("getArchivedMatchReplay", "Artefact de rejeu d'un match archivé", "study"))
	huma.Get(api, "/matches/{match_id}/participants", h.handleGetParticipants,
		humacore.Op("getArchivedMatchParticipants", "Roster d'un match archivé", "study"))
}

// listMatchesInput is the query string, taken as text and validated by parseFilter.
//
// The date and coverage parameters are STRINGS rather than huma's own `time.Time` and
// `float64`: Huma would reject a bare `2026-05-19` (it wants RFC 3339) and would accept a
// coverage of 85 without a word. The rules are in filter.go, where they are tested.
type listMatchesInput struct {
	Map         string `query:"map" doc:"Nom de carte (insensible à la casse)"`
	Mode        string `query:"mode" doc:"Mode de jeu (insensible à la casse)"`
	Player      string `query:"player" doc:"Gamertag ou xuid d'un participant"`
	From        string `query:"from" doc:"Début inclus : date AAAA-MM-JJ ou instant RFC 3339"`
	To          string `query:"to" doc:"Fin exclue : date AAAA-MM-JJ (le jour entier est inclus) ou instant RFC 3339"`
	MinCoverage string `query:"min_coverage" doc:"Couverture minimale, fraction de 1 (0.85, pas 85)"`
	Limit       int    `query:"limit" doc:"Nombre maximum de lignes (défaut 200, plafond 1000)"`
	Offset      int    `query:"offset" doc:"Rang de la première ligne"`
}

type listMatchesOutput struct{ Body matchPage }

func (h *studyHandler) handleListMatches(ctx context.Context, in *listMatchesInput) (*listMatchesOutput, error) {
	f, err := parseFilter(rawFilter{
		Map: in.Map, Mode: in.Mode, Player: in.Player,
		From: in.From, To: in.To, MinCoverage: in.MinCoverage,
		Limit: in.Limit, Offset: in.Offset,
	})
	if err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_filter", err.Error())
	}
	page, err := withArchive(ctx, h.archive, func(a *archive) (matchPage, error) {
		return a.listMatches(ctx, f)
	})
	if err != nil {
		return nil, archiveError(ctx, err)
	}
	return &listMatchesOutput{Body: page}, nil
}

// matchInput is the path parameter shared by the two single-match routes.
type matchInput struct {
	MatchID string `path:"match_id"`
}

// replayOutput carries the artifact's bytes UNTOUCHED. Huma writes a `[]byte` body straight
// to the wire — no marshalling, no float sanitisation walk over a document that runs to
// megabytes — which is exactly the pass-through artifact.go argues for. The content type is
// declared here because that shortcut also skips content negotiation.
type replayOutput struct {
	ContentType string `header:"Content-Type"`
	Body        []byte
}

func (h *studyHandler) handleGetReplay(ctx context.Context, in *matchInput) (*replayOutput, error) {
	match, err := withArchive(ctx, h.archive, func(a *archive) (matchIdentity, error) {
		return a.lookupBuiltMatch(ctx, in.MatchID)
	})
	if err != nil {
		return nil, archiveError(ctx, err)
	}
	blob, err := h.artifacts.read(match.ShortID)
	if errors.Is(err, errArtifactMissing) {
		// A DIFFERENT answer from "unknown match", and worth the distinction: the archive
		// says this one was built, so the file went missing after the fact and the place to
		// look is the replay cache, not the identifier.
		return nil, humacore.NewError(http.StatusNotFound, "replay_not_available",
			"artefact de rejeu absent du cache pour ce match archivé")
	}
	if err != nil {
		return nil, serverError(ctx, "replay_error", err)
	}
	return &replayOutput{ContentType: "application/json", Body: blob}, nil
}

type participantsOutput struct {
	Body struct {
		Participants []participantRow `json:"participants"`
	}
}

// handleGetParticipants serves the roster of any RECORDED match, built or not — the roster
// comes from the match stats and does not depend on the artifact (cf. lookupRecordedMatch).
func (h *studyHandler) handleGetParticipants(ctx context.Context, in *matchInput) (*participantsOutput, error) {
	rows, err := withArchive(ctx, h.archive, func(a *archive) ([]participantRow, error) {
		match, err := a.lookupRecordedMatch(ctx, in.MatchID)
		if err != nil {
			return nil, err
		}
		return a.readParticipants(ctx, match.MatchID)
	})
	if err != nil {
		return nil, archiveError(ctx, err)
	}
	out := &participantsOutput{}
	out.Body.Participants = rows
	return out, nil
}

// archiveError maps every way reading the archive can fail onto one answer per cause, in one
// place so the three routes cannot answer the same condition differently.
//
//   - an unknown match is a clean 404, never a 500 — the ticket's own criterion;
//   - a busy archive is a 503: a capture holds the file, nothing is broken, and the caller's
//     move is to come back rather than to report a fault. Retryable, which is exactly what
//     humacore's error contract publishes on a 5xx;
//   - anything else is a logged 500.
func archiveError(ctx context.Context, err error) error {
	switch {
	case errors.Is(err, errMatchUnknown):
		return humacore.NewError(http.StatusNotFound, "match_not_found",
			"aucun match archivé sous cet identifiant")
	case errors.Is(err, errArchiveBusy):
		return humacore.NewError(http.StatusServiceUnavailable, "archive_busy",
			"l'archive est momentanément tenue par un autre processus (capture en cours)")
	}
	return serverError(ctx, "archive_error", err)
}

// serverError LOGS the cause, then returns the client's 500.
//
// The log is not optional here. humacore replaces any 5xx message with a generic
// "internal error" so nothing internal reaches the caller — and this binary carries none of
// the app's HTTP logging middleware, so without this line the cause would exist NOWHERE:
// a swallowed error, repo rule 3 and anti-pattern 10, on the one path where the operator has
// nothing else to go on.
func serverError(ctx context.Context, code string, err error) error {
	slog.ErrorContext(ctx, "study-server: "+code, "err", err)
	return humacore.NewError(http.StatusInternalServerError, code, err.Error())
}
