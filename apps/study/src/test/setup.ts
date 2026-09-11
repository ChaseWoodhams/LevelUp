/**
 * Vitest setup — the two browser APIs jsdom does not implement and the replay canvas
 * asks for on mount.
 *
 * Neither is faked to produce a result a test then asserts on: the observer reports a
 * fixed width so the component gets past `renderWidth === 0` and actually runs its
 * layout work, and `matchMedia` never matches, which is the desktop default. Anything
 * a test wants to prove about drawing is proved against the recording context in
 * `canvasRecording.test.ts`, not against these stubs.
 *
 * jsdom's `getContext('2d')` returns null and the replay components already treat that
 * as "nothing to draw on" — no canvas backend is installed here on purpose.
 */

/** Width handed to every observed element, in CSS pixels. */
const OBSERVED_WIDTH = 900
const OBSERVED_HEIGHT = 480

if (typeof globalThis.ResizeObserver === 'undefined') {
  class ResizeObserverStub implements ResizeObserver {
    private readonly callback: ResizeObserverCallback

    constructor(callback: ResizeObserverCallback) {
      this.callback = callback
    }

    observe(target: Element): void {
      const rect = { width: OBSERVED_WIDTH, height: OBSERVED_HEIGHT } as DOMRectReadOnly
      this.callback([{ target, contentRect: rect } as ResizeObserverEntry], this)
    }

    unobserve(): void {}

    disconnect(): void {}
  }
  globalThis.ResizeObserver = ResizeObserverStub
}

if (typeof window !== 'undefined' && typeof window.matchMedia !== 'function') {
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    configurable: true,
    value: (query: string): MediaQueryList =>
      ({
        matches: false,
        media: query,
        onchange: null,
        addEventListener: () => {},
        removeEventListener: () => {},
        addListener: () => {},
        removeListener: () => {},
        dispatchEvent: () => false,
      }) as unknown as MediaQueryList,
  })
}
