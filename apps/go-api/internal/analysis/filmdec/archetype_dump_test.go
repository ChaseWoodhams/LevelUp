package filmdec

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// archetype_dump_test.go — QUE CONTIENT L'ARCHÉTYPE D'UN TYPE D'ENTITÉ.
//
// La table d'archétypes vit dans le film lui-même (chunk_00) : elle liste, POUR CHAQUE type
// d'entité, les composants que ses enregistrements peuvent porter, dans l'ordre des bits du
// masque de présence. C'est donc elle qui dit si un composant est seulement CONCEVABLE sur un
// type donné — avant toute question de largeur de default-state ou de traversée.
//
// POURQUOI CE TEST EXISTE. Calibrer la largeur de default-state de ti=0 pour atteindre
// `game-engine-round-timer-component` n'a de sens que si l'archétype de ti=0 le porte. La
// vérification coûte une lecture de chunk_00 ; la calibration coûte un balayage de largeurs
// sur tout le film. On fait la moins chère d'abord.
//
// LANCEMENT :
//
//	REPLAY_FILMS=<...>/data/cache/film_chunks go test ./internal/analysis/filmdec/ \
//	  -run TestArchetypeDump -v

func TestArchetypeDump(t *testing.T) {
	root := os.Getenv("REPLAY_FILMS")
	if root == "" {
		t.Skip("REPLAY_FILMS non défini : banc hors CI")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("REPLAY_FILMS illisible : %v", err)
	}
	var reg *Registry
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(root, e.Name(), "chunk_00.bin"))
		if err != nil {
			continue
		}
		if reg, err = ParseRegistryChunk(raw); err == nil {
			t.Logf("registre lu depuis %s", e.Name())
			break
		}
	}
	if reg == nil {
		t.Fatal("aucun registre lisible")
	}

	// Où vivent les composants qui nous intéressent, dans TOUS les archétypes.
	wanted := []string{"game-engine-", "statborg-", "team-designator", "respawn-timer",
		"dead-state", "lives-remaining"}
	t.Logf("=== OU VIVENT LES COMPOSANTS RECHERCHES (tous archetypes) ===")
	for ti := 0; ti < 50; ti++ {
		arch, ok := reg.Archetype(ti)
		if !ok {
			continue
		}
		var hits []string
		for i, c := range arch.Components {
			for _, w := range wanted {
				if strings.Contains(c, w) {
					hits = append(hits, c+"@i"+itoaSmall(i))
					break
				}
			}
		}
		if len(hits) > 0 {
			t.Logf("ti=%-2d (%d composants) : %s", ti, len(arch.Components), strings.Join(hits, ", "))
		}
	}

	t.Logf("")
	t.Logf("=== ARCHETYPE ti=0 EN ENTIER ===")
	if arch, ok := reg.Archetype(0); ok {
		t.Logf("ti=0 porte %d composants", len(arch.Components))
		for i, c := range arch.Components {
			t.Logf("  i%-3d %s", i, c)
		}
	} else {
		t.Logf("ti=0 : aucun archetype dans le registre")
	}
}

func itoaSmall(v int) string {
	if v == 0 {
		return "0"
	}
	var b []byte
	for v > 0 {
		b = append([]byte{byte('0' + v%10)}, b...)
		v /= 10
	}
	return string(b)
}

// TestArchetypeComponentCatalogue liste TOUS les composants declares par les archetypes du
// film, avec le nombre d'archetypes porteurs. C'est le catalogue AUTHENTIQUE de ce que le
// format peut porter : l'executable ne contient aucun nom de composant (verifie : 0 chaine
// « *-component »), il dispatche par hachage. La seule source de noms est donc le registre
// du film lui-meme.
//
// L'INTERET : diffuse contre les 130 noms que le decodeur dispatche, ce catalogue dit
// exactement ce qui existe et n'est PAS gere — c'est-a-dire a la fois une source de desync
// et un gisement de donnees non lues.
func TestArchetypeComponentCatalogue(t *testing.T) {
	root := os.Getenv("REPLAY_FILMS")
	if root == "" {
		t.Skip("REPLAY_FILMS non défini : banc hors CI")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("REPLAY_FILMS illisible : %v", err)
	}
	holders := map[string]int{}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(root, e.Name(), "chunk_00.bin"))
		if err != nil {
			continue
		}
		reg, err := ParseRegistryChunk(raw)
		if err != nil {
			continue
		}
		for ti := 0; ti < 50; ti++ {
			arch, ok := reg.Archetype(ti)
			if !ok {
				continue
			}
			seen := map[string]bool{}
			for _, c := range arch.Components {
				if !seen[c] {
					seen[c] = true
					holders[c]++
				}
			}
		}
		break // un seul film suffit : le registre est le meme catalogue
	}
	names := make([]string, 0, len(holders))
	for n := range holders {
		names = append(names, n)
	}
	sort.Strings(names)
	t.Logf("CATALOGUE %d composants distincts", len(names))
	for _, n := range names {
		t.Logf("CATNAME %s", n)
	}
}
