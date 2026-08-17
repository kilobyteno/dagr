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
}

export type EmojiCategory = {
  id: string
  label: string
  emojis: EmojiCatalogItem[]
}

export type ResolvedEmoji =
  | { kind: 'unicode'; id: string; native: string }
  | { kind: 'custom'; id: string; url: string; displayName: string }

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

function emojiById(id: string): Emoji | undefined {
  return mart.emojis[id]
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

export function getEmojiCatalog(): EmojiCategory[] {
  if (catalogCache) return catalogCache

  catalogCache = mart.categories.map((category) => ({
    id: category.id,
    label: CATEGORY_LABELS[category.id] ?? category.id,
    emojis: category.emojis
      .map((id) => {
        const emoji = emojiById(id)
        if (!emoji) return null
        const native = emoji.skins[0]?.native
        if (!native) return null
        return {
          id: emoji.id,
          name: emoji.name,
          native,
          keywords: emoji.keywords ?? [],
        } satisfies EmojiCatalogItem
      })
      .filter((item): item is EmojiCatalogItem => item !== null),
  }))

  return catalogCache
}

export function searchEmojiCatalog(
  query: string,
  limit = 80,
): EmojiCatalogItem[] {
  const q = query.trim().toLowerCase()
  if (!q) return []

  const results: EmojiCatalogItem[] = []
  for (const category of getEmojiCatalog()) {
    for (const emoji of category.emojis) {
      const haystack = [emoji.id, emoji.name, ...emoji.keywords]
        .join(' ')
        .toLowerCase()
      if (!haystack.includes(q)) continue
      results.push(emoji)
      if (results.length >= limit) return results
    }
  }
  return results
}
