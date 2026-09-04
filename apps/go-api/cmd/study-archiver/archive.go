package main

// archive.go — THE ARCHIVE'S MEMORY.
//
// A DuckDB file at data/study/archive.duckdb (PathResolver.StudyArchiveDBPath, added by
// #4) recording what has been captured, so the tool can answer "do I already have this
// match?" and so the study viewer can later browse by map, mode and player.
//
// WHY THE DDL LIVES HERE AND NOT IN internal/migration. That package is the app's boot
// migration chain, applied by cmd/server. This database belongs to the archiver alone:
// nothing else creates it, nothing else writes it, and it must not appear or disappear
// with the server's schema version. Its shape is therefore declared where its only
// writer is.
//
// SINGLE WRITER BY CONSTRUCTION: the archiver is a serial batch job and the only process
// that writes here, which is why the ticket exempts it from the shared-DB BatchBuilder
// machinery (ADR 0019/0030).
//
// BUT SINGLE-WRITER IS NOT A SAFETY ARGUMENT ABOUT THE ART BUG, and it must not be read
// as one. `internal/sync/no_art_patterns_test.go` records that a compaction DELETE
// "déclenchait le bug ART #23046 malgré mono-writer + PK BIGINT (crash JGtm 2026-06-20)".
// DuckDB #23046 is about index maintenance during row removal; it does not care how many
// processes are watching, and both keys here are VARCHAR — the case that file's allowlist
// explicitly refuses. The write discipline in archive_write.go is what keeps this
// database safe, not the writer count.
//
// READERS (the study server of Spec 2) must open through `duckdb.OpenReadForQuery`, NOT
// a forced `OpenReadOnly`: DuckDB refuses a read-only handle on a file already held
// read-write in the same process, which is the rule ADR 0013/0016 and CLAUDE.md's ART
// rule 4 already state for every other database here.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"levelup/go-api/internal/domain/title"
	ddb "levelup/go-api/internal/platform/duckdb"
)

// archiveSchema is the whole shape of the archive, applied on every open.
//
// The primary keys are declared HERE, in the initial CREATE, and that is deliberate:
// `CREATE TABLE IF NOT EXISTS` never adds a key to a table that already exists (a trap
// the repo has already paid for), so a key added later would silently not exist on any
// database created before it.
const archiveSchema = `
CREATE TABLE IF NOT EXISTS matches (
    match_id        VARCHAR PRIMARY KEY,
    short_id        VARCHAR NOT NULL,
    played_at       TIMESTAMPTZ,
    map_name        VARCHAR,
    map_module      VARCHAR,
    mode            VARCHAR,
    playlist        VARCHAR,
    duration_ms     BIGINT,
    source_gamertag VARCHAR,
    -- film_state: the life cycle of filmstate.go - pending / downloaded / failed /
    -- expired. Only expired is terminal: it stops every later run, the others do not.
    film_state      VARCHAR NOT NULL,
    skip_reason     VARCHAR,
    artifact_path   VARCHAR,
    built_at        TIMESTAMPTZ,
    decoder_rev     VARCHAR,
    tracks          INTEGER,
    points          INTEGER,
    shots           INTEGER,
    named_lives     INTEGER,
    total_lives     INTEGER,
    recorded_at     TIMESTAMPTZ NOT NULL
);
CREATE TABLE IF NOT EXISTS participants (
    match_id VARCHAR NOT NULL,
    xuid     VARCHAR NOT NULL,
    gamertag VARCHAR,
    -- team: 0 = Eagle, 1 = Cobra. outcome: Halo's own encoding, 1 = tie, 2 = win,
    -- 3 = loss, 4 = did-not-finish. Both are stored raw, as the API reports them, so a
    -- reader never has to guess which side of a re-mapping a row was written on.
    team     INTEGER,
    outcome  INTEGER,
    kills    INTEGER,
    deaths   INTEGER,
    assists  INTEGER,
    PRIMARY KEY (match_id, xuid)
);
CREATE TABLE IF NOT EXISTS watchlist (
    gamertag     VARCHAR PRIMARY KEY,
    xuid         VARCHAR,
    added_at     TIMESTAMPTZ NOT NULL,
    last_checked TIMESTAMPTZ
);`

