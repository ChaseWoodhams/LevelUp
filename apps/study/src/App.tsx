/**
 * App — the study tool's single screen, for now: one artifact, drawn.
 *
 * WHAT THIS IS THE TRACER BULLET FOR. Every piece below except the fixture is code
 * that already runs in the web app's match replay; putting a real artifact through it
 * on a real canvas, inside a separate app, is what proves the copy landed whole — the
 * boundary (`normalizeReplayDocument`), the floor, the layers, the roster join and
 * the coverage banner, all reading the same document at the same frame.
 *
 * The frame is held HERE and not inside the canvas: the canvas animates at screen
 * cadence and publishes the current frame back at a reduced rate, which is what keeps
 * the player cards from re-rendering sixty times a second.
 */
import { useCallback, useEffect, useMemo, useState } from 'react'

import { Button } from '@/components/ui/button'
import { KNOWN_LOCALES, type Locale } from '@/lib/i18n/locale'

import { SHELL_TEXT } from './app/i18n'
import { ReplayCanvas } from './features/replay/ReplayCanvas'
import { ReplayCoverageBanner } from './features/replay/ReplayCoverage'
import { ReplayTeams } from './features/replay/ReplayTeams'
import { FIXTURE_REPLAY_DOCUMENT, FIXTURE_SCOREBOARD } from './features/replay/fixtures/replayFixture'
import { normalizeReplayDocument } from './features/replay/replayNormalize'

export function App() {
  const [locale, setLocale] = useState<Locale>('fr')
  const [frame, setFrame] = useState(0)
  const t = SHELL_TEXT[locale]

  // The artifact crosses the nullability frontier ONCE, exactly where a fetched one
  // will (queryFn, issue #13). Everything downstream reads the `*Ready` shape.
  const doc = useMemo(() => normalizeReplayDocument(FIXTURE_REPLAY_DOCUMENT), [])
  const onFrameChange = useCallback((f: number) => setFrame(f), [])

  // The document itself speaks the chosen language, title and `lang` included. Leaving
  // `lang` frozen at the value in index.html would tell a screen reader to pronounce
  // English content in French for as long as the reader stays on the page.
  useEffect(() => {
    document.title = t.documentTitle
    document.documentElement.lang = locale
  }, [t.documentTitle, locale])

  return (
    <div className="mx-auto flex max-w-[1400px] flex-col gap-3 p-4">
      <header className="flex flex-wrap items-baseline justify-between gap-3">
        <div className="flex items-baseline gap-2">
          <h1 className="text-lg font-semibold">{t.appName}</h1>
          <span className="rounded-md border border-border px-2 py-0.5 text-xs text-muted-foreground">
            {t.fixtureBadge}
          </span>
        </div>
        <div className="flex items-center gap-1">
          <span className="mr-1 text-xs text-muted-foreground">{t.localeLabel}</span>
          {KNOWN_LOCALES.map((l) => (
            <Button
              key={l}
              variant={locale === l ? 'default' : 'ghost'}
              size="sm"
              className="h-7 px-2 text-xs uppercase"
              onClick={() => setLocale(l)}
              aria-pressed={locale === l}
            >
              {l}
            </Button>
          ))}
        </div>
      </header>

      <p className="text-xs text-muted-foreground">{t.fixtureHint}</p>

      <div className="grid gap-3 lg:grid-cols-[minmax(0,3fr)_minmax(0,1fr)]">
        <ReplayCanvas doc={doc} locale={locale} onFrameChange={onFrameChange} />
        <ReplayCoverageBanner doc={doc} locale={locale} />
      </div>

      <section className="flex flex-col gap-2">
        <h2 className="text-sm font-medium">{t.rosterTitle}</h2>
        <ReplayTeams doc={doc} scoreboard={FIXTURE_SCOREBOARD} frame={frame} locale={locale} />
      </section>
    </div>
  )
}
