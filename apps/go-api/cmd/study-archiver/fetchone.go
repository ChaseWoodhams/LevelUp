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
	// GetMatchHistory lists a player's recent matches, most recent first. Used only by
	// `watch` (#8) for discovery; the id must be in the xuid(N) form the API demands.
	GetMatchHistory(ctx context.Context, xuidForm, matchType string, start, count int) (
		[]haloclient.MatchHistoryEntry, error)
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
	// Settled marks a match a previous run had already reached a FINAL answer on: either
	// its artifact is on disk, or its film is permanently expired. Nothing was fetched,
	// built or written this time. A `failed` or skipped match is NOT settled — those are
	// the ones a later decoder or catalogue rescues.
	Settled bool
}

// fetchOne archives a single match: download the whole film into the chunk cache, build
// the 2D replay artifact from it, and record what happened in the archive.
//
// A SKIP IS NOT AN ERROR. An expired film, a map with no bounds, a film that decodes to
// nothing — each comes back as an outcome carrying a named reason, and each is RECORDED.
// An error is reserved for what the archiver cannot interpret: a failed API call, a disk
// it cannot write. Those are deliberately NOT recorded: a transient failure must not leave
// a terminal state behind (filmstate.go holds the whole policy).
//
// A DECODER THAT ERRORS IS BOTH. It is recorded — `failed`, with its reason, because the
// chunks are on disk and the fault is deterministic — and it is STILL returned as an
// error, so a decoder regression never reads as an ordinary archiving outcome.
func fetchOne(ctx context.Context, d deps, matchID string) (outcome, error) {
	return fetchOneWithStats(ctx, d, matchID, nil)
}

// fetchOneWithStats is fetchOne for a caller that has ALREADY read the match stats.
//
// `watch` (#8) has to: it reads the payload to decide whether the match is the 4v4 the
// archive is for, and re-reading it here would double the stats calls of every unattended
// pass against an API the tool is rate-limited on. A nil payload means "read it yourself",
// which is the ordinary fetch-one path.
func fetchOneWithStats(ctx context.Context, d deps, matchID string, stats map[string]any) (outcome, error) {
	out := outcome{MatchID: matchID, ShortID: title.FilmShortMatchID(matchID)}

	prior, done, err := settledEarlier(ctx, d, &out)
	if err != nil || done {
		return out, err
	}

	// Stats first, and a failure here is an ERROR rather than a skip: the map name has no
	// other source, and a stats call that fails is a transport problem the next run
	// retries — not a statement about this match.
	if stats == nil {
		if stats, err = d.Client.GetMatchStats(ctx, matchID); err != nil {
			return out, fmt.Errorf("match stats: %w", err)
		}
	}
	facts, err := readMatchFacts(stats, d.SourceGamertag)
	if err != nil {
		return out, err
	}
	out.MapName = facts.MapName
	mapInfo, mapErr := resolveMatchMap(facts.MapName, d.Catalog)
	out.MapModule = mapInfo.Module

	if err := downloadFilm(ctx, d, prior, &out); err != nil {
		return out, err
	}
	if out.SkipReason == "" && mapErr != nil {
		out = skipped(ctx, out, mapErr)
	}
	if out.SkipReason == "" {
		var buildErr error
		if out, buildErr = buildArtifact(ctx, d, out, mapInfo); buildErr != nil {
			var refused decodeFailure
			if !errors.As(buildErr, &refused) {
				// The decoder produced a document and the DISK refused it. Nothing about
				// this match is settled, so nothing is recorded: the next run retries.
				return out, buildErr
			}
			return recordBuildFailure(ctx, d, out, facts, buildErr)
		}
	}
	return out, recordOutcome(ctx, d, out, facts)
}

