/**
 * keyboard.test.ts — the key map, read as the table it is.
 */
import { describe, expect, it } from 'vitest'

import { actionForKey, focusOwnsKey, isTypingTarget } from './keyboard'

describe('actionForKey', () => {
  it('maps the four shortcuts the tool promises', () => {
    expect(actionForKey({ key: ' ' })).toEqual({ type: 'toggle' })
    expect(actionForKey({ key: 'ArrowRight' })).toEqual({ type: 'step', delta: 1 })
    expect(actionForKey({ key: 'ArrowLeft' })).toEqual({ type: 'step', delta: -1 })
    expect(actionForKey({ key: '.' })).toEqual({ type: 'jumpDeath', direction: 1 })
    expect(actionForKey({ key: ',' })).toEqual({ type: 'jumpDeath', direction: -1 })
  })

  it('addresses the eight players by digit, 1 being the first', () => {
    expect(actionForKey({ key: '1' })).toEqual({ type: 'focus', index: 0 })
    expect(actionForKey({ key: '8' })).toEqual({ type: 'focus', index: 7 })
  })

  it('releases the focus on 0 and on Escape', () => {
    expect(actionForKey({ key: '0' })).toEqual({ type: 'focusXUID', xuid: null })
    expect(actionForKey({ key: 'Escape' })).toEqual({ type: 'focusXUID', xuid: null })
  })

  it('makes a shifted arrow a run of frames, not one', () => {
    expect(actionForKey({ key: 'ArrowRight', shiftKey: true })).toEqual({ type: 'step', delta: 10 })
    expect(actionForKey({ key: 'ArrowLeft', shiftKey: true })).toEqual({ type: 'step', delta: -10 })
  })

  it('leaves every modified keystroke to the browser', () => {
    // Ctrl+. and Cmd+1 belong to tabs and panels; claiming them would break the reader's own
    // tools to save them a mouse move.
    expect(actionForKey({ key: '.', ctrlKey: true })).toBeNull()
    expect(actionForKey({ key: '1', metaKey: true })).toBeNull()
    expect(actionForKey({ key: ' ', altKey: true })).toBeNull()
  })

  it('claims nothing else', () => {
    expect(actionForKey({ key: '9' })).toBeNull()
    expect(actionForKey({ key: 'k' })).toBeNull()
    expect(actionForKey({ key: 'Enter' })).toBeNull()
  })
})

describe('isTypingTarget', () => {
  it('yields to a form field, the scrubber included', () => {
    // A range input has its own arrow-key behaviour: without this, one press would move the
    // slider AND step the replay.
    for (const tag of ['input', 'textarea', 'select']) {
      expect(isTypingTarget(document.createElement(tag))).toBe(true)
    }
  })

  it('yields to editable content', () => {
    const el = document.createElement('div')
    el.contentEditable = 'true'
    // jsdom does not implement `isContentEditable` from the attribute; assert on the property
    // the check actually reads.
    Object.defineProperty(el, 'isContentEditable', { value: true })
    expect(isTypingTarget(el)).toBe(true)
  })

  it('claims a keystroke aimed at the page itself', () => {
    expect(isTypingTarget(document.createElement('div'))).toBe(false)
    expect(isTypingTarget(document.createElement('button'))).toBe(false)
    expect(isTypingTarget(null)).toBe(false)
  })
})

describe('focusOwnsKey', () => {
  const button = () => document.createElement('button')
  const input = () => document.createElement('input')

  it('leaves a focused button the two keys that activate it', () => {
    // A reader who just clicked "next frame" still holds that button. Claiming Space would
    // make the whole transport row unreachable from the keyboard.
    expect(focusOwnsKey(button(), ' ')).toBe(true)
    expect(focusOwnsKey(button(), 'Enter')).toBe(true)
  })

  it('keeps the keys a button has no use for', () => {
    expect(focusOwnsKey(button(), '.')).toBe(false)
    expect(focusOwnsKey(button(), 'ArrowRight')).toBe(false)
    expect(focusOwnsKey(button(), '3')).toBe(false)
  })

  it('leaves a text field every key — the reader is typing, and `.` is a full stop', () => {
    expect(focusOwnsKey(input(), 'ArrowRight')).toBe(true)
    expect(focusOwnsKey(input(), '.')).toBe(true)
    expect(focusOwnsKey(input(), ' ')).toBe(true)
  })

  it('leaves the scrubber its arrows, and keeps everything else', () => {
    // The scrubber is an <input> too, but a range input types nothing. Reading it as a text
    // field killed every shortcut — Space included — for as long as a reader had touched it.
    const scrubber = input()
    scrubber.type = 'range'
    expect(focusOwnsKey(scrubber, 'ArrowRight')).toBe(true)
    expect(focusOwnsKey(scrubber, 'Home')).toBe(true)
    expect(focusOwnsKey(scrubber, ' ')).toBe(false)
    expect(focusOwnsKey(scrubber, '.')).toBe(false)
    expect(focusOwnsKey(scrubber, '3')).toBe(false)
  })

  it('claims everything aimed at the page itself', () => {
    expect(focusOwnsKey(document.createElement('div'), ' ')).toBe(false)
    expect(focusOwnsKey(null, ' ')).toBe(false)
  })
})
