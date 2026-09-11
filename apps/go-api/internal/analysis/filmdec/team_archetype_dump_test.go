package filmdec

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// team_archetype_dump_test.go — WHERE COULD A PLAYER-ENTITY SLOT BE TIED TO A PLAYER?
//
// The team designator reads cleanly (a 4-4 split by slot in all six films), but neither the
// slot rank nor the slot offset is the player index. This dump lists the team archetype's
// components, the components present in its keyframe records, and every archetype component
// whose name mentions a player, team, owner or biped: the candidates for the missing link.
//
//	REPLAY_FILM_DIR=C:/…/data/cache/film_chunks/0e97be38 go test ./internal/analysis/filmdec/ -run TestTeamArchetypeDump -v
func TestTeamArchetypeDump(t *testing.T) {
	dir := os.Getenv("REPLAY_FILM_DIR")
	if dir == "" {
		t.Skip("REPLAY_FILM_DIR not set: bench, not CI")
	}
	raw, err := os.ReadFile(filepath.Join(dir, "chunk_00.bin")) //nolint:gosec // bench input path
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	arch, ok := teamDesignatorArchetype(reg)
	if !ok {
		t.Fatal("no team archetype")
	}
	t.Logf("team archetype ti=%d, %d components:", arch.Index, len(arch.Components))
	for i, c := range arch.Components {
		t.Logf("  i%-2d %s", i, c)
	}
	for _, a := range reg.Archetypes {
		for i, c := range a.Components {
			l := strings.ToLower(c)
			if strings.Contains(l, "player") || strings.Contains(l, "team") ||
				strings.Contains(l, "owner") || strings.Contains(l, "biped") || strings.Contains(l, "user") {
				t.Logf("candidate ti=%d i%d %s", a.Index, i, c)
			}
		}
	}

	present := map[string]int{}
	payloads := map[string]string{}
	records, desync := 0, 0
	for c := 1; c <= 3; c++ {
		chunk, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range WalkPackets(chunk) {
			if p.Type != PacketTypeKeyframe {
				continue
			}
			pay := p.Payload(chunk)
			for _, r := range WalkKeyframeWorld(pay) {
				if r.TI != arch.Index {
					continue
				}
				br := NewBitReader(pay)
				br.SetBitPos(r.Bit + teamDesignatorHeaderBits)
				consumeDefaultStateTI9(br)
				br.ReadBit()
				tr := decodeDeltaWithArch(br, arch, uint32(r.TI))
				records++
				if tr.DesyncAt >= 0 {
					desync++
				}
				for _, cr := range tr.Comps {
					present[cr.Name]++
					if cr.Payload != nil && payloads[cr.Name] == "" {
						payloads[cr.Name] = fmt.Sprintf("%+v", cr.Payload)
					}
				}
			}
		}
	}
	t.Logf("keyframe records (chunks 1-3): %d, desynced %d", records, desync)
	names := make([]string, 0, len(present))
	for n := range present {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		t.Logf("  present %4d  %s  %s", present[n], n, payloads[n])
	}
}
