/**
 * App.test.tsx — the fixture artifact reaches the screen.
 *
 * SMOKE, AND DELIBERATELY ONLY SMOKE. jsdom has no 2D context, so nothing here can
 * assert a pixel; what it CAN prove is the chain that has to hold for the copy to be
 * whole — the raw document crosses the frontier, the canvas mounts against it, and
 * the roster join names the players it has rows for while leaving the one it does not
 * in a group of his own. Drawing itself is covered a layer down, against the
 * recording context (`features/replay/canvasRecording.test.ts`), and what the ARTIFACT
 * has to be true about lives beside it, in `fixtures/replayFixture.test.ts`.
 */
import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { App } from './App'

describe('App', () => {
  it('mounts the replay against the sample artifact', () => {
    const { container } = render(<App />)
    expect(container.querySelector('canvas')).not.toBeNull()
  })

  it('names the players the scoreboard knows, and keeps the unmatched one ungrouped', () => {
    render(<App />)
    expect(screen.getByTitle('Aigle-01')).toBeDefined()
    expect(screen.getByTitle('Cobra-02')).toBeDefined()
    // The film names the fifth player; the archive has no row for him. He keeps his
    // name and his own group, and is never attached to a team.
    expect(screen.getByTitle('Sans-equipe')).toBeDefined()
  })

  it('gives the document the language the reader chose', () => {
    render(<App />)
    expect(document.documentElement.lang).toBe('fr')
    expect(document.title).toBe('LevelUp — Étude')
  })
})
