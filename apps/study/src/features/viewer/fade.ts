/**
 * fade.ts — the same colour, quieter.
 *
 * WHY THE VIEWER NEEDS THIS AT ALL. Focusing a player has to leave the others VISIBLE: a fight
 * is two sides, and hiding the other seven turns "follow this player" into "watch someone walk
 * about an empty map". Dimming keeps the context and moves it behind the subject, which is
 * what focus means on a map.
 *
 * IT IS NOT A COLOUR DECISION AND INTRODUCES NO PALETTE. The input is a colour already
 * resolved from a semantic token; all that happens here is arithmetic on its alpha channel, so
 * the repository's rule — a colour that MEANS something comes from a token — is untouched.
 *
 * IT NEVER INVENTS A COLOUR IT COULD NOT READ. An input in a notation this file does not parse
 * comes back unchanged: a player drawn at full strength is a worse outcome than the one
 * intended, but it is still the right colour, where a guess would be neither.
 */

/** `#RGB`, `#RRGGBB`, `#RRGGBBAA` — the notations a resolved token actually arrives in. */
const HEX = /^#([0-9a-f]{3}|[0-9a-f]{6}|[0-9a-f]{8})$/i

/** `rgb(...)` / `rgba(...)`, in either the comma or the space syntax. */
const RGB = /^rgba?\(([^)]+)\)$/i

/**
 * fadeColor multiplies a colour's opacity by `alpha`.
 *
 * MULTIPLIES rather than sets: a colour that already carries an alpha — the eight-digit hex
 * form, or an `rgba()` — is asking to be partly transparent, and overwriting that would make a
 * dimmed marker MORE opaque than the one it was cut from.
 */
export function fadeColor(color: string, alpha: number): string {
  const a = Math.min(Math.max(alpha, 0), 1)
  const parsed = parse(color.trim())
  if (!parsed) return color
  const [r, g, b, own] = parsed
  return `rgba(${r}, ${g}, ${b}, ${round(own * a)})`
}

/** parse reads the channels of a colour, or gives up. */
function parse(color: string): [number, number, number, number] | null {
  const hex = HEX.exec(color)
  if (hex) return fromHex(hex[1])
  const rgb = RGB.exec(color)
  if (rgb) return fromRGB(rgb[1])
  return null
}

function fromHex(body: string): [number, number, number, number] {
  const wide = body.length === 3 ? [...body].map((c) => c + c).join('') : body
  const channel = (i: number) => parseInt(wide.slice(i * 2, i * 2 + 2), 16)
  const alpha = wide.length === 8 ? channel(3) / 255 : 1
  return [channel(0), channel(1), channel(2), alpha]
}

/**
 * fromRGB reads the functional notation, comma-separated or space-separated.
 *
 * A PERCENTAGE CHANNEL IS NOT PARSED, and that is why the result is nullable rather than
 * clamped: `rgb(50% 0% 0%)` read as `50` would be a colour nobody asked for, silently.
 */
function fromRGB(body: string): [number, number, number, number] | null {
  const parts = body.split(/[,/\s]+/).filter((p) => p !== '')
  if (parts.length < 3 || parts.some((p) => p.endsWith('%'))) return null
  const nums = parts.slice(0, 4).map(Number)
  if (nums.some((n) => !Number.isFinite(n))) return null
  return [nums[0], nums[1], nums[2], nums.length > 3 ? nums[3] : 1]
}

/** round keeps the alpha readable in a debugger without pretending to more precision. */
function round(a: number): number {
  return Math.round(a * 1000) / 1000
}
