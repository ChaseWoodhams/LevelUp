// Package handlers_test — help_test.go : tests HelpHandler.
package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/service"
)

func makeHelpHandler(t *testing.T, dir string) *handlers.HelpHandler {
	t.Helper()
	builder := service.NewReleaseNotesService(dir)
	return handlers.NewHelpHandler(builder, filepath.Join(dir, "data", "cache"))
}

// setupHelpRepo creates a temporary repository with English release notes.
func setupHelpRepo(t *testing.T, contentEN string) string {
	t.Helper()
	dir := t.TempDir()
	docsDir := filepath.Join(dir, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "RELEASE_NOTES.md"), []byte(contentEN), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

const sampleReadmeEN = `# LevelUp

## What's new

**v7.0 — Challenges**
- Feature A
- Feature B

**v6.5 — Heatmap**
- Feature C

## Features

Some content.
`

func TestHelpHandler_EN_ReturnsWhatsNew(t *testing.T) {
	dir := setupHelpRepo(t, sampleReadmeEN)
	h := makeHelpHandler(t, dir)
	r := chi.NewRouter()
	h.Mount(r)

	req := httptest.NewRequest(http.MethodGet, "/help/release-notes", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	content := resp["content"]
	if !contains(content, "v7.0") {
		t.Errorf("expected v7.0 block in content, got: %q", content)
	}
	if !contains(content, "v6.5") {
		t.Errorf("expected v6.5 block in content, got: %q", content)
	}
	if contains(content, "Some content") {
		t.Error("should not include content after ## Features heading")
	}
}

func TestHelpHandler_DefaultsToEnglish(t *testing.T) {
	dir := setupHelpRepo(t, sampleReadmeEN)
	h := makeHelpHandler(t, dir)
	r := chi.NewRouter()
	h.Mount(r)

	// No language parameter is accepted; the handler serves English.
	req := httptest.NewRequest(http.MethodGet, "/help/release-notes", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if !contains(resp["content"], "Challenges") {
		t.Errorf("expected English default content, got: %q", resp["content"])
	}
}

func TestHelpHandler_MissingReleaseNotes_Returns500(t *testing.T) {
	dir := t.TempDir() // Pas de RELEASE_NOTES.md
	h := makeHelpHandler(t, dir)
	r := chi.NewRouter()
	h.Mount(r)

	req := httptest.NewRequest(http.MethodGet, "/help/release-notes", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestHelpHandler_CacheHit(t *testing.T) {
	dir := setupHelpRepo(t, sampleReadmeEN)
	h := makeHelpHandler(t, dir)
	r := chi.NewRouter()
	h.Mount(r)

	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, httptest.NewRequest(http.MethodGet, "/help/release-notes", nil))
	if w1.Code != http.StatusOK {
		t.Fatalf("first call: expected 200, got %d", w1.Code)
	}

	// Modifier le fichier → le cache ne relit pas immédiatement
	newContent := "## What's new\n**v99.0 — New**\n- Changed\n"
	_ = os.WriteFile(filepath.Join(dir, "docs", "RELEASE_NOTES.md"), []byte(newContent), 0o600)

	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/help/release-notes", nil))

	var resp map[string]string
	_ = json.Unmarshal(w2.Body.Bytes(), &resp)
	if contains(resp["content"], "v99.0") {
		t.Error("cache should not have been invalidated within TTL")
	}
}

func TestHelpHandler_VersionOrder(t *testing.T) {
	readme := `## What's new
**v6.0 — Old**
- old feature
**v7.0 — New**
- new feature
**v6.5 — Mid**
- mid feature
`
	dir := setupHelpRepo(t, readme)
	h := makeHelpHandler(t, dir)
	r := chi.NewRouter()
	h.Mount(r)

	req := httptest.NewRequest(http.MethodGet, "/help/release-notes", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	content := resp["content"]

	pos70 := indexOf(content, "v7.0")
	pos65 := indexOf(content, "v6.5")
	pos60 := indexOf(content, "v6.0")
	if pos70 < 0 || pos65 < 0 || pos60 < 0 {
		t.Fatalf("missing version blocks: %q", content)
	}
	if pos70 >= pos65 || pos65 >= pos60 {
		t.Errorf("expected v7.0 > v6.5 > v6.0, positions: %d %d %d in:\n%s", pos70, pos65, pos60, content)
	}
}

func TestHelpHandler_DiskCacheSurvivesRestart(t *testing.T) {
	dir := setupHelpRepo(t, sampleReadmeEN)
	// Premier handler — construit le cache et l'écrit sur disque.
	h1 := makeHelpHandler(t, dir)
	r1 := chi.NewRouter()
	h1.Mount(r1)
	w1 := httptest.NewRecorder()
	r1.ServeHTTP(w1, httptest.NewRequest(http.MethodGet, "/help/release-notes", nil))
	if w1.Code != http.StatusOK {
		t.Fatalf("first handler: expected 200, got %d: %s", w1.Code, w1.Body.String())
	}
	var resp1 map[string]string
	_ = json.Unmarshal(w1.Body.Bytes(), &resp1)
	original := resp1["content"]

	// Deuxième handler (simule un redémarrage) — mémoire vide, doit utiliser le disque.
	h2 := makeHelpHandler(t, dir)
	r2 := chi.NewRouter()
	h2.Mount(r2)
	w2 := httptest.NewRecorder()
	r2.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/help/release-notes", nil))
	if w2.Code != http.StatusOK {
		t.Fatalf("second handler: expected 200, got %d: %s", w2.Code, w2.Body.String())
	}
	var resp2 map[string]string
	_ = json.Unmarshal(w2.Body.Bytes(), &resp2)

	if resp2["content"] != original {
		t.Errorf("disk cache should return same content after restart\ngot: %q\nwant: %q", resp2["content"], original)
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) >= len(substr) &&
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
