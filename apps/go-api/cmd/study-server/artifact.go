package main

// artifact.go — SERVING THE REPLAY ARTIFACT, BYTE FOR BYTE.
//
// THE PATH IS NOT READ OUT OF THE ARCHIVE, IT IS RESOLVED. The `matches.artifact_path`
// column holds the ABSOLUTE path of the machine that built the artifact, which is a record
// of what happened, not an address: restore an archive next to a repository at another
// location — or on another machine — and every one of those paths is wrong. The address
// comes from `PathResolver.ReplayArtifactPath`, the same call the archiver used to write
// the file, so the two agree by construction rather than by luck.
//
// That resolution is also where the short-film-ID rule lives: everything derived from a film
// is filed under the 8 hexadecimal characters before the first dash (`title.FilmShortMatchID`),
// while the rest of the application speaks full match ids. Reusing the resolver is what keeps
// this server from re-deriving that rule — the whole reason the ticket asks for Go here and
// not a second implementation in Node.
//
// THE BYTES ARE PASSED THROUGH, NOT DECODED AND RE-ENCODED. Two reasons, both load-bearing:
//
//   - A round trip through `replay.ReplayDocument` silently DROPS anything the struct does
//     not model. An artifact built by a newer decoder would arrive at the viewer quietly
//     shorn of its new fields, which is the failure the client's schema-version guard
//     (ticket #13) exists to catch — and it cannot catch what the server already discarded.
//   - These documents run to megabytes of trajectories. Decoding and re-marshalling one per
//     request buys nothing at all when the file on disk is already the response.

import (
	"errors"
	"fmt"
	"os"

	"levelup/go-api/internal/domain/title"
)

// errArtifactMissing is "the archive says this match was built, and the file is not there" —
// a torn state (a hand-cleared cache, a half-restored backup), not a normal outcome.
var errArtifactMissing = errors.New("replay artifact missing")

// artifacts resolves and reads the replay artifacts of one title.
type artifacts struct {
	paths     *title.PathResolver
	titleSlug string
}

// read returns the artifact of a match, exactly as it sits on disk.
//
// The identifier must come from an archive row, never from the request: it is interpolated
// into a filesystem path, and a value chosen by the caller would choose the path with it.
func (a artifacts) read(matchID string) ([]byte, error) {
	path := a.paths.ReplayArtifactPath(a.titleSlug, matchID)
	blob, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("%w for %s at %s", errArtifactMissing, matchID, path)
	}
	if err != nil {
		return nil, fmt.Errorf("reading the replay artifact of %s: %w", matchID, err)
	}
	return blob, nil
}
