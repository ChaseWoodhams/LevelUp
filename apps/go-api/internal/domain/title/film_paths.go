package title

import (
	"fmt"
	"path/filepath"
)

// film_paths.go — WHERE A MATCH'S FILM CHUNKS LIVE.
//
// A separate file rather than another method in `registry.go`, for the reason
// `film_id.go` already gives: registry.go is far past the repo's size threshold, and
// everything derived from a film belongs with the rule that keys it.
//
// The key is the SHORT form (cf. FilmShortMatchID). That is not a preference, it is
// what is on disk: `data/cache/film_chunks/000d5950/chunk_00.bin`.

// FilmChunksDir returns the directory holding a match's raw film chunks.
// Ex: data/cache/film_chunks/000d5950/
//
// Passing the full match_id or its short form gives the SAME directory — the property
// that makes the chunks reachable from a caller that only holds the route's full id.
//
// WHY THIS METHOD EXISTS. cmd/replay-build used to build this path itself, and built it
// from the FULL match_id: `data/cache/film_chunks/<full-uuid>/`, a directory nothing
// ever writes. It went unnoticed because callers pass an explicit film directory as the
// second argument, or hand the tool an already-short id. Routing through here is
// therefore a FIX, not a pure refactor: for a full-length id the resolved path changes,
// from a directory that never existed to the one the cache actually wrote.
func (p *PathResolver) FilmChunksDir(matchID string) string {
	return filepath.Join(p.CacheRootDir(), "film_chunks", FilmShortMatchID(matchID))
}

// FilmChunkPath returns one numbered chunk file inside FilmChunksDir.
// Ex: data/cache/film_chunks/000d5950/chunk_07.bin
//
// The two-digit zero-padded name is the layout the offline tools already read; it is
// pinned here so a caller never re-derives it with its own Sprintf.
func (p *PathResolver) FilmChunkPath(matchID string, index int) string {
	return filepath.Join(p.FilmChunksDir(matchID), fmt.Sprintf("chunk_%02d.bin", index))
}
