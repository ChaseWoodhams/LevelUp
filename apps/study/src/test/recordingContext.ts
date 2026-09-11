/**
 * recordingContext.ts — A CANVAS THAT EXECUTES NOTHING AND REMEMBERS EVERYTHING.
 *
 * The drawing code of this viewer never READS the canvas: it writes into it. So a context that
 * simply pushes `{op, args}` observes everything the renderer produces, with no `node-canvas`,
 * no jsdom canvas backend, and no reference image. The idea is the copied
 * `features/replay/canvasRecording.test.ts`'s — that file is byte-identical to `apps/web` and
 * exports nothing, so the helper is reproduced here for this app's own layers rather than the
 * copy being edited (which its drift guard forbids).
 *
 * WHAT A TEST BUILT ON THIS PROVES, AND WHAT IT DOES NOT: the geometry EMITTED — how many
 * strokes, in what order, at what widths and opacities. Never a pixel. A pixel assertion would
 * be a test of anti-aliasing and would break on every engine change without saying anything
 * about the rendering.
 */

/** One recorded operation: a method name (or `set <property>`) and its arguments. */
export interface CanvasOp {
  op: string
  args: unknown[]
}

export interface Recording {
  ops: CanvasOp[]
  ctx: CanvasRenderingContext2D
}

/**
 * recordingContext returns an inert 2D context and the log it writes to.
 *
 * The one operation that READS is `createRadialGradient`, whose result the aim cone calls
 * `addColorStop` on; it answers with an inert token and the call stays in the trace.
 */
export function recordingContext(): Recording {
  const ops: CanvasOp[] = []
  const state: Record<string, unknown> = {}
  const proxy = new Proxy(
    {},
    {
      get(_t, prop) {
        if (typeof prop !== 'string') return undefined
        if (prop === 'createRadialGradient') {
          return (...args: unknown[]) => {
            ops.push({ op: prop, args })
            return { addColorStop: (...a: unknown[]) => ops.push({ op: 'addColorStop', args: a }) }
          }
        }
        if (prop in state) return state[prop]
        return (...args: unknown[]) => {
          ops.push({ op: prop, args })
        }
      },
      set(_t, prop, value) {
        if (typeof prop === 'string') {
          state[prop] = value
          ops.push({ op: `set ${prop}`, args: [value] })
        }
        return true
      },
    },
  )
  return { ops, ctx: proxy as unknown as CanvasRenderingContext2D }
}

/** countOf gives how many times an operation was emitted. */
export function countOf(ops: CanvasOp[], op: string): number {
  return ops.filter((o) => o.op === op).length
}

/** valuesOf gives the successive values assigned to a property (globalAlpha, lineWidth…). */
export function valuesOf(ops: CanvasOp[], prop: string): number[] {
  return ops.filter((o) => o.op === `set ${prop}`).map((o) => o.args[0] as number)
}

/** stringsOf gives the successive values assigned to a string property (strokeStyle…). */
export function stringsOf(ops: CanvasOp[], prop: string): string[] {
  return ops.filter((o) => o.op === `set ${prop}`).map((o) => String(o.args[0]))
}
