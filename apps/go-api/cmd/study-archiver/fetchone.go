package main

// fetchone.go — ARCHIVE ONE MATCH: FILM TO ARTIFACT TO ARCHIVE ROW.
//
// ORDER OF OPERATIONS, AND WHY. The film is downloaded BEFORE the map is judged
// supported. It costs bandwidth on a match that cannot be built today, and it is the
// point of the whole tool: the CDN link expires in weeks, the quant-bounds catalogue
// grows whenever cmd/mapquant-build is run on a new map. Downloading first turns "this
// map has no bounds yet" into a rebuild (#10) instead of a permanent loss.
//
// The map IS resolved first, from the stats call, because the reason belongs in the log
// line of the download too — but nothing branches on it until the chunks are on disk.

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/sync/haloclient"
)

// filmAPI is the slice of the Halo client the archiver uses. Narrow on purpose: it says
// what fetch-one is allowed to reach for, and it is what the tests substitute.
type filmAPI interface {
	// GetFilmChunks downloads EVERY chunk type of a match's film (header, replication,
	// highlight) in one manifest read. Reports found=false on 404/410 — an absent film
	// is a normal outcome, not an error.
	GetFilmChunks(ctx context.Context, matchID string) ([]haloclient.FilmChunk, bool, error)
	// GetMatchStats returns the raw match-stats payload; the archiver reads the map,
	// the mode, the playlist and the whole roster out of it.
	GetMatchStats(ctx context.Context, matchID string) (map[string]any, error)
}

// deps is everything fetch-one needs, already resolved. The catalogues arrive LOADED
// rather than as paths so the orchestration can be tested without a repo checkout — and
// so a `watch` loop (#8) loads them once instead of once per match.
type deps struct {
	Client  filmAPI
	Paths   *title.PathResolver
	Title   string
	Catalog *filmdec.MapQuantCatalog
	Labels  replay.LabelCatalog
	Build   buildFilm
	// Archive is the archive database, and it is required: a match archived without a
	// row is a match the tool has forgotten it has.
	Archive *archive
	// SourceGamertag is whose archiving pass produced this match. Empty for a bare
	// fetch-one; `watch` (#8) fills it with the watchlist entry that surfaced the match.
	SourceGamertag string
	// FrameIntervalMS is the replay grid step; 0 -> replay.DefaultFrameIntervalMS.
	FrameIntervalMS int
}

// outcome is what fetch-one did with one match. It is deliberately a VALUE and not a log
// line: it is what the archive row is built from.
type outcome struct {
	MatchID       string
	ShortID       string
	MapName       string
	MapModule     string
	ChunksWritten int
	// ArtifactPath is empty when no artifact was written.
	ArtifactPath string
	// SkipReason is empty on success; otherwise one of the named reasons in skip.go.
	SkipReason reason
	Tracks     int
	Points     int
	Shots      int
	NamedLives int
	TotalLives int
	// AlreadyArchived marks a match a previous run had already finished with: nothing
	// was fetched, built or written this time.
	AlreadyArchived bool
}

// fetchOne archives a single match: download the whole film into the chunk cache, build
// the 2D replay artifact from it, and record what happened in the archive.
//
// A SKIP IS NOT AN ERROR. An expired film, a map with no bounds, a film that decodes to
// nothing — each comes back as an outcome carrying a named reason, and each is RECORDED.
// An error is reserved for what the archiver cannot interpret: a failed API call, a disk
// it cannot write, a decoder that returned an error. Those are deliberately NOT recorded:
// a transient failure must not leave a terminal state behind (#7 owns that distinction).
func fetchOne(ctx context.Context, d deps, matchID string) (outcome, error) {
	out := outcome{MatchID: matchID, ShortID: title.FilmShortMatchID(matchID)}

	done, err := alreadyArchived(ctx, d, &out)
	if err != nil || done {
		return out, err
	}

	// Stats first, and a failure here is an ERROR rather than a skip: the map name has no
	// other source, and a stats call that fails is a transport problem the next run
	// retries — not a statement about this match.
	stats, err := d.Client.GetMatchStats(ctx, matchID)
	if err != nil {
		return out, fmt.Errorf("match stats: %w", err)
	}
	facts, err := readMatchFacts(stats, d.SourceGamertag)
	if err != nil {
		return out, err
	}
	out.MapName = facts.MapName
	mapInfo, mapErr := resolveMatchMap(facts.MapName, d.Catalog)
	out.MapModule = mapInfo.Module

	if err := downloadFilm(ctx, d, &out); err != nil {
		return out, err
	}
	if out.SkipReason == "" && mapErr != nil {
		out = skipped(ctx, out, mapErr)
	}
	if out.SkipReason == "" {
		if out, err = buildArtifact(ctx, d, out, mapInfo); err != nil {
			return out, err
		}
	}
	return out, recordOutcome(ctx, d, out, facts)
}

