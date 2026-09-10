/**
 * Tests formatDataIssues — les dégradations de chargement remontées par l'API
 * doivent produire un message lisible (FR et EN), y compris pour un code inconnu.
 *
 * Régression visée : avant, un LoadMainTeamParticipants / LoadFor en échec était
 * avalé côté serveur (slog.Warn) et la page affichait des chiffres amputés sans
 * rien dire — d'où des compteurs non reproductibles d'une requête à l'autre.
 */
import { describe, expect, it } from 'vitest'
import { formatDataIssues } from './squadDataIssues'
import { getSquadText } from './i18n'

const text = getSquadText('en')

describe('formatDataIssues', () => {
  it('aucune dégradation → aucun message', () => {
    expect(formatDataIssues(undefined, text)).toEqual([])
    expect(formatDataIssues([], text)).toEqual([])
  })

  it('matchs d\'un coéquipier non chargés : le gamertag apparaît dans le message', () => {
    const [msg] = formatDataIssues([{ code: 'teammate_matches', detail: 'Chocoboflor' }], text)
    expect(msg).toContain('Chocoboflor')
  })

  it('heatmap : message distinct de celui des matchs du coéquipier', () => {
    const [tm] = formatDataIssues([{ code: 'teammate_matches', detail: 'Madina97294' }], text)
    const [hm] = formatDataIssues([{ code: 'heatmap_teammate', detail: 'Madina97294' }], text)
    expect(hm).not.toEqual(tm)
    expect(hm).toContain('Madina97294')
  })

  it('codes sans détail : message fixe', () => {
    expect(formatDataIssues([{ code: 'main_team_participants' }], text)[0]).toBe(text.dataIssues.mainTeamParticipants)
    expect(formatDataIssues([{ code: 'map_stats' }], text)[0]).toBe(text.dataIssues.mapStats)
  })

  it('code inconnu (backend plus récent) → message générique, jamais de ligne vide', () => {
    const [msg] = formatDataIssues([{ code: 'nouvelle_cause' }], text)
    expect(msg).toContain('nouvelle_cause')
    expect(msg.trim().length).toBeGreaterThan(0)
  })

  it('every code produces a non-empty message', () => {
    const issues = [
      { code: 'teammate_matches', detail: 'Chocoboflor' },
      { code: 'heatmap_teammate', detail: 'Chocoboflor' },
      { code: 'main_team_participants' },
      { code: 'map_stats' },
    ]
    const msgs = formatDataIssues(issues, text)
    expect(msgs).toHaveLength(4)
    for (const msg of msgs) {
      expect(msg.trim().length).toBeGreaterThan(0)
    }
  })

  it('conserve l\'ordre et le nombre de dégradations', () => {
    const msgs = formatDataIssues(
      [{ code: 'map_stats' }, { code: 'teammate_matches', detail: 'A' }, { code: 'heatmap_teammate', detail: 'B' }],
      text,
    )
    expect(msgs).toHaveLength(3)
    expect(msgs[0]).toBe(text.dataIssues.mapStats)
    expect(msgs[1]).toContain('A')
    expect(msgs[2]).toContain('B')
  })
})