// recordBuildFailure records a refused decode as `failed` and hands the decoder's error
// back unchanged.
//
// WHY RECORD AT ALL, when every other error leaves no row. Because this one is not
// transient: the chunks are on disk, the inputs are fixed, and the next run would decode
// the same bytes into the same failure. Recording it is what lets #9 report "these matches
// need a decoder fix" instead of "these matches keep erroring", and the state chosen —
// `failed`, never `expired` — is what keeps the match eligible for the rebuild that fixes
// it.
//
// A failure to record is logged and dropped: the caller must see the DECODER's error, not
// a database error raised while writing it down.
func recordBuildFailure(ctx context.Context, d deps, out outcome, facts matchFacts, buildErr error) (outcome, error) {
	out = skipped(ctx, out, skipError{
		Reason: skipBuildFailed,
		Detail: "the film downloaded but the decoder refused it",
		Cause:  buildErr,
	})
	if err := recordOutcome(ctx, d, out, facts); err != nil {
		slog.ErrorContext(ctx, "study-archiver: could not record a failed build - the match "+
			"will be re-attempted from scratch", "err", err, "match_id", out.MatchID)
	}
	return out, buildErr
}

// settledEarlier short-circuits a match a previous run already reached a FINAL answer on:
// no stats call, no re-download, no rebuild, no second row.
//
// TWO WAYS TO BE FINAL, and they are the two ends of the epic. Either the artifact is
// built and on disk — there is nothing left to do — or the film is `expired` and there
// never will be. Everything else is retried: a `downloaded` match with no artifact is one
// the quant-bounds catalogue may make buildable at any time, and a `failed` one is waiting
// on a decoder fix. That is why the artifact test is not simply "is there a row".
//
// A recorded artifact that has gone missing from disk is not an archive, so it is rebuilt
// rather than trusted.
//
// It also RETURNS the row when it does not short-circuit, because a match that is not
// settled may still be half done — the film captured, the artifact not built — and the
// rest of the run needs to know that to avoid re-downloading it.
func settledEarlier(ctx context.Context, d deps, out *outcome) (matchRecord, bool, error) {
	if d.Archive == nil {
		// deps.Archive is documented as required. A wiring that forgets it (#8 assembling
		// its own deps, say) must be told so, not panic three frames deeper.
		return matchRecord{}, false,
			errors.New("deps.Archive is nil: the archiver cannot run without its database")
	}
	rec, found, err := d.Archive.recorded(ctx, out.MatchID)
	if err != nil || !found {
		return matchRecord{}, false, err
	}
	if rec.State.terminal() {
		reportExpired(ctx, rec, out)
		return rec, true, nil
	}
	if !artifactOnDisk(rec) {
		if rec.ArtifactPath != "" {
			slog.WarnContext(ctx, "study-archiver: recorded artifact missing from disk - re-archiving",
				"match_id", out.MatchID, "path", rec.ArtifactPath)
		}
		return rec, false, nil
	}
	out.MapName, out.MapModule = rec.MapName, rec.MapModule
	out.ArtifactPath = rec.ArtifactPath
	out.Tracks, out.Points, out.Shots = rec.Tracks, rec.Points, rec.Shots
	out.NamedLives, out.TotalLives = rec.NamedLives, rec.TotalLives
	out.Settled = true
	slog.InfoContext(ctx, "study-archiver: already archived - nothing to do",
		"match_id", out.MatchID, "short_id", out.ShortID, "map", out.MapName,
		"path", out.ArtifactPath, "tracks", out.Tracks)
	return rec, true, nil
}

// artifactOnDisk reports whether a recorded artifact is actually there.
//
// A RECORDED PATH IS NOT AN ARCHIVE. The cache is a directory an operator can empty, and
// a row pointing at a file that is gone must send the match back through the build rather
// than pass for finished. Shared by fetch-one and by `watch`'s discovery filter (#8) so the
// two cannot drift into disagreeing about what "already archived" means — an earlier
// version had watch skipping matches that fetch-one would have rebuilt.
func artifactOnDisk(rec matchRecord) bool {
	if rec.ArtifactPath == "" {
		return false
	}
	_, err := os.Stat(rec.ArtifactPath)
	return err == nil
}

