package filmdec

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// team_designators.go — WHICH TEAM EACH PLAYER ENTITY IS ON, read from the film's keyframes.
//
// `managed-player-team-designator-component` (ti=9 i0, FUN_140f581e8, R(4)) is written in every
// keyframe for each managed-player entity. Reading it needs the ti=9 record header, which is NOT
// the 64 bits the biped uses: 47 bits, the only prefix out of 300 tried that gives eight player
// entities with a stable 4-4 team split in all six archived films (component_probe_test.go,
// default_state_sweep_test.go). The default-state that follows is consumeDefaultStateTI9.
//
// WHAT THIS DOES NOT GIVE: which PLAYER an entity slot is. That link is the consumer's to
// establish (replay), and it must be checked against an independent source before any team is
// published.
//
// NO VOTE. A slot read with two different team values is not a majority question: one of the
// reads is wrong, so the slot has no team (PlayerEntityTeam.Team reports false).

// teamDesignatorHeaderBits is the ti=9 keyframe record header, measured (see above).
const teamDesignatorHeaderBits = 47

// PlayerEntityTeam is what the keyframes say about one managed-player entity slot.
type PlayerEntityTeam struct {
	Slot int
	// Reads counts each team value read on the slot.
	Reads map[uint8]int
}

// Team returns the slot's team when every read agrees.
func (p PlayerEntityTeam) Team() (uint8, bool) {
	if len(p.Reads) != 1 {
		return 0, false
	}
	for v := range p.Reads {
		return v, true
	}
	return 0, false
}

// ScanFilmTeamDesignators reads the team designator of every managed-player entity in the
// film's keyframes, sorted by slot.
//
// OFFLINE (disk I/O over the whole film) — never from a request path.
func ScanFilmTeamDesignators(dir string) ([]PlayerEntityTeam, error) {
	n := CountFilmChunks(dir)
	if n == 0 {
		return nil, fmt.Errorf("no film chunks in %s", dir)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "chunk_00.bin")) //nolint:gosec // film cache path
	if err != nil {
		return nil, fmt.Errorf("film registry: %w", err)
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		return nil, fmt.Errorf("film registry: %w", err)
	}
	arch, ok := teamDesignatorArchetype(reg)
	if !ok {
		return nil, fmt.Errorf("no archetype carries %s in %s", compManagedPlayerTeamDesignator, dir)
	}
	bySlot := map[int]map[uint8]int{}
	for c := 1; c <= n; c++ {
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
				team, ok := readTeamDesignator(pay, r, arch)
				if !ok {
					continue
				}
				if bySlot[r.Slot] == nil {
					bySlot[r.Slot] = map[uint8]int{}
				}
				bySlot[r.Slot][team]++
			}
		}
	}
	out := make([]PlayerEntityTeam, 0, len(bySlot))
	for slot, reads := range bySlot {
		out = append(out, PlayerEntityTeam{Slot: slot, Reads: reads})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Slot < out[j].Slot })
	return out, nil
}

// teamDesignatorArchetype finds the archetype carrying the team designator by component NAME:
// a type index is a build index, not a format constant.
func teamDesignatorArchetype(reg *Registry) (Archetype, bool) {
	for _, a := range reg.Archetypes {
		for _, c := range a.Components {
			if c == compManagedPlayerTeamDesignator {
				return a, true
			}
		}
	}
	return Archetype{}, false
}

// readTeamDesignator decodes one keyframe record of the team-designator archetype. A record
// whose traversal desyncs is not read: the criterion the 47-bit header was measured under.
func readTeamDesignator(pay []byte, r KeyframeRec, arch Archetype) (uint8, bool) {
	br := NewBitReader(pay)
	br.SetBitPos(r.Bit + teamDesignatorHeaderBits)
	consumeDefaultStateTI9(br)
	br.ReadBit() // has-components gate
	tr := decodeDeltaWithArch(br, arch, uint32(r.TI))
	if tr.DesyncAt >= 0 {
		return 0, false
	}
	for _, c := range tr.Comps {
		if td, ok := c.Payload.(TeamDesignator); ok {
			return td.Team, true
		}
	}
	return 0, false
}
