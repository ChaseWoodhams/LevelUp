/**
 * App.test.tsx — the three screens, and the chain each of them holds together.
 *
 * SMOKE AND WIRING, DELIBERATELY. jsdom has no 2D context, so nothing here can assert a pixel;
 * what it CAN prove is what has to hold for the app to be one app — the URL choosing a screen,
 * the artifact crossing the frontier and reaching a mounted canvas, the roster join naming the
 * players it has rows for while leaving the one it does not in a group of his own, a keystroke
 * moving the match clock, and a failing server producing words rather than a spinner.
 *
 * Drawing is covered a layer down against the recording context
 * (`features/replay/canvasRecording.test.ts`), the playback rules in
 * `features/viewer/playbackLogic.test.ts`, and the fetch boundary in
 * `features/archive/studyApi.test.ts`.
 */
import { fireEvent, render, screen, within } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { App } from './App'

/** go puts the reader on a screen before the app mounts, the way a link would. */
function go(hash: string): void {
  window.location.hash = hash
}

afterEach(() => {
  go('')
  vi.unstubAllGlobals()
})

/** The archive as the browser's own route answers it. */
function stubArchive(matches: unknown[], total = matches.length): void {
  vi.stubGlobal('fetch', async () =>
    new Response(JSON.stringify({ matches, total }), {
      status: 200,
      headers: { 'content-type': 'application/json' },
    }),
  )
}

const ARCHIVED_MATCH = {
  match_id: '000d5950-8b0e-4a2c-9a1f-1c2d3e4f5a6b',
  short_id: '000d5950',
  played_at: '2026-05-19T20:15:00Z',
  map_name: 'Cliffhanger',
  mode: 'Slayer',
  coverage: 0.857,
  named_lives: 90,
  total_lives: 105,
  tracks: 8,
  points: 4200,
  shots: 519,
  participants: [
    { xuid: '1', gamertag: 'JGtm', team_side: 't0', kills: 15, deaths: 9, assists: 4 },
    { xuid: '2', gamertag: 'Rival', team_side: 't1', kills: 9, deaths: 15, assists: 2 },
  ],
}

describe('the landing screen', () => {
  it('offers a way in rather than opening on data about no match', () => {
    stubArchive([])
    render(<App />)
    expect(screen.getByLabelText('Identifiant du match')).toBeDefined()
    expect(document.querySelector('canvas')).toBeNull()
  })

  it('gives the document the language the reader chose', () => {
    stubArchive([])
    render(<App />)
    expect(document.documentElement.lang).toBe('fr')
    expect(document.title).toBe('LevelUp — Étude')
  })

  it('follows the reader into English, page title included', () => {
    stubArchive([])
    render(<App />)
    fireEvent.click(screen.getByRole('button', { name: 'en' }))
    expect(document.documentElement.lang).toBe('en')
    expect(document.title).toBe('LevelUp — Study')
  })
})

describe('the archive browser', () => {
  it('shows final objective scores, preserving zero instead of substituting kills', async () => {
    stubArchive([{ ...ARCHIVED_MATCH, mode: 'CTF', team0_score: 3, team1_score: 0 }])
    render(<App />)
    expect(await screen.findByRole('columnheader', { name: 'Score' })).toBeDefined()
    expect(within(screen.getByRole('table')).getByText('3 – 0')).toBeDefined()
  })

  it('lists what was archived, and every row leads to its replay', async () => {
    stubArchive([ARCHIVED_MATCH])
    render(<App />)

    const link = await screen.findByRole('link', { name: 'Cliffhanger' })
    expect(link.getAttribute('href')).toBe('#/match/000d5950-8b0e-4a2c-9a1f-1c2d3e4f5a6b')
    // The coverage is read BEFORE the match is opened: that is what the column is for.
    expect(screen.getByText('86 %')).toBeDefined()
    // Scoped to the table: the player's name is also one of the filter's options, which is the
    // point of building the filters out of what the archive actually holds.
    expect(within(screen.getByRole('table')).getByText('JGtm')).toBeDefined()
  })

  it('narrows the table on a filter, without asking the server again', async () => {
    stubArchive([
      ARCHIVED_MATCH,
      { ...ARCHIVED_MATCH, match_id: 'b', short_id: 'bbbb2222', map_name: 'Streets', mode: 'CTF' },
    ])
    render(<App />)

    await screen.findByRole('link', { name: 'Cliffhanger' })
    fireEvent.change(screen.getByLabelText('Mode'), { target: { value: 'CTF' } })
    expect(screen.queryByRole('link', { name: 'Cliffhanger' })).toBeNull()
    expect(screen.getByRole('link', { name: 'Streets' })).toBeDefined()
  })

  it('says an empty archive is empty rather than showing a table with nothing in it', async () => {
    stubArchive([])
    render(<App />)
    expect(await screen.findByText('Aucun match archivé pour l’instant')).toBeDefined()
    expect(document.querySelector('table')).toBeNull()
  })

  it('tells a capture holding the archive apart from a broken server', async () => {
    vi.stubGlobal('fetch', async () =>
      new Response(JSON.stringify({ code: 'archive_busy', message: 'busy', retryable: true }), {
        status: 503,
      }),
    )
    render(<App />)
    expect(await screen.findByText('L’archive est momentanément tenue par une capture')).toBeDefined()
  })
})

