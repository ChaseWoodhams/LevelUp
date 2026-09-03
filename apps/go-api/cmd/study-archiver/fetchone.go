package main

// fetchone.go — ARCHIVE ONE MATCH: FILM TO ARTIFACT.
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
	// GetMatchStats returns the raw match-stats payload; the archiver reads the map out
	// of it.
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
	// FrameIntervalMS is the replay grid step; 0 -> replay.DefaultFrameIntervalMS.
	FrameIntervalMS int
}

// outcome is what fetch-one did with one match. It is deliberately a VALUE and not a log
// line: ticket #6 persists these fields as the archive row.
type outcome struct {
	MatchID       string
	ShortID       string
	MapName       string
	ChunksWritten int
	// ArtifactPath is empty when no artifact was written.
	ArtifactPath string
	// SkipReason is empty on success; otherwise one of the named reasons in skip.go.
	SkipReason reason
	Tracks     int
	Points     int
	Shots      int
}

// fetchOne archives a single match: download the whole film into the chunk cache, then
// build the 2D replay artifact from it.
//
// A SKIP IS NOT AN ERROR. An expired film, a map with no bounds, a film that decodes to
// nothing — each comes back as an outcome carrying a named reason. An error is reserved
// for what the archiver cannot interpret: a failed API call, a disk it cannot write, a
// decoder that returned an error.
func fetchOne(ctx context.Context, d deps, matchID string) (outcome, error) {
	out := outcome{MatchID: matchID, ShortID: title.FilmShortMatchID(matchID)}

	// Stats first, and a failure here is an ERROR rather than a skip: the map name has no
	// other source, and a stats call that fails is a transport problem the next run
	// retries — not a statement about this match.
	stats, err := d.Client.GetMatchStats(ctx, matchID)
	if err != nil {
		return out, fmt.Errorf("match stats: %w", err)
	}
	mapInfo, mapErr := resolveMatchMap(stats, d.Catalog)
	out.MapName = mapInfo.Name

	chunks, found, err := d.Client.GetFilmChunks(ctx, matchID)
	if err != nil {
		return out, fmt.Errorf("film download: %w", err)
	}
	if !found {
		return skipped(ctx, out, skipError{
			Reason: skipFilmAbsent,
			Detail: "the film manifest or its blobs answered 404/410",
		}), nil
	}
	if out.ChunksWritten, err = writeFilmChunks(d.Paths, matchID, chunks); err != nil {
		return out, fmt.Errorf("film cache write: %w", err)
	}
	slog.InfoContext(ctx, "study-archiver: film archived",
		"match_id", matchID, "short_id", out.ShortID, "chunks", out.ChunksWritten,
		"dir", d.Paths.FilmChunksDir(matchID), "map", out.MapName)

	if mapErr != nil {
		return skipped(ctx, out, mapErr), nil
	}
	return buildArtifact(ctx, d, out, mapInfo)
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
	out.Tracks, out.Points, out.Shots = len(doc.Tracks), totalPoints(doc), len(doc.Shots)
	slog.InfoContext(ctx, "study-archiver: replay artifact written",
		"match_id", out.MatchID, "short_id", out.ShortID, "map", out.MapName,
		"path", path, "tracks", out.Tracks, "points", out.Points, "shots", out.Shots,
		"frames", doc.FrameCount, "durationMs", doc.DurationMS, "bytes", size)
	return out, nil
}

// skipped stamps the named reason onto the outcome and logs it. Every skip goes through
// here so that none of them can be recorded without also being visible.
func skipped(ctx context.Context, out outcome, err error) outcome {
	var skip skipError
	if !errors.As(err, &skip) {
		// Unreachable today: every caller passes a skipError. Kept because the failure it
		// guards is SILENT — without it an unnamed skip would leave SkipReason empty,
		// which reads as "archived successfully" at every call site and in #6's row.
		skip = skipError{Reason: skipUnknown, Cause: err}
	}
	out.SkipReason = skip.Reason
	slog.WarnContext(ctx, "study-archiver: match skipped - no artifact",
		"match_id", out.MatchID, "short_id", out.ShortID, "reason", skip.Reason,
		"map", out.MapName, "chunks", out.ChunksWritten, "err", err)
	return out
}
