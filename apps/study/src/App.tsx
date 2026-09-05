/**
 * App — the study tool's shell: the language, the way home, and which screen the URL names.
 *
 * THREE SCREENS AND NO MORE STATE THAN THAT. The landing screen, one archived match, and the
 * hand-written sample. Everything about a match — fetching it, ruling on its schema version,
 * driving its playback — belongs to the screen that shows it; what lives here is the chrome
 * that is the same on all three.
 *
 * THE SAMPLE IS A ROUTE AND NOT A DEFAULT. Until this ticket the app opened straight onto the
 * fixture, which was right while there was nothing else to open. Now that a real artifact can
 * be fetched, making the hand-written one the first thing on screen would be a tool that
 * greets its reader with data about no match at all. It keeps its own address, because a
 * viewer that cannot be looked at without a captured film and a running server is a viewer
 * nobody can review.
 */
import { useEffect, useMemo, useState } from 'react'

import { Toggle } from '@/components/ui/controls'
import { KNOWN_LOCALES, type Locale } from '@/lib/i18n/locale'

import { HomeScreen } from './app/HomeScreen'
import { SHELL_TEXT } from './app/i18n'
import { MatchScreen } from './app/MatchScreen'
import { homeHref, type Route } from './app/route'
import { useRoute } from './app/useRoute'
import { FIXTURE_REPLAY_DOCUMENT, FIXTURE_SCOREBOARD } from './features/replay/fixtures/replayFixture'
import { normalizeReplayDocument } from './features/replay/replayNormalize'
import { ReplayViewer } from './features/viewer/ReplayViewer'

export function App() {
  const [locale, setLocale] = useState<Locale>('fr')
  const route = useRoute()
  const t = SHELL_TEXT[locale]

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
          {route.kind !== 'home' && (
            <a href={homeHref()} className="text-xs text-primary underline-offset-4 hover:underline">
              {t.back}
            </a>
          )}
        </div>
        <div className="flex items-center gap-1">
          <span className="mr-1 text-xs text-muted-foreground">{t.localeLabel}</span>
          {KNOWN_LOCALES.map((l) => (
            <Toggle key={l} on={locale === l} onClick={() => setLocale(l)} label={l}>
              {l.toUpperCase()}
            </Toggle>
          ))}
        </div>
      </header>

      <Screen route={route} locale={locale} />
    </div>
  )
}

/** Screen picks the view the URL names. */
function Screen({ route, locale }: { route: Route; locale: Locale }) {
  switch (route.kind) {
    case 'home':
      return <HomeScreen locale={locale} />
    case 'match':
      return <MatchScreen matchId={route.matchId} locale={locale} />
    case 'sample':
      return <SampleScreen locale={locale} />
  }
}

/**
 * SampleScreen draws the hand-written artifact through the very same viewer.
 *
 * THE SAME VIEWER IS THE POINT. A demonstration rendered by a second code path would prove
 * nothing about the one that draws real matches; this one crosses the same normalisation
 * boundary, joins the same roster and drives the same timeline.
 */
function SampleScreen({ locale }: { locale: Locale }) {
  const t = SHELL_TEXT[locale]
  const doc = useMemo(() => normalizeReplayDocument(FIXTURE_REPLAY_DOCUMENT), [])
  return (
    <>
      <div className="flex flex-wrap items-baseline gap-2">
        <span className="rounded-md border border-border px-2 py-0.5 text-xs text-muted-foreground">
          {t.sampleBadge}
        </span>
        <p className="text-xs text-muted-foreground">{t.sampleHint}</p>
      </div>
      <ReplayViewer doc={doc} scoreboard={FIXTURE_SCOREBOARD} locale={locale} />
    </>
  )
}
