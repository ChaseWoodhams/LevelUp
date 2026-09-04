package main

// filmstore.go — THE RAW FILM ON DISK.
//
// The chunks are written to the film-chunk cache layout the offline tools already read
// (`data/cache/film_chunks/<short>/chunk_NN.bin`, owned by PathResolver.FilmChunkPath),
// and they are KEPT after a successful build. The artifact is regenerable, the film is
// not: the CDN link expires in weeks, and a decoder fix six months from now can only
// rebuild what is still on disk.
//
// The bytes are stored INFLATED. haloclient.downloadBlob already un-zlibs what the CDN
// serves, and filmdec.ReadFilmChunk accepts either form — so this matches what
// cmd/fetch_film_chunks has always written, and the existing cache stays one format.

import (
	"fmt"
	"os"
	"strings"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/sync/haloclient"
)

// cachedFilmChunks counts the chunks already in the match's film-chunk directory.
//
// It answers "is the film still on disk?", and a COUNT rather than a boolean because the
// number is what the outcome reports and the archive records. Read against a directory the
// operator can empty at any time — the recorded state alone is not evidence the bytes are
// still there, exactly as a recorded artifact path is not evidence of an artifact.
func cachedFilmChunks(paths *title.PathResolver, matchID string) int {
	entries, err := os.ReadDir(paths.FilmChunksDir(matchID))
	if err != nil {
		// An unreadable or absent directory is simply "nothing cached": the caller
		// re-downloads, and a real permission problem surfaces on the write that follows.
		return 0
	}
	n := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), filmChunkSuffix) {
			n++
		}
	}
	return n
}

// filmChunkSuffix is the extension PathResolver.FilmChunkPath writes
// (`chunk_NN.bin`). Counting only these keeps a stray note or a partial download from
// passing for a chunk.
const filmChunkSuffix = ".bin"

// writeFilmChunks writes every downloaded chunk into the match's film-chunk directory
// and returns how many were written.
//
// EVERY chunk type, not just replication data: the header (type 1) seeds the decoder's
// World and the highlight footer (type 3) carries the death feed that names the lives.
// Dropping either leaves a film that looks complete and rebuilds into a mute replay.
func writeFilmChunks(paths *title.PathResolver, matchID string, chunks []haloclient.FilmChunk) (int, error) {
	dir := paths.FilmChunksDir(matchID)
	if err := os.MkdirAll(dir, cacheDirPerm); err != nil {
		return 0, fmt.Errorf("film directory %s: %w", dir, err)
	}
	written := 0
	for _, chunk := range chunks {
		path := paths.FilmChunkPath(matchID, chunk.Index)
		if err := os.WriteFile(path, chunk.Data, cacheFilePerm); err != nil {
			return written, fmt.Errorf("chunk %d: %w", chunk.Index, err)
		}
		written++
	}
	return written, nil
}
