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

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/sync/haloclient"
)

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