// alreadyArchived short-circuits a match a previous run already finished with: no
// re-download, no rebuild, no second row.
//
// The test is the ARTIFACT, not the film state. A match can be `downloaded` and still
// have no artifact — an unsupported map, whose catalogue entry may have arrived since —
// and that one MUST be retried. And a recorded artifact that has gone missing from disk
// is not an archive, so it is rebuilt rather than trusted.
func alreadyArchived(ctx context.Context, d deps, out *outcome) (bool, error) {
	if d.Archive == nil {
		// deps.Archive is documented as required. A wiring that forgets it (#8 assembling
		// its own deps, say) must be told so, not panic three frames deeper.
		return false, errors.New("deps.Archive is nil: the archiver cannot run without its database")
	}
	rec, found, err := d.Archive.recorded(ctx, out.MatchID)
	if err != nil || !found || rec.ArtifactPath == "" {
		return false, err
	}
	if _, statErr := os.Stat(rec.ArtifactPath); statErr != nil {
		slog.WarnContext(ctx, "study-archiver: recorded artifact missing from disk - re-archiving",
			"match_id", out.MatchID, "path", rec.ArtifactPath, "err", statErr)
		return false, nil
	}
	out.MapName, out.MapModule = rec.MapName, rec.MapModule
	out.ArtifactPath = rec.ArtifactPath
	out.Tracks, out.Points, out.Shots = rec.Tracks, rec.Points, rec.Shots
	out.NamedLives, out.TotalLives = rec.NamedLives, rec.TotalLives
	out.AlreadyArchived = true
	slog.InfoContext(ctx, "study-archiver: already archived - nothing to do",
		"match_id", out.MatchID, "short_id", out.ShortID, "map", out.MapName,
		"path", out.ArtifactPath, "tracks", out.Tracks)
	return true, nil
}

// downloadFilm fetches the whole film and writes it into the chunk cache. An absent film
// stamps the named skip onto the outcome rather than returning an error.
func downloadFilm(ctx context.Context, d deps, out *outcome) error {
	chunks, found, err := d.Client.GetFilmChunks(ctx, out.MatchID)
	if err != nil {
		return fmt.Errorf("film download: %w", err)
	}
	if !found {
		*out = skipped(ctx, *out, skipError{
			Reason: skipFilmAbsent,
			Detail: "the film manifest or its blobs answered 404/410",
		})
		return nil
	}
	if out.ChunksWritten, err = writeFilmChunks(d.Paths, out.MatchID, chunks); err != nil {
		return fmt.Errorf("film cache write: %w", err)
	}
	slog.InfoContext(ctx, "study-archiver: film archived",
		"match_id", out.MatchID, "short_id", out.ShortID, "chunks", out.ChunksWritten,
		"dir", d.Paths.FilmChunksDir(out.MatchID), "map", out.MapName)
	return nil
}

