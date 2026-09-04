package main

// archive.go — READING THE ARCHIVE, AND ONLY READING IT.
//
// The archiver (cmd/study-archiver) declares the archive's shape and is its only writer.
// This server never creates a table, never writes a row, and holds no lease: it reads through
// `duckdb.OpenReadForQuery`, the repo's helper for reading a database somebody else may be
// holding read-write (CLAUDE.md ART rule 4, ADR 0013/0016).
//
// THE HANDLE IS HELD FOR ONE REQUEST, NOT FOR THE LIFE OF THE SERVER, and that is the whole
// design of this file. It was first written the obvious way — open at startup, keep it — with
// a comment claiming that "across processes OpenReadForQuery opens READ_ONLY beside the
// archiver's writer". THAT CLAIM IS FALSE, and a cross-process test now proves it
// (crossprocess_test.go). DuckDB is single-instance-per-file ACROSS PROCESSES — the repo's own
// `docs/RUNBOOK_OPS_DUCKDB_CLI_TOOLS.md` says so, and "aucun lock applicatif Go ne peut
// résoudre ce conflit cross-process" — so the lock is mutual and it bites in BOTH directions:
//
//   - a server started during a capture could not open the archive at all, and exited 1;
//   - worse, a server left running took the file and THE NEXT HOURLY CAPTURE COULD NOT WRITE.
//
// The second one is the one that matters. This tool exists to beat an expiry clock; a study
// server that quietly blocked the archiver would cost films, and films do not come back. So
// the server yields by construction: it holds the archive for the milliseconds of a query and
// gives it straight back, which is a window the hourly pass can essentially always win.
//
// WHAT REMAINS, HONESTLY STATED: the archiver holds the archive open for its WHOLE pass, so
// while a capture runs this server cannot read at all. That is DuckDB's model, not a bug here,
// and it degrades to a clean 503 saying so rather than to a 500 or a hang. Narrowing the
// archiver's own hold is the fix, and it belongs to the archiver.
//
// EVERY IDENTIFIER THE SERVER PUTS ON DISK COMES BACK OUT OF THIS FILE. A path is only ever
// built from a `short_id` read off a row (cf. artifact.go), never from the URL, so the
// artifact route cannot be talked into resolving a path of the caller's choosing.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	ddb "levelup/go-api/internal/platform/duckdb"
)

// errNoArchive is "nothing has been captured on this machine yet" — an operator state, not a
// failure of this run, and the first thing a new user sees.
var errNoArchive = errors.New("no study archive on this machine")

// errArchiveBusy is "another process holds the archive" — in practice, a capture pass. It is
// a 503 and not a 500: nothing is broken, the answer is simply not available this second.
var errArchiveBusy = errors.New("the archive is held by another process")

// errMatchUnknown is "this archive holds no match under that identifier".
var errMatchUnknown = errors.New("unknown match")

// Retry budget for the open. Short on purpose: it exists to ride out the overlap of two
// requests or the archiver's own brief moments, NOT to wait out a capture — a pass runs for
// minutes, and a request that hung for minutes would be worse than one that says "busy".
const (
	archiveOpenAttempts = 5
	archiveOpenBackoff  = 40 * time.Millisecond
)

// archiveSource is the archive as an ADDRESS, not an open handle. Holding the address rather
// than the file is what lets the server exist alongside the archiver at all.
type archiveSource struct{ path string }

// newArchiveSource checks that an archive exists, WITHOUT opening it.
//
// Not opening is the point: a startup that took the file would be the very thing this design
// avoids, and a server started mid-capture would fail to boot for no good reason. A stat is
// enough to give the operator the one message they need on a fresh machine.
func newArchiveSource(path string) (archiveSource, error) {
	// Checked here because DuckDB's own failure on an absent file names the driver and the
	// path and nothing an operator can act on, and this is the state every machine is in
	// until the first capture runs.
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return archiveSource{}, fmt.Errorf("%w at %s: run `study-archiver watch` or `fetch-one` to create it",
			errNoArchive, path)
	}
	return archiveSource{path: path}, nil
}

// archive is an open read handle on the archive database, for the length of one request.
type archive struct {
	db      *sql.DB
	release func()
	path    string
}

// open borrows the archive, retrying briefly if another process holds it.
func (s archiveSource) open(ctx context.Context) (*archive, error) {
	var err error
	for attempt := range archiveOpenAttempts {
		var (
			db      *sql.DB
			release func()
		)
		if db, release, err = ddb.OpenReadForQuery(s.path); err == nil {
			return &archive{db: db, release: release, path: s.path}, nil
		}
		// Only a LOCK is worth retrying. A corrupt or unreadable file will not heal in
		// 200 ms, and retrying it would replace a precise error with a vague one.
		if !isLocked(err) {
			return nil, fmt.Errorf("opening the archive %s for reading: %w", s.path, err)
		}
		if attempt < archiveOpenAttempts-1 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(archiveOpenBackoff):
			}
		}
	}
	slog.WarnContext(ctx, "study-server: the archive is held by another process - a capture is "+
		"probably running", "path", s.path, "err", err)
	return nil, fmt.Errorf("%w: %s", errArchiveBusy, s.path)
}

