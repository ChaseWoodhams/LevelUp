// Package archlint — no_data_path_join_test.go : ratchet L2-(2) (ARCHI TOP3).
//
// Interdit tout NOUVEAU `filepath.Join(..., "data", ...)` à la main dans internal/ :
// les chemins physiques passent par PathResolver (domain/title/registry.go), source
// unique (ADR 0008, règle CLAUDE.md « jamais de filepath.Join(..., "data", ...) »).
//
// Allowlist DÉCROISSANTE (datée 2026-07-05) : les sites LÉGITIMES actuels — la
// définition du resolver lui-même, la couche config (défauts du data-root, amont du
// resolver : chicken-and-egg de bootstrap), le wiring DI (server.go) et l'ops de
// seed/migration (chemins legacy). Toute occurrence HORS allowlist (service, repo,
// adapter, sync, persist, games…) fait échouer le test → utiliser PathResolver.
package archlint

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

var dataPathJoinAllowlist = map[string]bool{
	// PathResolver = LA source des chemins data. Il vit désormais dans PLUSIEURS
	// fichiers : `registry.go` a dépassé le seuil de 500 L du dépôt (846 L), donc les
	// nouvelles familles de chemins sortent dans leur propre fichier plutôt que de
	// grossir une dette gelée (règle 5). Ce sont les mêmes méthodes sur le même type —
	// l'intention du ratchet (les chemins physiques ne se composent QUE dans le
	// resolver) est intacte ; seul le nombre de fichiers qui le portent a changé.
	// Ajouter ici tout nouveau fichier de PathResolver, et RIEN d'autre.
	"internal/domain/title/registry.go": true,
	// study_paths.go (2026-09-03, ticket #4) : racine `data/study/` de l'outil d'étude.
	"internal/domain/title/study_paths.go": true,
	// Config : défauts du data-root (amont du resolver, bootstrap).
	"internal/config/config.go":          true,
	"internal/config/config_settings.go": true,
	// Wiring DI (racine api/, hors handlers) : chemins cache/jobs/stash au boot.
	"internal/api/server.go":       true,
	"internal/api/server_apiv1.go": true, // K2a : construction DI extraite de NewRouter
	// Seed démo + migration legacy + backup (chemins historiques / data-root ops).
	"internal/ops/seed_demo.go":            true,
	"internal/ops/seed_demo_multititle.go": true,
	"internal/ops/migrate/migrate.go":      true,
	"internal/ops/backup_service.go":       true,
	// Infra de test (localise les fixtures data ; hors code service prod).
	"internal/testfixtures/paths.go": true,
}

var dataPathJoinRE = regexp.MustCompile(`filepath\.Join\(.*"data"`)

func TestNoNewDataPathJoin(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	internalRoot := filepath.Dir(filepath.Dir(thisFile))

	var violations []string
	err := filepath.WalkDir(internalRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "migrations" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, _ := filepath.Rel(filepath.Dir(internalRoot), path)
		rel = filepath.ToSlash(rel)
		if dataPathJoinAllowlist[rel] {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for i, line := range strings.Split(string(data), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") {
				continue
			}
			if dataPathJoinRE.MatchString(line) {
				violations = append(violations, rel+":"+strconv.Itoa(i+1)+"  "+trimmed)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk internal/: %v", err)
	}
	if len(violations) > 0 {
		t.Errorf("filepath.Join(..., \"data\", ...) à la main interdit (L2-2) — "+
			"passer par PathResolver (domain/title/registry.go), ou allowlister "+
			"transitoirement un site de bootstrap justifié :\n  %s",
			strings.Join(violations, "\n  "))
	}
}
