/**
 * HomeScreen — the archive, browsable, and the way into any match in it.
 *
 * IT USED TO BE A FIELD AND A SAMPLE LINK, and it said so: listing the archive was its own
 * piece of work, and until it existed the way in was an identifier the reader already had. The
 * table is that work. What the field bought — opening a match by an id from
 * `study-archiver status`, from a shell, from a match page — it still buys, so it stays
 * underneath rather than being replaced: it is the one way in that survives an archive too big
 * for one read (cf. `ARCHIVE_PAGE_LIMIT`), and it costs a form.
 *
 * IT STILL DOES NOT VALIDATE THE IDENTIFIER. Two forms are legitimate (the short film id and the
 * full match id) and the server is the one that knows which matches exist; a client-side rule
 * about their shape would be a second, weaker copy of that knowledge, and it would reject a form
 * the server accepts on the day the archive learns a third one. What the reader types is what
 * gets asked for, and "no archived match under that identifier" is a complete answer.
 *
 * THE FOUR STATES OF THE TABLE ARE THE FOUR STATES OF THE ARCHIVE, and they are told apart for
 * the same reason the match screen tells its own apart: a capture holding the database is not a
 * broken server, and an empty archive is not either.
 */
import { useState } from 'react'

import { Button } from '@/components/ui/button'
import type { Locale } from '@/lib/i18n/locale'

import { useArchive } from '../features/archive/useArchive'

import { ArchiveBrowser } from './ArchiveBrowser'
import { SHELL_TEXT } from './i18n'
import { Notice } from './Notice'
import { matchHref, sampleHref } from './route'

interface HomeScreenProps {
  locale: Locale
}

export function HomeScreen({ locale }: HomeScreenProps) {
  const t = SHELL_TEXT[locale]
  const { screen, reload } = useArchive()

  const retry = (
    <Button variant="outline" size="sm" onClick={reload}>
      {t.retry}
    </Button>
  )

  return (
    <div className="flex flex-col gap-4">
      <p className="max-w-[70ch] text-sm text-muted-foreground">{t.homeIntro}</p>

      <h2 className="text-sm font-medium">{t.browserTitle}</h2>
      {screen.kind === 'loading' && <p className="text-xs text-muted-foreground">{t.loading}</p>}
      {screen.kind === 'ready' && (
        <ArchiveBrowser matches={screen.matches} total={screen.total} locale={locale} />
      )}
      {screen.kind === 'busy' && (
        // Not a fault, and not painted like one: a capture is writing, and it will finish.
        <Notice tone="info" title={t.busy} hint={t.busyHint} action={retry} />
      )}
      {screen.kind === 'changed' && (
        <Notice tone="info" title={t.archiveChanged} hint={t.archiveChangedHint} action={retry} />
      )}
      {screen.kind === 'failed' && (
        <Notice
          tone="destructive"
          title={t.failed}
          hint={`${t.failedHint} (${screen.message})`}
          action={retry}
        />
      )}

      <OpenById locale={locale} />

      <p className="text-xs">
        <a href={sampleHref()} className="text-primary underline-offset-4 hover:underline">
          {t.sampleLink}
        </a>
      </p>
    </div>
  )
}

/** OpenById — the way in that needs no table, for an identifier the reader already holds. */
function OpenById({ locale }: { locale: Locale }) {
  const t = SHELL_TEXT[locale]
  const [entry, setEntry] = useState('')
  const trimmed = entry.trim()

  return (
    <section className="flex flex-col gap-2 border-t border-border pt-3">
      <h2 className="text-sm font-medium">{t.openByIdTitle}</h2>
      <form
        className="flex flex-wrap items-end gap-2"
        onSubmit={(e) => {
          e.preventDefault()
          if (trimmed !== '') window.location.hash = matchHref(trimmed)
        }}
      >
        <label className="flex flex-col gap-1 text-xs text-muted-foreground">
          {t.openLabel}
          <input
            value={entry}
            onChange={(e) => setEntry(e.currentTarget.value)}
            placeholder={t.openPlaceholder}
            spellCheck={false}
            autoComplete="off"
            className="h-9 w-64 rounded-md border border-border bg-card px-2 font-mono text-sm text-foreground"
          />
        </label>
        <Button type="submit" size="sm" disabled={trimmed === ''}>
          {t.openSubmit}
        </Button>
      </form>
      <p className="max-w-[70ch] text-xs text-muted-foreground">{t.openHint}</p>
    </section>
  )
}
