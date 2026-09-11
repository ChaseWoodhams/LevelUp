package replay

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// team_link_measure_test.go — CAN THE FILM'S TEAM READS BE TIED TO PLAYERS?
//
// The film carries a team value per managed-player entity slot (filmdec.ScanFilmTeamDesignators)
// but not which player a slot is. The official match stats give each player's team. This bench
// tests candidate slot -> player-index links against that independent split: a link is real only
// if, under ONE fixed designator -> side labelling, every player of every film lands on their
// official team.
//
// LAUNCH (films and participants are not in the repository; absolute paths):
//
//	REPLAY_FILMS=C:/…/data/cache/film_chunks \
//	REPLAY_PARTICIPANTS=C:/…/scratchpad \   // part_<shortid>.json: {"participants":[{xuid,team_side}]}
//	go test ./internal/analysis/replay/ -run TestTeamLinkMeasurement -v
func TestTeamLinkMeasurement(t *testing.T) {
	root, partDir := os.Getenv("REPLAY_FILMS"), os.Getenv("REPLAY_PARTICIPANTS")
	if root == "" || partDir == "" {
		t.Skip("REPLAY_FILMS / REPLAY_PARTICIPANTS not set: bench, not CI")
	}
	files, _ := filepath.Glob(filepath.Join(partDir, "part_*.json"))
	if len(files) == 0 {
		t.Fatalf("no part_*.json in %s", partDir)
	}
	// hypothesis -> designator value -> official side -> players
	tally := map[string]map[uint8]map[string]int{}
	add := func(h string, team uint8, side string) {
		if tally[h] == nil {
			tally[h] = map[uint8]map[string]int{}
		}
		if tally[h][team] == nil {
			tally[h][team] = map[string]int{}
		}
		tally[h][team][side]++
	}
	for _, f := range files {
		id := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(f), "part_"), ".json")
		dir := filepath.Join(root, id)
		parts, err := readParticipants(f)
		if err != nil {
			t.Errorf("%s: %v", id, err)
			continue
		}
		teams, err := filmdec.ScanFilmTeamDesignators(dir)
		if err != nil {
			t.Errorf("%s: %v", id, err)
			continue
		}
		roster := make([]uint64, 0, len(parts))
		for x := range parts {
			roster = append(roster, x)
		}
		sort.Slice(roster, func(i, j int) bool { return roster[i] < roster[j] })
		pit, err := ScanFilmPlayerIndices(dir, roster)
		if err != nil {
			t.Errorf("%s: player indices: %v", id, err)
			continue
		}

		var stable []filmdec.PlayerEntityTeam
		desc := make([]string, 0, len(teams))
		for _, e := range teams {
			desc = append(desc, strconv.Itoa(e.Slot)+":"+readsString(e.Reads))
			if _, ok := e.Team(); ok {
				stable = append(stable, e)
			}
		}
		t.Logf("%s slots %d (stable %d): %s", id, len(teams), len(stable), strings.Join(desc, " "))
		t.Logf("%s player indices (readings %d, disagreements %d): %v", id, pit.Readings,
			pit.Disagreements, pit.ByXUID)
		if len(stable) == 0 {
			continue
		}
		first := stable[0].Slot
		bySlot := map[int]uint8{}
		for _, e := range stable {
			bySlot[e.Slot], _ = e.Team()
		}
		for _, x := range roster {
			pi, ok := pit.ByXUID[x]
			if !ok {
				continue
			}
			side := parts[x]
			if pi < len(stable) {
				team, _ := stable[pi].Team()
				add("slot rank = player index", team, side)
			}
			if team, ok := bySlot[first+pi]; ok {
				add("slot - first slot = player index", team, side)
			}
		}
	}
	hyps := make([]string, 0, len(tally))
	for h := range tally {
		hyps = append(hyps, h)
	}
	sort.Strings(hyps)
	for _, h := range hyps {
		for team := uint8(0); team < 16; team++ {
			if sides, ok := tally[h][team]; ok {
				t.Logf("%-34s designator %2d -> official %v", h, team, sides)
			}
		}
	}
}

func readParticipants(path string) (map[uint64]string, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // bench input path from the environment
	if err != nil {
		return nil, err
	}
	var doc struct {
		Participants []struct {
			XUID     string `json:"xuid"`
			TeamSide string `json:"team_side"`
		} `json:"participants"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	out := map[uint64]string{}
	for _, p := range doc.Participants {
		x, err := strconv.ParseUint(p.XUID, 10, 64)
		if err != nil {
			continue
		}
		out[x] = p.TeamSide
	}
	return out, nil
}

func readsString(reads map[uint8]int) string {
	keys := make([]int, 0, len(reads))
	for k := range reads {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, "e"+strconv.Itoa(k)+"x"+strconv.Itoa(reads[uint8(k)]))
	}
	return strings.Join(parts, "/")
}