// reportExpired fills in the outcome of a match whose film a previous run found gone.
//
// The row is NOT rewritten. There is nothing new to say about it, and rewriting would
// move recorded_at forward every hour, making a match settled months ago look freshly
// examined to anything reading the archive.
func reportExpired(ctx context.Context, rec matchRecord, out *outcome) {
	out.MapName, out.MapModule = rec.MapName, rec.MapModule
	out.SkipReason = rec.SkipReason
	if out.SkipReason == "" {
		// An expired row with no reason predates nothing and should not exist, but reading
		// it as "archived successfully" is the one interpretation that would be dangerous.
		out.SkipReason = skipFilmAbsent
	}
	out.Settled = true
	slog.InfoContext(ctx, "study-archiver: film expired in an earlier run - not re-attempted",
		"match_id", out.MatchID, "short_id", out.ShortID, "map", out.MapName,
		"reason", string(out.SkipReason))
}

// downloadFilm fetches the whole film and writes it into the chunk cache. An absent film
// stamps the named skip onto the outcome rather than returning an error.
//
// A FILM ALREADY CAPTURED IS NEVER FETCHED AGAIN, and that is not merely an optimisation.
// A match recorded `failed` is waiting for a decoder fix that may be months away, by which
// time its CDN link is certainly dead — and a rebuild that went back to the CDN would get
// a 404 and record `expired`, burying, permanently, a match whose film is sitting on disk
// intact. Rebuilding from the cache is what makes the `failed` state mean what it says.
//
// THE TWO SHAPES OF EXPIRY, AND WHY BOTH LAND HERE. Halo serves the film manifest from
// its own store and the chunks from pre-signed CDN blobs, and the two die on separate
// schedules. A dead manifest comes back as `found=false`; a dead blob comes back as an
// ERROR, because the client cannot know that its caller reads 404 as a verdict rather than
// a fault. Both mean the same thing — the bytes will never exist again — so both record
// `expired`. Reading the second as a transport fault is what would have the hourly run
// re-download a dead link forever.
func downloadFilm(ctx context.Context, d deps, prior matchRecord, out *outcome) error {
	if prior.State.filmCaptured() {
		if cached := cachedFilmChunks(d.Paths, out.MatchID); cached > 0 {
			out.ChunksWritten = cached
			slog.InfoContext(ctx, "study-archiver: film already captured - rebuilding from the chunk cache",
				"match_id", out.MatchID, "short_id", out.ShortID, "chunks", cached,
				"dir", d.Paths.FilmChunksDir(out.MatchID), "prior_state", string(prior.State))
			return nil
		}
		// The row says captured and the disk disagrees: the cache was emptied. Fall through
		// and re-fetch — which may well find the film expired by now, and that verdict is
		// then correct, since neither copy of the bytes exists any more.
		slog.WarnContext(ctx, "study-archiver: recorded film missing from the chunk cache - re-downloading",
			"match_id", out.MatchID, "dir", d.Paths.FilmChunksDir(out.MatchID),
			"prior_state", string(prior.State))
	}
	chunks, found, err := d.Client.GetFilmChunks(ctx, out.MatchID)
	if err != nil && !haloclient.IsFilmGoneErr(err) {
		// Everything else — 5xx, timeouts, a dropped connection — is transient, and stays
		// an error precisely so that NO state is recorded and the next run starts over.
		return fmt.Errorf("film download: %w", err)
	}
	if err != nil || !found {
		*out = skipped(ctx, *out, skipError{
			Reason: skipFilmAbsent,
			Detail: "the film manifest or its blobs answered 404/410",
			Cause:  err,
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
//
// Its two failures are different facts, and the caller has to tell them apart: the
// decoder's refusal is wrapped in a decodeFailure and gets recorded, the disk's is not.
func buildArtifact(ctx context.Context, d deps, out outcome, mapInfo matchMap) (outcome, error) {
	filmDir := d.Paths.FilmChunksDir(out.MatchID)
	doc, err := runBuild(d.Build, out.MatchID, d.Title, filmDir, d.buildOptions(ctx, mapInfo))
	if err != nil {
		return out, decodeFailure{fmt.Errorf("replay build: %w", err)}
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
		Team0Score: facts.Team0Score, Team1Score: facts.Team1Score,
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