// isLocked recognises DuckDB's single-instance-per-file refusal.
//
// BY MESSAGE, because the driver offers nothing else: the failure arrives as a generic
// `database/sql/driver` connect error wrapping DuckDB's own IO error text. Both platform
// wordings are matched — Windows names the sharing violation, POSIX names the lock — and an
// unrecognised error is deliberately NOT treated as a lock, so a real fault stays a real fault
// rather than being reported to the operator as "busy, try later".
func isLocked(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	for _, sign := range []string{
		"being used by another process", // Windows sharing violation
		"could not set lock",            // POSIX flock
		"file is already open",          // DuckDB's own wording, both platforms
	} {
		if strings.Contains(msg, sign) {
			return true
		}
	}
	return false
}

// Close gives the archive back. Called at the end of every request, and the reason the
// archiver can take the file whenever it needs it.
func (a *archive) Close() {
	if a != nil && a.release != nil {
		a.release()
	}
}

// matchSummary is one row of the archive browser: what was played, and what the decoder got
// out of it. Everything here is a fact the archiver recorded.
type matchSummary struct {
	MatchID   string     `json:"match_id"`
	ShortID   string     `json:"short_id"`
	PlayedAt  *time.Time `json:"played_at,omitempty"`
	MapName   string     `json:"map_name,omitempty"`
	MapModule string     `json:"map_module,omitempty"`
	Mode      string     `json:"mode,omitempty"`
	Playlist  string     `json:"playlist,omitempty"`
	// DurationMS is milliseconds, the replay document's own clock, so the two can be
	// compared without a unit change.
	DurationMS *int64 `json:"duration_ms,omitempty"`
	// SourceGamertag is whose archiving pass DISCOVERED the match. It is NOT "whose match
	// this is": when two tracked players meet, the first pass takes credit for the game.
	// Filter by player instead; this field is provenance, not ownership.
	SourceGamertag string     `json:"source_gamertag,omitempty"`
	BuiltAt        *time.Time `json:"built_at,omitempty"`
	// Coverage is named lives over total lives, 0..1 — ABSENT when the artifact reported no
	// lives at all, which is "unknown", not "zero" (cf. coverageRatio).
	Coverage   *float64 `json:"coverage,omitempty"`
	NamedLives int      `json:"named_lives"`
	TotalLives int      `json:"total_lives"`
	Tracks     int      `json:"tracks"`
	Points     int      `json:"points"`
	Shots      int      `json:"shots"`
}

// matchPage is one page of the browser, and how many rows the filter matched in total —
// the second number being the one a pager needs and the first cannot give.
type matchPage struct {
	Matches []matchSummary `json:"matches"`
	Total   int            `json:"total"`
}

// matchIdentity is a servable match: both forms of its identifier, read off the row.
type matchIdentity struct {
	MatchID string
	ShortID string
}

// listMatches runs a filter and returns one page of archived matches, newest first.
func (a *archive) listMatches(ctx context.Context, f matchFilter) (matchPage, error) {
	where, args := f.where()

	page := matchPage{Matches: []matchSummary{}}
	if err := a.db.QueryRowContext(ctx,
		`SELECT count(*) FROM matches m WHERE `+where, args...).Scan(&page.Total); err != nil {
		return matchPage{}, fmt.Errorf("counting the archived matches: %w", err)
	}

	// NULLS LAST so a match whose stats never named a start time sinks to the bottom
	// rather than heading a list sorted by "most recent". match_id breaks ties, so two
	// matches played in the same second keep a stable order across pages.
	rows, err := a.db.QueryContext(ctx, `
        SELECT m.match_id, m.short_id, m.played_at, m.map_name, m.map_module, m.mode,
               m.playlist, m.duration_ms, m.source_gamertag, m.built_at,
               m.tracks, m.points, m.shots, m.named_lives, m.total_lives,
               `+coverageRatio+` AS coverage
        FROM matches m
        WHERE `+where+`
        ORDER BY m.played_at DESC NULLS LAST, m.match_id
        LIMIT ? OFFSET ?`, append(args, f.Limit, f.Offset)...)
	if err != nil {
		return matchPage{}, fmt.Errorf("listing the archived matches: %w", err)
	}
	defer closeRows(ctx, rows, "archived matches")

	for rows.Next() {
		m, err := scanMatch(rows)
		if err != nil {
			return matchPage{}, err
		}
		page.Matches = append(page.Matches, m)
	}
	if err := rows.Err(); err != nil {
		return matchPage{}, fmt.Errorf("reading the archived matches: %w", err)
	}
	return page, nil
}