describe('the sample artifact', () => {
  it('mounts the replay, through the same viewer a real match uses', () => {
    go('#/sample')
    const { container } = render(<App />)
    expect(container.querySelector('canvas')).not.toBeNull()
  })

  it('names the players the roster knows, and keeps the unmatched one ungrouped', () => {
    go('#/sample')
    render(<App />)
    expect(screen.getByTitle('Aigle-01')).toBeDefined()
    expect(screen.getByTitle('Cobra-02')).toBeDefined()
    // The film names the fifth player; the archive has no row for him. He keeps his
    // name and his own group, and is never attached to a team.
    expect(screen.getByTitle('Sans-equipe')).toBeDefined()
  })

  it('jumps to the first death on the keyboard, and the clock says so', () => {
    go('#/sample')
    render(<App />)
    // The fixture runs 600 frames at 100 ms; the first death closes a life at frame 260.
    fireEvent.keyDown(window, { key: '.' })
    expect(screen.getByLabelText('Chronomètre du rejeu').textContent).toContain('0:26')
  })

  it('leaves the scrubber its own arrow keys', () => {
    go('#/sample')
    render(<App />)
    // A range input steps itself on an arrow. Acting on it here too would be two seeks for
    // one press.
    fireEvent.keyDown(screen.getByLabelText('Temps de match'), { key: 'ArrowRight' })
    expect(screen.getByLabelText('Chronomètre du rejeu').textContent).toContain('0:00')
  })

  it('still jumps on a key the scrubber has no use for', () => {
    go('#/sample')
    render(<App />)
    // Treating the scrubber as a text field — which its tag alone suggests — would kill every
    // shortcut for as long as the reader had touched it.
    fireEvent.keyDown(screen.getByLabelText('Temps de match'), { key: '.' })
    expect(screen.getByLabelText('Chronomètre du rejeu').textContent).toContain('0:26')
  })
})

describe('an archived match', () => {
  it('says which match it is opening while the archive is being read', () => {
    vi.stubGlobal('fetch', () => new Promise<Response>(() => {}))
    go('#/match/000d5950')
    render(<App />)
    expect(screen.getByText('Lecture de l’archive…')).toBeDefined()
  })

  it('answers an unknown identifier with words, not with a spinner', async () => {
    vi.stubGlobal('fetch', async () =>
      new Response(JSON.stringify({ code: 'match_not_found', message: 'nope', retryable: false }), {
        status: 404,
      }),
    )
    go('#/match/deadbeef')
    render(<App />)
    expect(await screen.findByText('Aucun match archivé sous cet identifiant')).toBeDefined()
    expect(document.querySelector('canvas')).toBeNull()
  })

  it('refuses to draw an artifact of a schema version it does not read', async () => {
    vi.stubGlobal('fetch', async () =>
      new Response(JSON.stringify({ schemaVersion: 99, matchId: 'x', frameCount: 1, bounds: {}, tracks: [] }), {
        status: 200,
        headers: { 'content-type': 'application/json' },
      }),
    )
    go('#/match/000d5950')
    render(<App />)
    // The version found is named: it is what tells a stale artifact apart from a stale viewer.
    expect(await screen.findByText(/version de schéma 99/)).toBeDefined()
    expect(document.querySelector('canvas')).toBeNull()
  })
})