// buildArtifact runs the (serialised) decode and writes the replay artifact.
func buildArtifact(ctx context.Context, d deps, out outcome, mapInfo matchMap) (outcome, error) {
	filmDir := d.Paths.FilmChunksDir(out.MatchID)
	doc, err := runBuild(d.Build, out.MatchID, d.Title, filmDir, d.buildOptions(ctx, mapInfo))
	if err != nil {
		return out, fmt.Errorf("replay build: %w", err)
	}
	if len(doc.Tracks) == 0 {
		return skipped(ctx, out, skipError{
			Reason: skipNoTracks,
			Detail: "the film decoded but yielded no trajectory",
		}), nil
	}

	path := d.Paths.ReplayArtifactPath(d.Title, out.MatchID)
	size, err := writeArtifact(path, doc)
	if err != nil {
		return out, err
	}
	out.ArtifactPath = path
	out.Tracks, out.Points, out.Shots, out.NamedLives, out.TotalLives = countsOf(doc)
	slog.InfoContext(ctx, "study-archiver: replay artifact written",
		"match_id", out.MatchID, "short_id", out.ShortID, "map", out.MapName,
		"path", path, "tracks", out.Tracks, "points", out.Points, "shots", out.Shots,
		"named_lives", out.NamedLives, "total_lives", out.TotalLives,
		"frames", doc.FrameCount, "durationMs", doc.DurationMS, "bytes", size)
	return out, nil
}

// recordOutcome writes the archive row for whatever just happened.
func recordOutcome(ctx context.Context, d deps, out outcome, facts matchFacts) error {
	rec := matchRecord{
		MatchID: out.MatchID, ShortID: out.ShortID,
		PlayedAt: facts.PlayedAt, MapName: out.MapName, MapModule: out.MapModule,
		Mode: facts.Mode, Playlist: facts.Playlist, DurationMS: facts.DurationMS,
		SourceGT: d.SourceGamertag,
		State:    filmStateOf(out), SkipReason: out.SkipReason,
		ArtifactPath: out.ArtifactPath,
		Tracks:       out.Tracks, Points: out.Points, Shots: out.Shots,
		NamedLives: out.NamedLives, TotalLives: out.TotalLives,
	}
	if out.ArtifactPath != "" {
		built := time.Now().UTC()
		rec.BuiltAt = &built
		rec.DecoderRev = decoderRevision()
		if rec.DecoderRev == "" {
			// Not an error, but the operator has to know: an artifact recorded without a
			// revision cannot be traced to the build that produced it, which is the whole
			// reason the column exists. `go run` never stamps one.
			slog.WarnContext(ctx, "study-archiver: artifact recorded without a decoder revision "+
				"- build the binary (go build) instead of `go run` to make builds traceable",
				"match_id", out.MatchID)
		}
	}
	if err := d.Archive.recordMatch(ctx, rec, facts.Roster); err != nil {
		return err
	}
	slog.InfoContext(ctx, "study-archiver: archive row recorded",
		"match_id", out.MatchID, "film_state", string(rec.State),
		"skip_reason", string(rec.SkipReason), "participants", len(facts.Roster),
		"decoder_rev", rec.DecoderRev)
	return nil
}

// filmStateOf maps an outcome onto the film's life-cycle state. The three terminal
// answers mean different things to a later run, which is why they are not one flag:
// `expired` will never succeed, `failed` may succeed after a decoder fix, `downloaded`
// with a skip reason may succeed once the map catalogue grows.
func filmStateOf(out outcome) filmState {
	switch out.SkipReason {
	case skipFilmAbsent:
		return stateExpired
	case skipNoTracks:
		return stateFailed
	}
	if out.ChunksWritten > 0 {
		return stateDownloaded
	}
	return statePending
}

// skipped stamps the named reason onto the outcome and logs it. Every skip goes through
// here so that none of them can be recorded without also being visible.
func skipped(ctx context.Context, out outcome, err error) outcome {
	var skip skipError
	if !errors.As(err, &skip) {
		// Unreachable today: every caller passes a skipError. Kept because the failure it
		// guards is SILENT — without it an unnamed skip would leave SkipReason empty,
		// which reads as "archived successfully" at every call site and in the archive row.
		skip = skipError{Reason: skipUnknown, Cause: err}
	}
	out.SkipReason = skip.Reason
	slog.WarnContext(ctx, "study-archiver: match skipped - no artifact",
		"match_id", out.MatchID, "short_id", out.ShortID, "reason", string(skip.Reason),
		"map", out.MapName, "chunks", out.ChunksWritten, "err", err)
	return out
}
