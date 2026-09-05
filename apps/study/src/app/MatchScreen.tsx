/**
 * MatchScreen — one archived match, from its identifier in the URL to a map on screen.
 *
 * THE COMPONENT IS A SWITCH AND NOTHING ELSE. Every decision — which failure this is, whether
 * the artifact is readable, what the roster looks like — was made at the fetch boundary
 * (`studyApi.ts`) and arrives here as one tagged value. What is left is choosing the words,
 * which is the one thing a component is the right place for.
 *
 * FIVE OUTCOMES THAT ARE NOT A REPLAY, and they are five because they send the reader to five
 * different places: an unknown identifier is a typo or an uncaptured match, a missing artifact
 * is a rebuild, a busy archive is a wait, an unreachable server is a process to start, and an
 * unreadable schema version is a format that has moved on. A shared "something went wrong"
 * would be wrong four times out of five.
 */
import { Button } from '@/components/ui/button'
import type { Locale } from '@/lib/i18n/locale'

import { SUPPORTED_SCHEMA_VERSION } from '../features/archive/schemaVersion'
import { useArchivedMatch } from '../features/archive/useArchivedMatch'
import { ReplayViewer } from '../features/viewer/ReplayViewer'

import { SHELL_TEXT } from './i18n'
import { Notice } from './Notice'

interface MatchScreenProps {
  matchId: string
  locale: Locale
}

export function MatchScreen({ matchId, locale }: MatchScreenProps) {
  const t = SHELL_TEXT[locale]
  const { screen, reload } = useArchivedMatch(matchId)

  const retry = (
    <Button variant="outline" size="sm" onClick={reload}>
      {t.retry}
    </Button>
  )

  switch (screen.kind) {
    case 'loading':
      return <p className="text-xs text-muted-foreground">{t.loading}</p>
    case 'ready':
      return (
        <>
          <p className="font-mono text-xs text-muted-foreground">{matchId}</p>
          {/* KEYED ON THE MATCH so the playback state cannot outlive the document it belongs
              to: a frame past the end of a shorter replay, or a focus on a xuid who is not in
              this match. The loading state already unmounts the viewer between two matches —
              the key is what stops that from being the only thing holding the invariant. */}
          <ReplayViewer
            key={matchId}
            doc={screen.doc}
            scoreboard={screen.scoreboard}
            locale={locale}
          />
        </>
      )
    case 'unsupported':
      // NOTHING IS DRAWN, and the version that was found is named: without it the reader
      // cannot tell an artifact from an older builder apart from one from a newer one, which
      // is the difference between rebuilding the artifact and updating this viewer.
      return (
        <Notice
          tone="warning"
          title={t.unsupported(screen.version, SUPPORTED_SCHEMA_VERSION)}
          hint={t.unsupportedHint}
        />
      )
    case 'no-artifact':
      return <Notice tone="warning" title={t.noArtifact} hint={t.noArtifactHint} action={retry} />
    case 'not-found':
      return <Notice tone="warning" title={t.notFound} hint={t.notFoundHint} />
    case 'busy':
      // Not a fault, and not painted like one: a capture is writing, and it will finish.
      return <Notice tone="info" title={t.busy} hint={t.busyHint} action={retry} />
    case 'failed':
      return (
        <Notice
          tone="destructive"
          title={t.failed}
          hint={`${t.failedHint} (${screen.message})`}
          action={retry}
        />
      )
  }
}
