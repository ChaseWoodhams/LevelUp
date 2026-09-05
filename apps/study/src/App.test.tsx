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
import { fireEvent, render, screen } from '@testing-library/react'
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

describe('the landing screen', () => {
  it('offers a way in rather than opening on data about no match', () => {
    render(<App />)
    expect(screen.getByLabelText('Identifiant du match')).toBeDefined()
    expect(document.querySelector('canvas')).toBeNull()
  })

  it('gives the document the language the reader chose', () => {
    render(<App />)
    expect(document.documentElement.lang).toBe('fr')
    expect(document.title).toBe('LevelUp — Étude')
  })

  it('follows the reader into English, page title included', () => {
    render(<App />)
    fireEvent.click(screen.getByRole('button', { name: 'en' }))
    expect(document.documentElement.lang).toBe('en')
    expect(document.title).toBe('LevelUp — Study')
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
    expect(screen.getByLabelText('Chronomètre du match').textContent).toContain('0:26')
  })

  it('leaves the scrubber its own arrow keys', () => {
    go('#/sample')
    render(<App />)
    // A range input steps itself on an arrow. Acting on it here too would be two seeks for
    // one press.
    fireEvent.keyDown(screen.getByLabelText('Temps de match'), { key: 'ArrowRight' })
    expect(screen.getByLabelText('Chronomètre du match').textContent).toContain('0:00')
  })

  it('still jumps on a key the scrubber has no use for', () => {
    go('#/sample')
    render(<App />)
    // Treating the scrubber as a text field — which its tag alone suggests — would kill every
    // shortcut for as long as the reader had touched it.
    fireEvent.keyDown(screen.getByLabelText('Temps de match'), { key: '.' })
    expect(screen.getByLabelText('Chronomètre du match').textContent).toContain('0:26')
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
