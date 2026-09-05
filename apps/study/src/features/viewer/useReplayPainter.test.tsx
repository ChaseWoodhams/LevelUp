/**
 * useReplayPainter.test.tsx — THE WORK THAT MUST NOT HAPPEN TWICE.
 *
 * WHY THIS FILE EXISTS AT ALL, when jsdom cannot draw a single pixel. The expensive parts of
 * this hook are held up by nothing a type checker or a rendering test can see: the floor is
 * rasterised once per document because an effect's dependency keeps its identity across
 * renders, and the animation loop survives a re-render for the same reason. Both properties are
 * one unmemoised object literal away from silently inverting — the screen would look identical,
 * the suite would stay green, and the cost would surface as a replay that stutters and runs
 * slow. That regression was real: splitting this hook out of its component introduced it, and
 * nothing in the suite noticed.
 *
 * So the assertions below are about CALL COUNTS, not pixels. Drawing is covered against the
 * recording context in `features/replay/canvasRecording.test.ts`.
 *
 * THE HARNESS ATTACHES ONLY THE CONTAINER, on purpose. The painter needs a measured container
 * to do any work at all (the `ResizeObserver` stub in `src/test/setup.ts` reports 900 px), but
 * leaving the canvas itself unattached makes `draw` return early — so every
 * `document.createElement('canvas')` counted below is the offscreen floor and nothing else.
 */
import { render } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { FIXTURE_REPLAY_DOCUMENT } from '../replay/fixtures/replayFixture'
import { normalizeReplayDocument } from '../replay/replayNormalize'

import { DEFAULT_TRAIL_WINDOW_MS } from './trailLogic'
import { useReplayPainter, type ReplayPainterOptions } from './useReplayPainter'

/** The fixture carries a reconstructed floor — without one the rasterisation never runs. */
const doc = normalizeReplayDocument(FIXTURE_REPLAY_DOCUMENT)

/**
 * Stable references, hoisted deliberately. A fresh `inks` array per render would change the
 * draw closure by itself and the tests would pass or fail for the wrong reason — which is the
 * very mistake under test, made in the test.
 */
const INKS = doc.tracks.map(() => 'rgb(1, 2, 3)')
const NO_INK = () => null
const NO_REPORT = () => {}
const ANCHOR = { frame: 0, nonce: 0 }

function options(over: Partial<ReplayPainterOptions> = {}): ReplayPainterOptions {
  return {
    doc,
    inks: INKS,
    inkOfSlotAt: NO_INK,
    playing: false,
    speed: 1,
    anchor: ANCHOR,
    onFrameChange: NO_REPORT,
    showAim: true,
    showShield: true,
    showShots: true,
    showGrenades: true,
    trailWindowMs: DEFAULT_TRAIL_WINDOW_MS,
    mapModule: null,
    floor: null,
    ...over,
  }
}

function Harness(props: ReplayPainterOptions) {
  const painter = useReplayPainter(props)
  return <div ref={painter.containerRef} />
}

/** offscreenCanvases counts the canvases the floor rasteriser asks the document for. */
function offscreenCanvases(): () => number {
  const create = document.createElement.bind(document)
  const spy = vi.spyOn(document, 'createElement')
  spy.mockImplementation(((tag: string) => create(tag)) as typeof document.createElement)
  return () => spy.mock.calls.filter(([tag]) => tag === 'canvas').length
}

beforeEach(() => {
  vi.restoreAllMocks()
})

describe('the offscreen floor', () => {
  it('is rasterised once, and not again on a re-render with the same inputs', () => {
    const count = offscreenCanvases()
    const { rerender } = render(<Harness {...options()} />)
    const afterMount = count()
    expect(afterMount).toBeGreaterThan(0)

    // A re-render with identical inputs is the ORDINARY case: it happens every 150 ms while
    // the replay plays, because the canvas reports its frame back into React state.
    rerender(<Harness {...options()} />)
    rerender(<Harness {...options()} />)
    expect(count()).toBe(afterMount)
  })

  it('is not rasterised again when only a layer toggle changes', () => {
    const count = offscreenCanvases()
    const { rerender } = render(<Harness {...options()} />)
    const afterMount = count()

    // The aim layer is drawn over the floor, never into it.
    rerender(<Harness {...options({ showAim: false })} />)
    expect(count()).toBe(afterMount)
  })
})

describe('the playback loop', () => {
  it('is requested once, and survives a re-render with the same inputs', () => {
    const request = vi.spyOn(window, 'requestAnimationFrame').mockReturnValue(1)
    const cancel = vi.spyOn(window, 'cancelAnimationFrame').mockImplementation(() => {})

    const { rerender } = render(<Harness {...options({ playing: true })} />)
    expect(request).toHaveBeenCalledTimes(1)

    // Tearing the loop down and re-requesting it on every render also DROPS the time elapsed
    // between the two, so playback would run slow by an amount that varies with how often
    // React happens to render.
    rerender(<Harness {...options({ playing: true })} />)
    rerender(<Harness {...options({ playing: true })} />)
    expect(cancel).not.toHaveBeenCalled()
    expect(request).toHaveBeenCalledTimes(1)
  })

  it('is torn down when the replay is paused, and reports where it stopped', () => {
    vi.spyOn(window, 'requestAnimationFrame').mockReturnValue(1)
    const cancel = vi.spyOn(window, 'cancelAnimationFrame').mockImplementation(() => {})
    const onFrameChange = vi.fn()

    const { rerender } = render(<Harness {...options({ playing: true, onFrameChange })} />)
    rerender(<Harness {...options({ playing: false, onFrameChange })} />)

    expect(cancel).toHaveBeenCalled()
    // The settling report is unthrottled: at 4x the loop can be six frames past its last one.
    expect(onFrameChange).toHaveBeenCalled()
  })

  it('is never requested while the replay is paused', () => {
    const request = vi.spyOn(window, 'requestAnimationFrame').mockReturnValue(1)
    render(<Harness {...options()} />)
    expect(request).not.toHaveBeenCalled()
  })
})
