import data, { type Emoji, type EmojiMartData } from '@emoji-mart/data'

export type CustomEmoji = {
  name: string
  url: string
  displayName?: string
}

export type EmojiCatalogItem = {
  id: string
  name: string
  native: string
  keywords: string[]
  emoticons: string[]
}

export type EmojiCategory = {
  id: string
  label: string
  emojis: EmojiCatalogItem[]
}

export type ResolvedEmoji =
  | { kind: 'unicode'; id: string; native: string }
  | { kind: 'custom'; id: string; url: string; displayName: string }

export type EmoticonEntry = {
  token: string
  id: string
}

const mart = data as EmojiMartData

const CATEGORY_LABELS: Record<string, string> = {
  people: 'Smileys & people',
  nature: 'Animals & nature',
  foods: 'Food & drink',
  activity: 'Activity',
  places: 'Travel & places',
  objects: 'Objects',
  symbols: 'Symbols',
  flags: 'Flags',
}

const SKIN_TONE_RE = /^skin-tone-([2-6])$/i

/** Preferred emoji-mart ids when several faces share the same emoticon. */
const CANONICAL_EMOTICON_IDS: Record<string, string> = {
  ':)': 'smiley',
  '=)': 'smiley',
  '=-)': 'smiley',
  ':D': 'grinning',
  ':-D': 'smile',
  ':P': 'stuck_out_tongue',
  ':p': 'stuck_out_tongue',
  ':-P': 'stuck_out_tongue',
  ':-p': 'stuck_out_tongue',
  ':b': 'stuck_out_tongue',
  ':-b': 'stuck_out_tongue',
  '<3': 'heart',
  '</3': 'broken_heart',
  ':(': 'disappointed',
  ":'(": 'cry',
}

function emojiById(id: string): Emoji | undefined {
  return mart.emojis[id]
}

function catalogItemFromEmoji(emoji: Emoji): EmojiCatalogItem | null {
  const native = emoji.skins[0]?.native
  if (!native) return null
  return {
    id: emoji.id,
    name: emoji.name,
    native,
    keywords: emoji.keywords ?? [],
    emoticons: emoji.emoticons ?? [],
  }
}

/** Resolve alias or id to the canonical emoji-mart id. */
export function canonicalEmojiId(name: string): string | undefined {
  const key = name.toLowerCase()
  if (mart.emojis[key]) return key
  const aliased = mart.aliases?.[key]
  if (aliased && mart.emojis[aliased]) return aliased
  return undefined
}

/** Slack-style shortcode for a standard emoji, optionally with skin tone. */
export function toSlackShortcode(id: string, skinTone = 1): string {
  const base = `:${id}:`
  if (skinTone <= 1 || skinTone > 6) return base
  return `${base}:skin-tone-${skinTone}:`
}

export function isSkinToneShortcode(name: string): number | null {
  const match = SKIN_TONE_RE.exec(name)
  if (!match) return null
  return Number(match[1])
}

export function resolveEmoji(
  name: string,
  options?: {
    skinTone?: number
    customByName?: Map<string, CustomEmoji>
  },
): ResolvedEmoji | null {
  const key = name.toLowerCase()
  const custom = options?.customByName?.get(key)
  if (custom) {
    return {
      kind: 'custom',
      id: custom.name,
      url: custom.url,
      displayName: custom.displayName ?? custom.name,
    }
  }

  const id = canonicalEmojiId(key)
  if (!id) return null
  const emoji = emojiById(id)
  if (!emoji) return null

  const skinTone = options?.skinTone ?? 1
  const skinIndex = Math.min(Math.max(skinTone, 1), emoji.skins.length) - 1
  const native = emoji.skins[skinIndex]?.native ?? emoji.skins[0]?.native
  if (!native) return null

  return { kind: 'unicode', id, native }
}

let catalogCache: EmojiCategory[] | null = null
let emoticonEntriesCache: EmoticonEntry[] | null = null
let emoticonByToken: Map<string, string> | null = null
let emoticonByTokenLower: Map<string, string> | null = null

function indexEmoticons() {
  if (emoticonEntriesCache && emoticonByToken && emoticonByTokenLower) {
    return
  }

  const byToken = new Map<string, string>()
  for (const emoji of Object.values(mart.emojis)) {
    for (const token of emoji.emoticons ?? []) {
      if (!token || byToken.has(token)) continue
      byToken.set(token, emoji.id)
    }
  }
  for (const [token, id] of Object.entries(CANONICAL_EMOTICON_IDS)) {
    if (mart.emojis[id]) byToken.set(token, id)
  }

  const byTokenLower = new Map<string, string>()
  for (const [token, id] of byToken) {
    const lower = token.toLowerCase()
    if (!byTokenLower.has(lower)) byTokenLower.set(lower, id)
  }

  emoticonByToken = byToken
  emoticonByTokenLower = byTokenLower
  emoticonEntriesCache = [...byToken.entries()]
    .map(([token, id]) => ({ token, id }))
    .sort((a, b) => b.token.length - a.token.length || a.token.localeCompare(b.token))
}

export function getEmoticonEntries(): EmoticonEntry[] {
  indexEmoticons()
  return emoticonEntriesCache ?? []
}

export function resolveEmoticon(token: string): string | undefined {
  indexEmoticons()
  return (
    emoticonByToken?.get(token) ??
    emoticonByTokenLower?.get(token.toLowerCase())
  )
}

