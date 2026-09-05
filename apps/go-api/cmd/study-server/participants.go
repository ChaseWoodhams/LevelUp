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
// AN UNKNOWN VALUE IS `null`, NEVER A MISSING KEY. None of these fields carries `omitempty`,
// and that is deliberate: the consumer's type declares them as `T | null`
// (`apps/web/src/lib/api/types.ts` — `team_side: string | null`, `kills: number | null`), so
// `null` is a value it already models and an absent key is not. `omitempty` would have made
// the key VANISH on a player whose stats named no team — the exact row this endpoint exists to
// represent honestly — and the difference would not have shown up in any fixture where every
// player has a team.
//
// IT IS A SUBSET, AND FULL ASSIGNABILITY WAS NEVER ON THE TABLE. `MatchScoreboardRow` declares
// some twenty non-optional fields (`rank`, `score`, `accuracy`, `damage_dealt`, `shots_fired`,
// …); the archive stores six of them. The ticket asks for "xuid, team side, gamertag, kills,
// deaths, assists ... in the shape the copied roster logic already consumes", and the thing
// that must work unchanged is the LOGIC — `rosterLogic.ts` joins on `xuid`, names by
// `gamertag`, groups by `team_side`, and `ReplayTeams.tsx` reads the three counters. Publishing
// fifteen more nulls to satisfy a compiler would be inventing a scoreboard the archive does
// not have.
//
// The two the server could have faked and does not:
//
//   - `is_me` asks which row is the viewer's. Studying an archive is always looking at a
//     match from OUTSIDE it; there is no viewer in the archive to point at.
//   - `outcome_label` is a localised label. The archive stores Halo's raw outcome code, and
//     turning a code into words is the TitleSemanticAdapter's job against a versioned TOML
//     (CLAUDE.md rule 1 forbids an FR/EN label written into Go). Publishing the label would
//     mean wiring the whole title adapter chain into a server that has no other use for it.
//
// So the study app's own type is a `Pick<>` of these six, which is what the ticket named.

import (
	"context"
	"database/sql"
	"fmt"
)

// teamSideFormat is the app's encoding of a team id as a `team_side`, and this is the SECOND
// copy of it — the first is `internal/service/match_view_builders_team.go`, which the ticket
// puts out of bounds for edits, so the literal cannot be centralised today.
//
// Two copies is what the repo allows (CLAUDE.md rule 6); what the rule asks for at the second
// is that the copies cannot drift apart in silence. `TestTeamSideEncodingMatchesTheApp` is that
// guard-rail: it reads the app's source and fails if the two ever stop agreeing. A third copy
// owes a shared helper.
const teamSideFormat = "t%d"

// participantRow is one player of an archived match. Every field is always present; an
// unknown one is null (cf. the file header).
type participantRow struct {
	XUID     string `json:"xuid"`
	Gamertag string `json:"gamertag"`
	// TeamSide is the app's own "t{N}" encoding of the team id (cf. the scoreboard builder
	// in internal/service, pinned by TestTeamSideEncodingMatchesTheApp). Null when the stats
	// named no team for this player, which is what puts them in the viewer's ungrouped
	// bucket rather than in a guessed team.
	TeamSide *string `json:"team_side"`
	Kills    *int    `json:"kills"`
	Deaths   *int    `json:"deaths"`
	Assists  *int    `json:"assists"`
}

// readParticipants returns a match's roster, in a stable order.
//
// ORDERED BY TEAM THEN XUID, never by insertion: a roster panel that reshuffles between two
// loads of the same match is unreadable, and the archive's own row order is whatever the
// stats payload happened to list.
func (a *archive) readParticipants(ctx context.Context, matchID string) ([]participantRow, error) {
	rows, err := a.db.QueryContext(ctx, `
        SELECT `+participantColumns+`
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

// participantColumns is what a participant is read from, in the order the scanners below expect.
// One list for both readers — the roster of one match, and the rosters of a page of them.
const participantColumns = `xuid, gamertag, team, kills, deaths, assists`

// participantFields holds the nullable columns as SQL hands them over, before they become the
// row's pointers. Shared so that "an unknown value is null, never a zero" is decided ONCE,
// however the row was queried.
type participantFields struct {
	gamertag        sql.NullString
	team, kills     sql.NullInt64
	deaths, assists sql.NullInt64
}

// dest is the scan target list, in `participantColumns` order.
func (f *participantFields) dest(p *participantRow) []any {
	return []any{&p.XUID, &f.gamertag, &f.team, &f.kills, &f.deaths, &f.assists}
}

func (f participantFields) into(p *participantRow) {
	p.Gamertag = f.gamertag.String
	if f.team.Valid {
		side := fmt.Sprintf(teamSideFormat, f.team.Int64)
		p.TeamSide = &side
	}
	p.Kills, p.Deaths, p.Assists = nullInt(f.kills), nullInt(f.deaths), nullInt(f.assists)
}

func scanParticipant(rows *sql.Rows) (participantRow, error) {
	var p participantRow
	var f participantFields
	if err := rows.Scan(f.dest(&p)...); err != nil {
		return participantRow{}, err
	}
	f.into(&p)
	return p, nil
}

// scanPageParticipant reads a participant WITH the match it belongs to — the shape the browser's
// one-query-per-page roster read comes back in (cf. archive.attachRosters).
func scanPageParticipant(rows *sql.Rows, matchID *string) (participantRow, error) {
	var p participantRow
	var f participantFields
	if err := rows.Scan(append([]any{matchID}, f.dest(&p)...)...); err != nil {
		return participantRow{}, fmt.Errorf("scanning a participant of a page of matches: %w", err)
	}
	f.into(&p)
	return p, nil
}

func nullInt(v sql.NullInt64) *int {
	if !v.Valid {
		return nil
	}
	n := int(v.Int64)
	return &n
}
