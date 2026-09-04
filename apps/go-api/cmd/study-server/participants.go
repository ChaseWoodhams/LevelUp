package main

// participants.go — WHO PLAYED, IN THE SHAPE THE VIEWER ALREADY READS.
//
// The replay artifact carries no team information at all — the film does not record one —
// and it identifies players by xuid and nothing else. Team, gamertag and K/D/A come from the
// match stats, which the archiver recorded alongside the artifact. This endpoint is that
// half of the join.
//
// THE PAYLOAD IS A SUBSET OF THE APP'S OWN `MatchScoreboardRow`, field for field, because the
// study viewer reuses the app's roster logic unchanged: `rosterLogic.ts` joins on `xuid`,
// names players by `gamertag` and groups them by `team_side`, and `ReplayTeams.tsx` reads
// `kills` / `deaths` / `assists`. Those six are exactly what is published here.
//
// AND IT IS A SUBSET ON PURPOSE — the two required fields of `MatchScoreboardRow` this server
// leaves out are the two it cannot answer honestly:
//
//   - `is_me` asks which row is the viewer's. Studying an archive is always looking at a
//     match from OUTSIDE it; there is no viewer in the archive to point at, and emitting
//     `false` for everyone would be a claim rather than an omission.
//   - `outcome_label` is a localised label. The archive stores Halo's raw outcome code, and
//     turning a code into words is the TitleSemanticAdapter's job against a versioned TOML
//     (CLAUDE.md rule 1 forbids an FR/EN label written into Go). Publishing the label would
//     mean wiring the whole title adapter chain into a server that has no other use for it.
//
// Both are optional in the study app's own copy of the type. A reader who needs the outcome
// can have the raw code added here; what must not happen is a made-up label.

import (
	"context"
	"database/sql"
	"fmt"
)

// participantRow is one player of an archived match.
type participantRow struct {
	XUID     string `json:"xuid"`
	Gamertag string `json:"gamertag"`
	// TeamSide is the app's own "t{N}" encoding of the team id (cf. the scoreboard builder
	// in internal/service). Absent when the stats named no team for this player, which is
	// what puts them in the viewer's ungrouped bucket rather than in a guessed team.
	TeamSide *string `json:"team_side,omitempty"`
	Kills    *int    `json:"kills,omitempty"`
	Deaths   *int    `json:"deaths,omitempty"`
	Assists  *int    `json:"assists,omitempty"`
}

// readParticipants returns a match's roster, in a stable order.
//
// ORDERED BY TEAM THEN XUID, never by insertion: a roster panel that reshuffles between two
// loads of the same match is unreadable, and the archive's own row order is whatever the
// stats payload happened to list.
func (a *archive) readParticipants(ctx context.Context, matchID string) ([]participantRow, error) {
	rows, err := a.db.QueryContext(ctx, `
        SELECT xuid, gamertag, team, kills, deaths, assists
        FROM participants WHERE match_id = ?
        ORDER BY team NULLS LAST, xuid`, matchID)
	if err != nil {
		return nil, fmt.Errorf("reading the roster of %s: %w", matchID, err)
	}
	defer closeRows(ctx, rows, "participants")

	out := []participantRow{}
	for rows.Next() {
		p, err := scanParticipant(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning a participant of %s: %w", matchID, err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading the roster of %s: %w", matchID, err)
	}
	return out, nil
}

func scanParticipant(rows *sql.Rows) (participantRow, error) {
	var (
		p               participantRow
		gamertag        sql.NullString
		team, kills     sql.NullInt64
		deaths, assists sql.NullInt64
	)
	if err := rows.Scan(&p.XUID, &gamertag, &team, &kills, &deaths, &assists); err != nil {
		return participantRow{}, err
	}
	p.Gamertag = gamertag.String
	if team.Valid {
		side := fmt.Sprintf("t%d", team.Int64)
		p.TeamSide = &side
	}
	p.Kills, p.Deaths, p.Assists = nullInt(kills), nullInt(deaths), nullInt(assists)
	return p, nil
}

func nullInt(v sql.NullInt64) *int {
	if !v.Valid {
		return nil
	}
	n := int(v.Int64)
	return &n
}