// archive is an open handle on the archive database.
type archive struct {
	db *ddb.DB
}

// openArchive opens (creating if needed) the archive database and applies its schema.
func openArchive(path string) (*archive, error) {
	if err := os.MkdirAll(filepath.Dir(path), cacheDirPerm); err != nil {
		return nil, fmt.Errorf("study data directory: %w", err)
	}
	db, err := ddb.OpenReadWrite(path)
	if err != nil {
		return nil, fmt.Errorf("archive database %s: %w", path, err)
	}
	if _, err := db.Exec(context.Background(), archiveSchema); err != nil {
		// Logged, not swallowed: a close that also fails hides nothing behind the schema
		// error the caller is about to see, but it does mean a leaked DuckDB handle.
		if cErr := db.Close(); cErr != nil {
			slog.Error("study-archiver: closing the archive after a failed schema apply",
				"err", cErr, "path", path)
		}
		return nil, fmt.Errorf("archive schema: %w", err)
	}
	return &archive{db: db}, nil
}

// openArchiveAt is the PathResolver-rooted form, the one production uses.
func openArchiveAt(paths *title.PathResolver) (*archive, error) {
	return openArchive(paths.StudyArchiveDBPath())
}

func (a *archive) Close() error {
	if a == nil || a.db == nil {
		return nil
	}
	return a.db.Close()
}

// matchRecord is one row of `matches`. It is the archive's view of an outcome: what was
// played, what state its film is in, and what the decoder got out of it.
type matchRecord struct {
	MatchID    string
	ShortID    string
	PlayedAt   *time.Time
	MapName    string
	MapModule  string
	Mode       string
	Playlist   string
	DurationMS *int64
	SourceGT   string
	State      filmState
	SkipReason reason
	// ArtifactPath, BuiltAt and DecoderRev are set only when an artifact was written.
	ArtifactPath string
	BuiltAt      *time.Time
	DecoderRev   string
	Tracks       int
	Points       int
	Shots        int
	NamedLives   int
	TotalLives   int
}

// participantRecord is one row of `participants`. Team and outcome come from MATCH
// STATS and never from the film: the film carries no team information at all, and the
// replay artifact's team field is unset by design.
type participantRecord struct {
	XUID     string
	Gamertag string
	Team     *int
	Outcome  *int
	Kills    *int
	Deaths   *int
	Assists  *int
}

// recorded reads back what the archive already knows about a match.
//
// ONE reader rather than one per question. The idempotency check needs the state, the
// artifact path AND the counts (to report what a skipped re-run already holds), and
// three separate small queries would each be a place for the answers to disagree.
//
// PARTIAL BY DESIGN: it selects what the idempotency check uses, so Mode, Playlist,
// PlayedAt, SourceGT, BuiltAt and DecoderRev come back ZERO whatever the row holds.
// Do not hand the result to anything that reports those fields without widening the
// SELECT first.
func (a *archive) recorded(ctx context.Context, matchID string) (matchRecord, bool, error) {
	// Every nullable column is scanned through sql.NullString: map_name and map_module
	// are NULL for a match whose stats named no map, and scanning that into a string
	// fails at runtime on exactly the row this check exists to read.
	var (
		rec                                matchRecord
		state                              string
		mapName, mapModule, skip, artifact sql.NullString
	)
	err := a.db.QueryRow(ctx, `
        SELECT short_id, map_name, map_module, film_state, skip_reason, artifact_path,
               tracks, points, shots, named_lives, total_lives
        FROM matches WHERE match_id = ?`, matchID).
		Scan(&rec.ShortID, &mapName, &mapModule, &state, &skip, &artifact,
			&rec.Tracks, &rec.Points, &rec.Shots, &rec.NamedLives, &rec.TotalLives)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return matchRecord{}, false, nil
	case err != nil:
		return matchRecord{}, false, fmt.Errorf("reading archive row of %s: %w", matchID, err)
	}
	rec.MatchID = matchID
	rec.State = filmState(state)
	rec.MapName, rec.MapModule = mapName.String, mapModule.String
	rec.SkipReason = reason(skip.String)
	rec.ArtifactPath = artifact.String
	return rec, true, nil
}
