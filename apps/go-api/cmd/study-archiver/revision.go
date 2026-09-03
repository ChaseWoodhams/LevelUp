package main

// revision.go — WHICH BUILD OF THE DECODER PRODUCED THIS ARTIFACT.
//
// The archive records the revision so that a coverage drop noticed months from now can
// be traced to the build that caused it. Without it, "this match only decoded 3 tracks"
// is unanswerable: nobody can tell whether the film is poor or the decoder regressed
// between two archiving runs.
//
// READ FROM THE BINARY, NOT FROM A LINKER FLAG. Go stamps the VCS revision into every
// binary built inside a Git work tree, so this needs no -ldflags and cannot be forgotten
// at build time. The stamp is ABSENT in two cases that both matter here:
//
//   - `go run ./cmd/study-archiver`, the way this tool is actually used day to day, and
//     `go test`: neither stamps VCS information.
//   - a build with -buildvcs=false, or from outside a work tree.
//
// A missing revision is therefore NORMAL, not an error, and is recorded as the empty
// string (SQL NULL) rather than a fabricated value. `dirty` is kept when present: an
// artifact built from uncommitted code is exactly what a reader needs warned about.

import (
	"runtime/debug"
	"sync"
)

// decoderRevisionOnce caches the answer: reading build info walks the binary's embedded
// table, and every match in a `watch` pass would otherwise pay for it.
var decoderRevisionOnce = sync.OnceValue(readDecoderRevision)

// decoderRevision returns the VCS revision this binary was built from, or "" when the
// build carries no stamp (go run, go test, -buildvcs=false).
func decoderRevision() string { return decoderRevisionOnce() }

func readDecoderRevision() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	return revisionFrom(info.Settings)
}

// revisionFrom is the pure half, split out so it can be tested: a `go test` binary
// carries no VCS stamp of its own, so a test driving readDecoderRevision could only ever
// observe the empty case and would prove nothing about the other two.
func revisionFrom(settings []debug.BuildSetting) string {
	var revision, modified string
	for _, s := range settings {
		switch s.Key {
		case "vcs.revision":
			revision = s.Value
		case "vcs.modified":
			modified = s.Value
		}
	}
	if revision == "" {
		return ""
	}
	if modified == "true" {
		// An artifact built from a dirty tree is not reproducible from the commit
		// alone. Saying so is the whole point of recording the revision.
		return revision + "-dirty"
	}
	return revision
}
