/**
 * ArchiveBrowser — narrowing a growing archive down to the match somebody wants to watch.
 *
 * THE COMPONENT DECIDES NOTHING. Which rows survive a filter, in what order they are shown, what
 * the archive actually contains to filter BY, and how a row's players group into sides are all
 * `browserLogic.ts`'s, tested there against fixture rows with no network and no React. What is
 * left here is the part only a component can do: hold what the reader has chosen, and render it.
 *
 * FILTERING IS LOCAL, and it is a decision rather than a shortcut. The server can filter — and
 * still does for any other caller — but doing it here means the table answers a keystroke
 * instantly, and it means the rule for "an unknown coverage never satisfies a floor" exists ONCE
 * instead of in SQL and again in TypeScript. It also keeps this server's hands off the archive
 * file, which the hourly capture needs (`useArchive.ts`).
 *
 * THE OPTIONS COME FROM THE ARCHIVE, not from a list of Halo maps: a filter offering a map
 * nobody has captured can only ever empty the table.
 */
import { useMemo, useState } from 'react'

import { Button } from '@/components/ui/button'
import type { Locale } from '@/lib/i18n/locale'

import {
  DEFAULT_SORT,
  EMPTY_FILTER,
  facetsOf,
  filterMatches,
  sortMatches,
  type ArchiveFilter,
  type ArchiveSort,
  type SortKey,
} from '../features/archive/browserLogic'
import type { MatchSummary } from '../features/archive/studyApi'

import { ArchiveTable } from './ArchiveTable'
import { formatShare } from './format'
import { SHELL_TEXT } from './i18n'
import { Notice } from './Notice'

/** The coverage floors offered, as fractions of 1 — ADR 0006's unit, and the server's. */
const COVERAGE_STEPS = [0.5, 0.7, 0.85] as const

interface ArchiveBrowserProps {
  matches: MatchSummary[]
  /** How many rows matched in the archive, which can exceed how many were loaded. */
  total: number
  locale: Locale
}

export function ArchiveBrowser({ matches, total, locale }: ArchiveBrowserProps) {
  const t = SHELL_TEXT[locale]
  const [filter, setFilter] = useState<ArchiveFilter>(EMPTY_FILTER)
  const [sort, setSort] = useState<ArchiveSort>(DEFAULT_SORT)

  const facets = useMemo(() => facetsOf(matches), [matches])
  const shown = useMemo(
    () => sortMatches(filterMatches(matches, filter), sort),
    [matches, filter, sort],
  )

  /** A click on the sorted column reverses it; a click on another one takes it over, descending. */
  const onSort = (key: SortKey) =>
    setSort((current) =>
      current.key === key
        ? { key, direction: current.direction === 'asc' ? 'desc' : 'asc' }
        : { key, direction: 'desc' },
    )

  if (matches.length === 0) {
    // AN EMPTY ARCHIVE IS NOT AN EMPTY TABLE. A grid of headers with nothing under it says
    // "something went wrong"; this says what happened and what to run.
    return <Notice tone="info" title={t.emptyArchive} hint={t.emptyArchiveHint} />
  }

  return (
    <section className="flex flex-col gap-3">
      <Filters filter={filter} onChange={setFilter} facets={facets} locale={locale} />

      <p className="text-xs text-muted-foreground">
        {t.shownOf(shown.length, matches.length)}
        {/* The archive can hold more than one read brings back. Saying so is the difference
            between a filter over the archive and a filter over a page that looks like one. */}
        {total > matches.length ? ` — ${t.truncated(matches.length, total)}` : ''}
      </p>

      {shown.length === 0 ? (
        <Notice tone="info" title={t.emptyFiltered} hint={t.emptyFilteredHint} />
      ) : (
        <ArchiveTable rows={shown} sort={sort} onSort={onSort} locale={locale} />
      )}
    </section>
  )
}

/** The controls, each one a value of `ArchiveFilter` and nothing more. */
function Filters({
  filter,
  onChange,
  facets,
  locale,
}: {
  filter: ArchiveFilter
  onChange: (filter: ArchiveFilter) => void
  facets: { maps: string[]; modes: string[]; players: string[] }
  locale: Locale
}) {
  const t = SHELL_TEXT[locale]
  const set = <K extends keyof ArchiveFilter>(key: K, value: ArchiveFilter[K]) =>
    onChange({ ...filter, [key]: value })

  return (
    <div className="flex flex-wrap items-end gap-2">
      <Choice label={t.filterMap} value={filter.map} onPick={(v) => set('map', v)} options={facets.maps} any={t.filterAny} />
      <Choice label={t.filterMode} value={filter.mode} onPick={(v) => set('mode', v)} options={facets.modes} any={t.filterAny} />
      <Choice
        label={t.filterPlayer}
        value={filter.player}
        onPick={(v) => set('player', v)}
        options={facets.players}
        any={t.filterAny}
      />
      <Field label={t.filterFrom}>
        <input
          type="date"
          value={filter.from}
          onChange={(e) => set('from', e.currentTarget.value)}
          className={FIELD_CLASS}
        />
      </Field>
      <Field label={t.filterTo}>
        <input
          type="date"
          value={filter.to}
          onChange={(e) => set('to', e.currentTarget.value)}
          className={FIELD_CLASS}
        />
      </Field>
      <Field label={t.filterCoverage}>
        <select
          value={filter.minCoverage === null ? '' : String(filter.minCoverage)}
          onChange={(e) => set('minCoverage', e.currentTarget.value === '' ? null : Number(e.currentTarget.value))}
          className={FIELD_CLASS}
        >
          <option value="">{t.filterAny}</option>
          {COVERAGE_STEPS.map((step) => (
            <option key={step} value={step}>
              {formatShare(step, locale)}
            </option>
          ))}
        </select>
      </Field>
      <Button variant="outline" size="sm" onClick={() => onChange(EMPTY_FILTER)}>
        {t.filterClear}
      </Button>
    </div>
  )
}

const FIELD_CLASS = 'h-9 rounded-md border border-border bg-card px-2 text-sm text-foreground'

/** A labelled control. The label is a real `<label>`, so clicking it reaches the field. */
function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="flex flex-col gap-1 text-xs text-muted-foreground">
      {label}
      {children}
    </label>
  )
}

/** One value out of what the archive actually holds, or all of them. */
function Choice({
  label,
  value,
  onPick,
  options,
  any,
}: {
  label: string
  value: string
  onPick: (value: string) => void
  options: string[]
  any: string
}) {
  return (
    <Field label={label}>
      <select value={value} onChange={(e) => onPick(e.currentTarget.value)} className={FIELD_CLASS}>
        <option value="">{any}</option>
        {options.map((option) => (
          <option key={option} value={option}>
            {option}
          </option>
        ))}
      </select>
    </Field>
  )
}
