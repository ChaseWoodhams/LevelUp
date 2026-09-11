/**
 * keyboard.ts — THE KEY MAP, as a pure function of a keystroke.
 *
 * WHY IT IS NOT INSIDE THE LISTENER. A key map written inline in an effect can only be tested
 * by dispatching synthetic events at a mounted component; written as `keystroke -> action` it
 * is a table, and the two properties that actually break — a modifier combination swallowed
 * from the browser, a digit beyond the roster — are one assertion each.
 *
 * WHAT IS NOT DECIDED HERE: whether the reader is typing. That is a question about the DOM
 * (is the focus in a text field?), and it belongs to the listener that owns the DOM, not to a
 * table over key names.
 */
import { FOCUSABLE_PLAYERS, type PlaybackAction } from './playbackLogic'

/** The part of a keyboard event this map reads. */
export interface Keystroke {
  key: string
  ctrlKey?: boolean
  metaKey?: boolean
  altKey?: boolean
  shiftKey?: boolean
}

/**
 * actionForKey translates a keystroke, or returns null to let the browser have it.
 *
 * ANY MODIFIER MEANS THE KEYSTROKE IS NOT OURS. Ctrl+. and Cmd+1 are browser and window
 * manager shortcuts (tabs, panels); claiming them would break the reader's own tools to save
 * them a mouse move. Shift is the exception — it carries no shortcut of its own on these keys,
 * and it is what turns a one-frame step into a run of them.
 */
export function actionForKey(e: Keystroke): PlaybackAction | null {
  if (e.ctrlKey || e.metaKey || e.altKey) return null
  switch (e.key) {
    case ' ':
    case 'Spacebar':
      return { type: 'toggle' }
    case 'ArrowRight':
      return { type: 'step', delta: e.shiftKey ? STEP_RUN : 1 }
    case 'ArrowLeft':
      return { type: 'step', delta: e.shiftKey ? -STEP_RUN : -1 }
    case ',':
      return { type: 'jumpDeath', direction: -1 }
    case '.':
      return { type: 'jumpDeath', direction: 1 }
    case 'Escape':
    case '0':
      return { type: 'focusXUID', xuid: null }
  }
  const digit = digitOf(e.key)
  return digit === null ? null : { type: 'focus', index: digit - 1 }
}

/**
 * A shifted arrow covers a second of match time at the film's nominal cadence — the step that
 * crosses an engagement rather than a frame of one. Expressed in frames because the axis is
 * frames; at 10 Hz, the cadence the builder writes, that is one second.
 */
const STEP_RUN = 10

/** digitOf reads `1`..`8` — the players a hand can address without moving. */
function digitOf(key: string): number | null {
  if (key.length !== 1) return null
  const n = key.charCodeAt(0) - '0'.charCodeAt(0)
  return n >= 1 && n <= FOCUSABLE_PLAYERS ? n : null
}

/**
 * isTypingTarget says whether a keystroke belongs to a form field rather than to the viewer.
 *
 * The scrubber is why this exists and not only the short-id box: a range input has its own
 * arrow-key behaviour, and a reader who has just dragged it holds the focus. Without this
 * check, an arrow would both move the slider and step the replay — two seeks for one press.
 */
export function isTypingTarget(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) return false
  if (target.isContentEditable) return true
  return ['INPUT', 'TEXTAREA', 'SELECT'].includes(target.tagName)
}

/** The keys a focused button activates on. Both, per the platform — not just Enter. */
const BUTTON_KEYS = [' ', 'Spacebar', 'Enter']

/** The keys a focused slider moves on. Everything else means nothing to it. */
const SLIDER_KEYS = ['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown', 'Home', 'End', 'PageUp', 'PageDown']

/**
 * focusOwnsKey says the focused element handles this key itself, so the viewer must not.
 *
 * ONLY WHAT THE ELEMENT ACTUALLY USES, and that granularity is the point. Three cases, and the
 * one in the middle is the one that gets written wrong:
 *
 *   - a TEXT FIELD owns every key: the reader is typing, and `.` is a full stop, not a jump;
 *   - the SCRUBBER is an `<input>` too, but a range input types nothing. It owns its ARROWS —
 *     without that, one press would both move the slider and step the replay — and nothing
 *     else. Treating it as a text field, which is what the tag alone suggests, silently killed
 *     every shortcut for as long as a reader had touched the scrubber, `Space` included;
 *   - a BUTTON owns `Space` and `Enter`, the two keys that activate it. A reader who has just
 *     clicked "next frame" still holds that button, and claiming `Space` unconditionally would
 *     make the whole transport row unreachable from the keyboard — a shortcut taking away the
 *     thing it was meant to make faster.
 */
export function focusOwnsKey(target: EventTarget | null, key: string): boolean {
  if (!(target instanceof HTMLElement)) return false
  if (isSlider(target)) return SLIDER_KEYS.includes(key)
  if (isTypingTarget(target)) return true
  return target.tagName === 'BUTTON' && BUTTON_KEYS.includes(key)
}

/** isSlider: a range input — the scrubber, and nothing else in this app today. */
function isSlider(target: HTMLElement): boolean {
  return target instanceof HTMLInputElement && target.type === 'range'
}
