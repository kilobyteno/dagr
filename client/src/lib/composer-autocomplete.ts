import type { ChatChannelRef, ChatUserRef } from '@/components/chat/user-handle'
import {
  getDefaultEmojiSuggestions,
  searchEmojiCatalog,
  type EmojiCatalogItem,
} from '@/lib/emoji'

export const COMPOSER_AUTOCOMPLETE_LIMIT = 20
export const COMPOSER_AUTOCOMPLETE_LIST_ID = 'composer-autocomplete'

export type ComposerAutocompleteTrigger =
  | { kind: 'mention'; start: number; query: string }
  | { kind: 'channel'; start: number; query: string }
  | { kind: 'emoji'; start: number; query: string }

export type ComposerAutocompleteItem =
  | { kind: 'mention'; id: string; user: ChatUserRef }
  | { kind: 'channel'; id: string; channel: ChatChannelRef }
  | { kind: 'emoji'; id: string; emoji: EmojiCatalogItem }

const MENTION_RE = /(?:^|\s)(@([^\s@]*))$/
const CHANNEL_RE = /(?:^|\s)(#([a-zA-Z0-9_-]*))$/
const EMOJI_RE = /(?:^|\s)(:([^\s:]*))$/

export function detectComposerTrigger(
  value: string,
  caret: number,
): ComposerAutocompleteTrigger | null {
  if (caret < 0 || caret > value.length) return null
  const before = value.slice(0, caret)
  const candidates: ComposerAutocompleteTrigger[] = []
  const mention = MENTION_RE.exec(before)
  if (mention) {
    candidates.push({
      kind: 'mention',
      start: before.length - mention[1].length,
      query: mention[2],
    })
  }
  const channel = CHANNEL_RE.exec(before)
  if (channel) {
    candidates.push({
      kind: 'channel',
      start: before.length - channel[1].length,
      query: channel[2],
    })
  }
  const emoji = EMOJI_RE.exec(before)
  if (emoji) {
    candidates.push({
      kind: 'emoji',
      start: before.length - emoji[1].length,
      query: emoji[2],
    })
  }
  if (candidates.length === 0) return null
  return candidates.reduce((best, item) =>
    item.start >= best.start ? item : best,
  )
}

function mentionScore(member: ChatUserRef, query: string): number | null {
  const handle = member.handle.toLowerCase()
  const name = member.displayName.toLowerCase()
  const former = (member.formerHandles ?? []).map((item) => item.toLowerCase())
  if (!query) return 2
  if (handle === query || name === query) return 0
  if (
    handle.startsWith(query) ||
    name.startsWith(query) ||
    former.some((item) => item.startsWith(query))
  ) {
    return 1
  }
  if (
    handle.includes(query) ||
    name.includes(query) ||
    former.some((item) => item.includes(query))
  ) {
    return 2
  }
  return null
}

function channelScore(channel: ChatChannelRef, query: string): number | null {
  const name = channel.name.toLowerCase()
  const topic = channel.topic?.toLowerCase() ?? ''
  if (!query) return 2
  if (name === query) return 0
  if (name.startsWith(query)) return 1
  if (name.includes(query) || (topic && topic.includes(query))) return 2
  return null
}

export function filterMentionMembers(
  members: ChatUserRef[],
  query: string,
  options?: { currentUserId?: string; limit?: number },
): ChatUserRef[] {
  const limit = options?.limit ?? COMPOSER_AUTOCOMPLETE_LIMIT
  const currentUserId = options?.currentUserId
  const q = query.trim().toLowerCase()
  const scored: { member: ChatUserRef; score: number }[] = []
  for (const member of members) {
    if (!member.handle) continue
    if (currentUserId && member.userId === currentUserId) continue
    const score = mentionScore(member, q)
    if (score === null) continue
    scored.push({ member, score })
  }
  scored.sort(
    (a, b) =>
      a.score - b.score ||
      a.member.displayName.localeCompare(b.member.displayName) ||
      a.member.handle.localeCompare(b.member.handle),
  )
  return scored.slice(0, limit).map((row) => row.member)
}

export function filterMentionChannels(
  channels: ChatChannelRef[],
  query: string,
  limit = COMPOSER_AUTOCOMPLETE_LIMIT,
): ChatChannelRef[] {
  const q = query.trim().toLowerCase()
  const scored: { channel: ChatChannelRef; score: number }[] = []
  for (const channel of channels) {
    if (!channel.name) continue
    const score = channelScore(channel, q)
    if (score === null) continue
    scored.push({ channel, score })
  }
  scored.sort(
    (a, b) =>
      a.score - b.score || a.channel.name.localeCompare(b.channel.name),
  )
  return scored.slice(0, limit).map((row) => row.channel)
}

export function listComposerAutocompleteItems(
  trigger: ComposerAutocompleteTrigger,
  options: {
    members: ChatUserRef[]
    channels: ChatChannelRef[]
    currentUserId?: string
  },
): ComposerAutocompleteItem[] {
  if (trigger.kind === 'mention') {
    return filterMentionMembers(options.members, trigger.query, {
      currentUserId: options.currentUserId,
    }).map((user) => ({
      kind: 'mention',
      id: `mention-${user.userId}`,
      user,
    }))
  }
  if (trigger.kind === 'channel') {
    return filterMentionChannels(options.channels, trigger.query).map(
      (channel) => ({
        kind: 'channel',
        id: `channel-${channel.id}`,
        channel,
      }),
    )
  }
  const emojis = trigger.query.trim()
    ? searchEmojiCatalog(trigger.query, COMPOSER_AUTOCOMPLETE_LIMIT)
    : getDefaultEmojiSuggestions(COMPOSER_AUTOCOMPLETE_LIMIT)
  return emojis.map((emoji) => ({
    kind: 'emoji',
    id: `emoji-${emoji.id}`,
    emoji,
  }))
}

export function composerAutocompleteInsertion(
  item: ComposerAutocompleteItem,
): string {
  if (item.kind === 'mention') return `@${item.user.handle} `
  if (item.kind === 'channel') return `#${item.channel.name} `
  return `:${item.emoji.id}: `
}

export function applyAutocompleteInsertion(
  value: string,
  trigger: ComposerAutocompleteTrigger,
  caret: number,
  insertion: string,
): { next: string; selectionStart: number; selectionEnd: number } {
  const next = `${value.slice(0, trigger.start)}${insertion}${value.slice(caret)}`
  const selectionStart = trigger.start + insertion.length
  return {
    next,
    selectionStart,
    selectionEnd: selectionStart,
  }
}
