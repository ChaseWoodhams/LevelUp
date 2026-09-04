package main

// rebuild.go — RE-ASSEMBLING AN ARTIFACT FROM CHUNKS ALREADY ON DISK (#10).
//
// This is what makes the archive DURABLE rather than merely current. Every other path in
// this tool races an expiry clock; this one has already won that race for the matches it
// touches. When the decoder improves — or a game patch shifts what it can read — every
// match whose CDN links died months ago is still rebuildable from the bytes captured while
// they were alive.
//
// IT MAKES NO NETWORK CALL, and that is a hard property rather than an incidental one. A
// rebuild that could quietly re-download would, on the day the decoder is fixed, hit a
// dead link for every match worth rebuilding and report failures that say nothing about
// the decoder. So `rebuild` takes no Halo client and needs NO CREDENTIAL: it runs on a
// machine with no tokens at all, and a match whose chunks are missing fails with a message
// that says exactly that.
//
// THE ROW IS UPDATED IN PLACE, not re-recorded. `archive.recorded` is a PARTIAL reader by
// design (#6), so handing its result back to `recordMatch` would blank the columns it does
// not carry — mode, playlist, played-at, source gamertag — and a rebuild is not a re-read
// of the match stats. Only what the BUILD produced is written: the artifact, the counts,
// the decoder revision, the built-at, and the film state those imply.

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/domain/title"
)

// rebuildOne re-assembles one match's artifact from its cached chunks.
func rebuildOne(ctx context.Context, d deps, matchID string) (outcome, error) {
	out := outcome{MatchID: matchID, ShortID: title.FilmShortMatchID(matchID)}
	if d.Archive == nil {
		return out, errors.New("deps.Archive is nil: rebuild cannot run without the archive")
	}
	rec, found, err := d.Archive.recorded(ctx, matchID)
	if err != nil {
		return out, err
	}
	if !found {
		return out, fmt.Errorf("%s is not in the archive: rebuild re-assembles a match "+
			"already captured - run fetch-one first", matchID)
	}
	out.MapName, out.MapModule = rec.MapName, rec.MapModule

	// The chunks, and the refusal that keeps this command honest.
	out.ChunksWritten = cachedFilmChunks(d.Paths, matchID)
	if out.ChunksWritten == 0 {
		return out, fmt.Errorf("no film chunks on disk for %s (%s): rebuild never downloads - "+
			"if the film has not expired, run fetch-one to capture it again",
			matchID, d.Paths.FilmChunksDir(matchID))
	}

	mapInfo, mapErr := resolveMatchMap(rec.MapName, d.Catalog)
	if mapErr != nil {
		// Still not a failure of the decoder: the map's bounds have not arrived yet, and
		// building with another map's would be wrong by an arbitrary scale factor.
		out = skipped(ctx, out, mapErr)
		return out, recordRebuild(ctx, d, out)
	}

	if out, err = buildArtifact(ctx, d, out, mapInfo); err != nil {
		var refused decodeFailure
		if !errors.As(err, &refused) {
			// The disk refused the write: nothing about the match changed, so the row is
			// left exactly as it was.
			return out, err
		}
		out = skipped(ctx, out, skipError{
			Reason: skipBuildFailed,
			Detail: "the cached film was rebuilt but the decoder refused it",
			Cause:  err,
		})
		if recErr := recordRebuild(ctx, d, out); recErr != nil {
			slog.ErrorContext(ctx, "study-archiver: could not record a failed rebuild",
				"err", recErr, "match_id", matchID)
		}
		return out, err
	}
	return out, recordRebuild(ctx, d, out)
}

// recordRebuild writes ONLY what the rebuild changed.
//
// film_state is recomputed from the outcome, which is what moves a match OUT of `failed`
// when a fixed decoder finally reads it — the state is derived from what just happened,
// never patched by hand.
func recordRebuild(ctx context.Context, d deps, out outcome) error {
	var (
		builtAt *time.Time
		rev     string
	)
	if out.ArtifactPath != "" {
		now := time.Now().UTC()
		builtAt, rev = &now, decoderRevision()
	}
	if err := d.Archive.updateBuild(ctx, out, builtAt, rev); err != nil {
		return err
	}
	slog.InfoContext(ctx, "study-archiver: rebuild recorded",
		"match_id", out.MatchID, "film_state", string(filmStateOf(out)),
		"skip_reason", string(out.SkipReason), "artifact", out.ArtifactPath,
		"tracks", out.Tracks, "decoder_rev", rev)
	return nil
}
