package main

// helpers_test.go — the plumbing the fake Halo server needs, kept out of the tests
// themselves so the assertions stay readable.

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"levelup/go-api/internal/sync/haloclient"
)

// chunkNamePrefix is the blob file name the Halo manifest uses for a film chunk.
const chunkNamePrefix = "filmChunk"

// statsWithMap builds a match-stats payload shaped like the real one, for testMatchID.
func statsWithMap(mapName string) map[string]any {
	return statsFor(testMatchID, mapName, twoPlayerRoster())
}

// twoPlayerRoster is the smallest honest roster: one player per team.
func twoPlayerRoster() []any {
	return []any{
		statsPlayer("xuid(1)", "JGtm", 0, 2, 15, 9, 4),
		statsPlayer("xuid(2)", "Rival", 1, 3, 9, 15, 2),
	}
}

// fourVFourRoster is the shape the archive exists for: two teams of four. `watch` keeps
// only these, so the fixture has to be able to produce both this and something else.
func fourVFourRoster(firstGamertag string) []any {
	roster := make([]any, 0, 8)
	for i := 0; i < 8; i++ {
		gt := fmt.Sprintf("Player%d", i+1)
		if i == 0 && firstGamertag != "" {
			gt = firstGamertag
		}
		team, outcome := 0, 2
		if i >= 4 {
			team, outcome = 1, 3
		}
		roster = append(roster, statsPlayer(fmt.Sprintf("xuid(%d)", i+1), gt, team, outcome, 15, 9, 4))
	}
	return roster
}

// statsFor builds a match-stats payload shaped like the real one.
//
// It has to be the REAL shape, not just the fields the archiver reads: the extraction
// goes through the repo's own sync.ExtractRegistry / sync.ExtractParticipants, so a
// fixture missing MatchId or StartTime fails there rather than in the archiver. That is
// the point of reusing them — the fixture is held to the same standard as the API.
func statsFor(matchID, mapName string, players []any) map[string]any {
	stats := map[string]any{
		"MatchId": matchID,
		"MatchInfo": map[string]any{
			"StartTime": "2026-05-19T20:15:00.000Z",
			"EndTime":   "2026-05-19T20:24:13.000Z",
			"Duration":  "PT9M13S",
			"Playlist":  map[string]any{"PublicName": "Ranked Arena", "AssetId": "playlist-1"},
			"UgcGameVariant": map[string]any{
				"PublicName": "Slayer", "AssetId": "variant-1",
			},
			"PlaylistMapModePair": map[string]any{"PublicName": "Slayer on Cliffhanger"},
		},
		"Players": players,
	}
	if mapName != "" {
		info, _ := stats["MatchInfo"].(map[string]any)
		info["MapVariant"] = map[string]any{"PublicName": mapName, "AssetId": "map-1"}
	}
	return stats
}

// statsPlayer builds one entry of the stats payload's Players array, in the nesting the
// API actually uses (PlayerTeamStats -> Stats -> CoreStats).
func statsPlayer(playerID, gamertag string, team, outcome, kills, deaths, assists int) map[string]any {
	return map[string]any{
		"PlayerId":   playerID,
		"Gamertag":   gamertag,
		"LastTeamId": float64(team),
		"Outcome":    float64(outcome),
		"Rank":       float64(1),
		"PlayerTeamStats": []any{map[string]any{
			"TeamId": float64(team),
			"Stats": map[string]any{
				"CoreStats": map[string]any{
					"Kills":         float64(kills),
					"Deaths":        float64(deaths),
					"Assists":       float64(assists),
					"PersonalScore": float64(1000),
					"Score":         float64(1000),
				},
			},
		}},
	}
}

// testChunks is a two-chunk film, in the shape writeFilmChunks takes: a header and one
// replication chunk. Enough to be "captured" as far as everything but the decoder is
// concerned, which is what the offline rebuild tests need.
func testChunks() []haloclient.FilmChunk {
	return []haloclient.FilmChunk{
		{Index: 0, ChunkType: haloclient.FilmChunkTypeHeader, Data: []byte("header")},
		{Index: 1, ChunkType: haloclient.FilmChunkTypeReplicationData, Data: []byte("replication")},
	}
}

func fmtChunkName(idx int) string { return fmt.Sprintf("%s%d", chunkNamePrefix, idx) }

// parseChunkIndex reads the chunk index back out of a blob file name.
func parseChunkIndex(base string) (int, error) {
	var idx int
	_, err := fmt.Sscanf(base, chunkNamePrefix+"%d", &idx)
	return idx, err
}

// zlibBytes compresses a payload the way the Halo blob CDN serves film chunks — raw
// zlib, which haloclient.downloadBlob inflates on the way in.
func zlibBytes(t *testing.T, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)
	if _, err := zw.Write(data); err != nil {
		t.Fatalf("zlib write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zlib close: %v", err)
	}
	return buf.Bytes()
}

// redirectTo sends every request to the test server, whatever host the Halo client
// resolved. Cheaper and more faithful than injecting an endpoint resolver: it also
// catches the blob URLs, which the client builds from the manifest rather than from any
// endpoint table.
type redirectTo string

func (r redirectTo) RoundTrip(req *http.Request) (*http.Response, error) {
	target, err := url.Parse(string(r))
	if err != nil {
		return nil, err
	}
	req = req.Clone(req.Context())
	req.URL.Scheme = target.Scheme
	req.URL.Host = target.Host
	req.Host = target.Host
	return http.DefaultTransport.RoundTrip(req)
}