export function matchEmoticonAt(
  text: string,
  index: number,
): EmoticonEntry | null {
  if (index < 0 || index >= text.length) return null
  if (index > 0 && !/\s/.test(text[index - 1] ?? '')) return null

  for (const entry of getEmoticonEntries()) {
    if (!text.startsWith(entry.token, index)) continue
    const end = index + entry.token.length
    const next = text[end]
    if (next !== undefined && /[A-Za-z0-9]/.test(next)) continue
    return entry
  }

  const remaining = text.slice(index)
  for (const entry of getEmoticonEntries()) {
    const slice = remaining.slice(0, entry.token.length)
    if (slice.toLowerCase() !== entry.token.toLowerCase()) continue
    const end = index + entry.token.length
    const next = text[end]
    if (next !== undefined && /[A-Za-z0-9]/.test(next)) continue
    return { token: slice, id: entry.id }
  }

  return null
}

const SHORTCODE_AT_START_RE = /^:([a-zA-Z0-9_+-]+):/
const graphemeSegmenter = new Intl.Segmenter(undefined, {
  granularity: 'grapheme',
})

function isEmojiGrapheme(grapheme: string): boolean {
  if (/\p{Extended_Pictographic}/u.test(grapheme)) return true
  if (/^\p{Regional_Indicator}{2}$/u.test(grapheme)) return true
  if (/^[#*0-9]\uFE0F?\u20E3$/u.test(grapheme)) return true
  return false
}

function firstGrapheme(text: string): string {
  const first = graphemeSegmenter.segment(text)[Symbol.iterator]().next().value
  return first?.segment ?? text[0] ?? ''
}

/** True when the body is only emoji, shortcodes, emoticons, and whitespace. */
export function isEmojiOnlyMessage(
  body: string,
  options?: { customByName?: Map<string, CustomEmoji> },
): boolean {
  const text = body.trim()
  if (!text) return false

  let index = 0
  let found = false
  while (index < text.length) {
    const ch = text[index]
    if (ch !== undefined && /\s/.test(ch)) {
      index += 1
      continue
    }

    const shortcode = SHORTCODE_AT_START_RE.exec(text.slice(index))
    if (shortcode?.[1] && resolveEmoji(shortcode[1], options)) {
      found = true
      index += shortcode[0].length
      const skin = SHORTCODE_AT_START_RE.exec(text.slice(index))
      if (skin?.[1] && isSkinToneShortcode(skin[1]) !== null) {
        index += skin[0].length
      }
      continue
    }

    const emoticon = matchEmoticonAt(text, index)
    if (emoticon) {
      found = true
      index += emoticon.token.length
      continue
    }

    const grapheme = firstGrapheme(text.slice(index))
    if (grapheme && isEmojiGrapheme(grapheme)) {
      found = true
      index += grapheme.length
      continue
    }

    return false
  }

  return found
}

export function getEmojiCatalog(): EmojiCategory[] {
  if (catalogCache) return catalogCache

  catalogCache = mart.categories.map((category) => ({
    id: category.id,
    label: CATEGORY_LABELS[category.id] ?? category.id,
    emojis: category.emojis
      .map((id) => {
        const emoji = emojiById(id)
        if (!emoji) return null
        return catalogItemFromEmoji(emoji)
      })
      .filter((item): item is EmojiCatalogItem => item !== null),
  }))

  return catalogCache
}

export function getDefaultEmojiSuggestions(limit = 20): EmojiCatalogItem[] {
  const people = getEmojiCatalog().find((category) => category.id === 'people')
  const source = people?.emojis ?? getEmojiCatalog()[0]?.emojis ?? []
  return source.slice(0, limit)
}

function emoticonSearchHaystack(emoji: EmojiCatalogItem): string[] {
  return emoji.emoticons.map((token) => token.toLowerCase())
}

function exactEmoticonMatch(emoji: EmojiCatalogItem, query: string): boolean {
  const q = query.toLowerCase()
  const withColon = q.startsWith(':') ? q : `:${q}`
  return emoji.emoticons.some((token) => {
    const lower = token.toLowerCase()
    return lower === q || lower === withColon || lower.slice(1) === q
  })
}

export function searchEmojiCatalog(
  query: string,
  limit = 80,
): EmojiCatalogItem[] {
  const q = query.trim().toLowerCase()
  if (!q) return []

  const withColon = q.startsWith(':') ? q : `:${q}`
  const canonicalId =
    resolveEmoticon(q) ??
    resolveEmoticon(withColon) ??
    resolveEmoticon(`:${q}`)

  const scored: { emoji: EmojiCatalogItem; score: number; order: number }[] = []
  let order = 0
  for (const category of getEmojiCatalog()) {
    for (const emoji of category.emojis) {
      const id = emoji.id.toLowerCase()
      const name = emoji.name.toLowerCase()
      const keywords = emoji.keywords.map((keyword) => keyword.toLowerCase())
      const emoticons = emoticonSearchHaystack(emoji)

      let score = Infinity
      if (canonicalId === emoji.id && exactEmoticonMatch(emoji, q)) {
        score = -1
      } else if (exactEmoticonMatch(emoji, q) || id === q) {
        score = 0
      } else if (id.startsWith(q)) {
        score = 1
      } else if (name.startsWith(q) || emoticons.some((token) => token.startsWith(withColon) || token.startsWith(q))) {
        score = 2
      } else if (
        id.includes(q) ||
        name.includes(q) ||
        keywords.some((keyword) => keyword.includes(q)) ||
        emoticons.some((token) => token.includes(q) || token.includes(withColon))
      ) {
        score = 3
      }
      if (score === Infinity) continue
      scored.push({ emoji, score, order: order++ })
    }
  }

  scored.sort((a, b) => a.score - b.score || a.order - b.order)
  const seen = new Set<string>()
  const results: EmojiCatalogItem[] = []
  for (const row of scored) {
    if (seen.has(row.emoji.id)) continue
    seen.add(row.emoji.id)
    results.push(row.emoji)
    if (results.length >= limit) break
  }
  return results
}
