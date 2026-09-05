/**
 * ArchiveTable — the archive as rows, and the way into a replay.
 *
 * WHAT A ROW HAS TO CARRY for the table to be worth having: when it was played, on what, by
 * whom, and HOW WELL COVERED IT IS. The last one is the reason this screen exists in front of
 * the viewer rather than behind it — coverage is what separates a match worth studying from one
 * whose data is too thin, and finding that out after opening a match is finding it out too late.
 *
 * PLAIN MARKUP, DELIBERATELY, AND NOT TANSTACK TABLE. The repository's rule ("interactive
 * tables: TanStack Table") is about `apps/web`, whose tables have column resizing, virtualised
 * bodies, pinned columns and per-column filters. This one has six columns, one page of rows
 * already in memory, and sorting that is a pure function tested in `browserLogic.ts` — a table
 * library here would add a dependency to this app's otherwise React-and-nothing bundle in order
 * to re-implement `sortMatches`.
 *
 * NO COLOUR LITERAL: the team chips take the same comparison tokens the roster panel and the map
 * markers use, through `teamTokenAt`, so one player is one colour on every screen of this tool.
 */
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import type { Locale } from '@/lib/i18n/locale'

import { teamTalliesOf, type ArchiveSort, type SortKey } from '../features/archive/browserLogic'
import type { MatchSummary } from '../features/archive/studyApi'
import { teamTokenAt } from '../features/viewer/teamColors'

import { formatInstant, formatShare } from './format'
import { SHELL_TEXT } from './i18n'
import { matchHref } from './route'

interface ArchiveTableProps {
  rows: MatchSummary[]
  sort: ArchiveSort
  onSort: (key: SortKey) => void
  locale: Locale
}

export function ArchiveTable({ rows, sort, onSort, locale }: ArchiveTableProps) {
  const t = SHELL_TEXT[locale]
  return (
    <div className="overflow-x-auto rounded-lg border border-border">
      <table className="w-full border-collapse text-sm">
        <thead>
          <tr className="border-b border-border text-left text-xs text-muted-foreground">
            <SortableHeader label={t.colDate} column="played_at" sort={sort} onSort={onSort} />
            <SortableHeader label={t.colMap} column="map_name" sort={sort} onSort={onSort} />
            <SortableHeader label={t.colMode} column="mode" sort={sort} onSort={onSort} />
            <th scope="col" className="px-3 py-2 font-medium">
              {t.colPlayers}
            </th>
            <th scope="col" className="px-3 py-2 font-medium" title={t.scoreHint}>
              {t.colScore}
            </th>
            <SortableHeader
              label={t.colCoverage}
              column="coverage"
              sort={sort}
              onSort={onSort}
              title={t.coverageHint}
            />
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <MatchRow key={row.match_id} row={row} locale={locale} />
          ))}
        </tbody>
      </table>
    </div>
  )
}

/** A column header that also says which way the table is sorted, in text and to a screen reader. */
function SortableHeader({
  label,
  column,
  sort,
  onSort,
  title,
}: {
  label: string
  column: SortKey
  sort: ArchiveSort
  onSort: (key: SortKey) => void
  title?: string
}) {
  const active = sort.key === column
  return (
    <th
      scope="col"
      className="px-3 py-2 font-medium"
      aria-sort={active ? (sort.direction === 'asc' ? 'ascending' : 'descending') : 'none'}
      title={title}
    >
      <button type="button" onClick={() => onSort(column)} className="hover:text-foreground">
        {label}
        {/* The arrow is decorative: `aria-sort` above is what a screen reader reads. */}
        <span aria-hidden>{active ? (sort.direction === 'asc' ? ' ↑' : ' ↓') : ''}</span>
      </button>
    </th>
  )
}

/**
 * MatchRow — one archived match.
 *
 * THE WHOLE ROW OPENS THE REPLAY, and the map cell also holds a real link. The row is the target
 * a mouse wants; the link is the one a keyboard and a screen reader need, and it is what makes
 * "open in a new tab" work. A row that was only an `onClick` would be neither.
 */
function MatchRow({ row, locale }: { row: MatchSummary; locale: Locale }) {
  const t = SHELL_TEXT[locale]
  const href = matchHref(row.short_id)
  const tallies = teamTalliesOf(row)
  return (
    <tr
      className="cursor-pointer border-b border-border last:border-0 hover:bg-muted"
      onClick={() => {
        window.location.hash = href
      }}
    >
      <td className="whitespace-nowrap px-3 py-2 text-muted-foreground">
        {formatInstant(row.played_at, locale) ?? t.unknownValue}
      </td>
      <td className="px-3 py-2">
        <a href={href} className="text-primary underline-offset-4 hover:underline">
          {row.map_name || t.unknownValue}
        </a>
      </td>
      <td className="px-3 py-2 text-muted-foreground">{row.mode || t.unknownValue}</td>
      <td className="px-3 py-2">
        <div className="flex flex-wrap gap-1">
          {tallies.map((tally, index) =>
            tally.players.map((p) => (
              <span
                key={p.xuid}
                className="rounded-md border border-border px-1.5 py-0.5 text-xs"
                style={{ color: tokenCssVar(teamTokenAt(index)) }}
              >
                {p.gamertag || p.xuid}
              </span>
            )),
          )}
        </div>
      </td>
      <td className="whitespace-nowrap px-3 py-2 font-mono text-xs" title={t.scoreHint}>
        {tallies.map((tally, index) => (
          <span key={tally.side ?? 'none'} style={{ color: tokenCssVar(teamTokenAt(index)) }}>
            {index > 0 ? ' – ' : ''}
            {tally.kills}
          </span>
        ))}
      </td>
      <td className="whitespace-nowrap px-3 py-2">
        <Coverage value={row.coverage} locale={locale} />
      </td>
    </tr>
  )
}

/**
 * Coverage — a share, or the word for not knowing.
 *
 * An absent coverage is NOT rendered as 0 %: the artifact reported no lives, so there was
 * nothing to attach anything to. Painting it as zero would put a real measurement's face on the
 * absence of one, and it is exactly the row a reader is deciding whether to open.
 */
function Coverage({ value, locale }: { value: number | undefined; locale: Locale }) {
  const t = SHELL_TEXT[locale]
  if (value === undefined) {
    return (
      <span className="text-muted-foreground" title={t.coverageUnknownHint}>
        {t.coverageUnknown}
      </span>
    )
  }
  return <span title={t.coverageHint}>{formatShare(value, locale)}</span>
}

