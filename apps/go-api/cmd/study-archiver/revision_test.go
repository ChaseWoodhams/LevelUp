package main

import (
	"runtime/debug"
	"testing"
)

func TestRevisionFrom(t *testing.T) {
	const sha = "48227ab77c0ffee0000000000000000000000000"

	for _, tc := range []struct {
		name     string
		settings []debug.BuildSetting
		want     string
	}{
		{
			name: "clean build records the commit",
			settings: []debug.BuildSetting{
				{Key: "vcs", Value: "git"},
				{Key: "vcs.revision", Value: sha},
				{Key: "vcs.modified", Value: "false"},
			},
			want: sha,
		},
		{
			// An artifact built from uncommitted code is not reproducible from the
			// commit alone, and a reader chasing a coverage drop has to be told.
			name: "dirty tree is marked",
			settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: sha},
				{Key: "vcs.modified", Value: "true"},
			},
			want: sha + "-dirty",
		},
		{
			// `go run` and `go test` stamp nothing. That is NORMAL, and must record as
			// absent rather than as a fabricated value.
			name:     "no stamp records nothing",
			settings: []debug.BuildSetting{{Key: "-tags", Value: "netgo"}},
			want:     "",
		},
		{
			name:     "no settings at all",
			settings: nil,
			want:     "",
		},
		{
			// A revision with no `vcs.modified` alongside it is still a revision.
			name:     "revision without a modified flag",
			settings: []debug.BuildSetting{{Key: "vcs.revision", Value: sha}},
			want:     sha,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := revisionFrom(tc.settings); got != tc.want {
				t.Errorf("revisionFrom = %q, want %q", got, tc.want)
			}
		})
	}
}

// decoderRevision must be safe to call in any build, including this test binary (which
// carries no stamp). It is cached, so calling it twice must give the same answer.
func TestDecoderRevision_IsStableAndSafe(t *testing.T) {
	first := decoderRevision()
	if second := decoderRevision(); first != second {
		t.Errorf("decoderRevision is not stable: %q then %q", first, second)
	}
}