// scanMatch reads one row. Every text column is nullable in the archive — a match whose
// stats named no map really does store NULL — so all of them go through sql.NullString
// rather than crashing the scan on exactly the degraded row worth looking at.
func scanMatch(rows *sql.Rows) (matchSummary, error) {
	var (
		m                                  matchSummary
		mapName, mapModule, mode, playlist sql.NullString
		sourceGT                           sql.NullString
		playedAt, builtAt                  sql.NullTime
		durationMS                         sql.NullInt64
		coverage                           sql.NullFloat64
	)
	if err := rows.Scan(&m.MatchID, &m.ShortID, &playedAt, &mapName, &mapModule, &mode,
		&playlist, &durationMS, &sourceGT, &builtAt,
		&m.Tracks, &m.Points, &m.Shots, &m.NamedLives, &m.TotalLives, &coverage); err != nil {
		return matchSummary{}, fmt.Errorf("scanning an archived match: %w", err)
	}
	m.MapName, m.MapModule = mapName.String, mapModule.String
	m.Mode, m.Playlist, m.SourceGamertag = mode.String, playlist.String, sourceGT.String
	m.PlayedAt, m.BuiltAt = nullTime(playedAt), nullTime(builtAt)
	if durationMS.Valid {
		ms := durationMS.Int64
		m.DurationMS = &ms
	}
	if coverage.Valid {
		c := coverage.Float64
		m.Coverage = &c
	}
	return m, nil
}

// lookupBuiltMatch resolves an identifier to a match that HAS AN ARTIFACT — what /replay
// needs, since a match with nothing built has nothing to serve.
//
// A match recorded without an artifact answers errMatchUnknown here, deliberately: to a caller
// asking for a replay, "the archiver saw this match but could never build it" and "no such
// match" lead to the same place, and the archive's own `status` command is where the
// difference is reported.
func (a *archive) lookupBuiltMatch(ctx context.Context, id string) (matchIdentity, error) {
	return a.lookup(ctx, id, "AND artifact_path IS NOT NULL")
}

// lookupRecordedMatch resolves an identifier to ANY match the archive recorded, built or not.
//
// WHY THE ROSTER DOES NOT NEED AN ARTIFACT. Participants come from the MATCH STATS, never from
// the film (the film carries no team information at all). Their existence therefore has
// nothing to do with whether the decoder managed to build a replay: a match whose map had no
// quant bounds yet still has eight players, their teams and their K/D/A, all correctly
// recorded. Refusing to serve them because a DIFFERENT layer failed would be an accident of
// implementation, not a fact about the match.
func (a *archive) lookupRecordedMatch(ctx context.Context, id string) (matchIdentity, error) {
	return a.lookup(ctx, id, "")
}

// lookup resolves either form of a match identifier, deterministically.
//
// THE ORDERING IS NOT DECORATION. `match_id` is the table's only declared key; `short_id` is
// the first 8 hex characters of it and nothing stops two matches from sharing one. Without an
// order, a collision would resolve to whichever row the engine happened to hand back — a
// different replay on different days. An exact full-id hit therefore always wins, and among
// short-id hits the lowest id wins, every time.
func (a *archive) lookup(ctx context.Context, id, extra string) (matchIdentity, error) {
	var got matchIdentity
	err := a.db.QueryRowContext(ctx, `
        SELECT match_id, short_id FROM matches
        WHERE (match_id = ? OR short_id = ?) `+extra+`
        ORDER BY CASE WHEN match_id = ? THEN 0 ELSE 1 END, match_id
        LIMIT 1`, id, id, id).
		Scan(&got.MatchID, &got.ShortID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return matchIdentity{}, fmt.Errorf("%w: %s", errMatchUnknown, id)
	case err != nil:
		return matchIdentity{}, fmt.Errorf("looking up match %s: %w", id, err)
	}
	return got, nil
}

// closeRows logs a cursor that refuses to close rather than dropping it (repo rule 3: no
// error goes by in silence, even one the caller can do nothing about).
func closeRows(ctx context.Context, rows *sql.Rows, what string) {
	if err := rows.Close(); err != nil {
		slog.ErrorContext(ctx, "study-server: closing a cursor", "err", err, "query", what)
	}
}

func nullTime(t sql.NullTime) *time.Time {
	if !t.Valid {
		return nil
	}
	v := t.Time
	return &v
}
