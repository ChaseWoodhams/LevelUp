package main

// main_test.go — the wiring, end to end: a repo root on disk, the archive where the
// PathResolver says it lives, and an artifact where the resolver says that one lives.
//
// The handler tests build the pieces directly, which is what makes them readable; this one
// exists because assembling those pieces is itself a thing that can be wrong — an archive
// looked for beside the binary, an artifact resolved under the wrong title — and no amount of
// handler coverage would notice.

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/domain/title"
)

func TestNewHandler_ResolvesArchiveAndArtifactFromTheRepoRoot(t *testing.T) {
	root := t.TempDir()
	t.Setenv("LEVELUP_REPO_ROOT", root)

	paths := title.NewPathResolver(root)
	if err := os.MkdirAll(paths.StudyDataDir(), 0o755); err != nil {
		t.Fatalf("study data directory: %v", err)
	}
	seedArchive(t, paths.StudyArchiveDBPath())
	writeTestArtifact(t, artifacts{paths: paths, titleSlug: title.DefaultSlug}, cliffhangerID)

	h, err := newHandler(title.DefaultSlug)
	if err != nil {
		t.Fatalf("newHandler: %v", err)
	}

	r := chi.NewRouter()
	h.mount(r)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/matches/000d5950/replay", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}
	var doc map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &doc); err != nil {
		t.Fatalf("unreadable artifact: %v", err)
	}
	if doc["matchId"] != "000d5950" {
		t.Errorf("served %v, want the archived artifact", doc["matchId"])
	}
}

// TestNewHandler_NoArchiveYet is the state every machine is in until the first capture. The
// message has to name the command that fixes it.
func TestNewHandler_NoArchiveYet(t *testing.T) {
	t.Setenv("LEVELUP_REPO_ROOT", t.TempDir())
	_, err := newHandler(title.DefaultSlug)
	if !errors.Is(err, errNoArchive) {
		t.Fatalf("err = %v, want errNoArchive", err)
	}
	if got := err.Error(); !strings.Contains(got, "study-archiver") {
		t.Errorf("the message must point at the archiver: %s", got)
	}
}

func TestRun_RejectsAPositionalArgument(t *testing.T) {
	t.Setenv("LEVELUP_REPO_ROOT", t.TempDir())
	if code := run(context.Background(), []string{"matches"}); code != exitUsage {
		t.Errorf("exit = %d, want %d", code, exitUsage)
	}
}

// TestRun_ReportsAMissingArchive — a failure to start is exit 1, distinct from the exit 2 a
// misspelled flag gets: one is the operator's typing, the other is the machine's state.
func TestRun_ReportsAMissingArchive(t *testing.T) {
	t.Setenv("LEVELUP_REPO_ROOT", t.TempDir())
	if code := run(context.Background(), nil); code != exitFailure {
		t.Errorf("exit = %d, want %d", code, exitFailure)
	}
}
