/**
 * HomeScreen — the way in, until the archive browser exists.
 *
 * A FIELD AND NOT A TABLE, deliberately and temporarily. Listing the archive is its own piece
 * of work (the browser, with its filters on map, mode, player and date); what this screen owes
 * today is a way to open a match whose identifier the reader already has — from
 * `study-archiver status`, from a match page, from the shell they just ran a capture in.
 *
 * IT DOES NOT VALIDATE THE IDENTIFIER. Two forms are legitimate (the short film id and the
 * full match id) and the server is the one that knows which matches exist; a client-side rule
 * about their shape would be a second, weaker copy of that knowledge, and it would reject a
 * form the server accepts on the day the archive learns a third one. What the reader types is
 * what gets asked for, and "no archived match under that identifier" is a complete answer.
 */
import { useState } from 'react'

import { Button } from '@/components/ui/button'
import type { Locale } from '@/lib/i18n/locale'

import { SHELL_TEXT } from './i18n'
import { matchHref, sampleHref } from './route'

interface HomeScreenProps {
  locale: Locale
}

export function HomeScreen({ locale }: HomeScreenProps) {
  const t = SHELL_TEXT[locale]
  const [entry, setEntry] = useState('')
  const trimmed = entry.trim()

  return (
    <section className="flex max-w-[70ch] flex-col gap-4">
      <p className="text-sm text-muted-foreground">{t.homeIntro}</p>

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

      <p className="text-xs text-muted-foreground">{t.openHint}</p>

      <p className="text-xs">
        <a href={sampleHref()} className="text-primary underline-offset-4 hover:underline">
          {t.sampleLink}
        </a>
      </p>
    </section>
  )
}
