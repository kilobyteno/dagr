import { apiFetch } from '@/lib/api/client'

export type ApiLinkPreview = {
  id: string
  url: string
  status: 'pending' | 'ready' | 'failed' | 'skipped'
  title?: string
  description?: string
  siteName?: string
  imageUrl?: string
}

export type ApiMessageReaction = {
  emoji: string
  count: number
  reacted: boolean
  userIds?: string[]
}

export type ApiRichField = {
  name: string
  value: string
  inline?: boolean
}

export type ApiRichPayload = {
  text?: string
  username?: string
  iconUrl?: string
  embeds?: {
    title?: string
    url?: string
    description?: string
    color?: string
    author?: { name?: string; url?: string; iconUrl?: string }
    fields?: ApiRichField[]
    thumbnailUrl?: string
    imageUrl?: string
    footer?: { text?: string; iconUrl?: string }
    timestamp?: string
  }[]
}

export type ApiMessage = {
  id: string
  channelId: string
  authorId: string
  authorName?: string
  authorHandle?: string
  authorHasAvatar?: boolean
  authorAvatarUpdatedAt?: string | null
  authorKind?: string
  authorIconUrl?: string
  body: string
  contentType: string
  payload?: ApiRichPayload
  linkPreviews?: ApiLinkPreview[]
  reactions?: ApiMessageReaction[]
  createdAt: string
  editedAt?: string | null
}

export type ApiScheduledMessage = {
  id: string
  channelId: string
  authorId: string
  body: string
  contentType: string
  sendAt: string
  status: string
  createdAt: string
}

export type ListMessagesResponse = {
  messages: ApiMessage[]
  historyLimited?: boolean
  historyRetentionDays?: number | null
}

export type PostMessageResponse = {
  message: ApiMessage
}

export type ScheduleMessageResponse = {
  scheduledMessage: ApiScheduledMessage
}

export type ListScheduledResponse = {
  scheduledMessages: ApiScheduledMessage[]
}

export function listMessages(
  serverUrl: string,
  token: string,
  channelId: string,
  opts?: { before?: string; beforeId?: string; limit?: number; signal?: AbortSignal },
) {
  const params = new URLSearchParams()
  if (opts?.before) params.set('before', opts.before)
  if (opts?.beforeId) params.set('beforeId', opts.beforeId)
  if (opts?.limit) params.set('limit', String(opts.limit))
  const query = params.toString()
  return apiFetch<ListMessagesResponse>(
    serverUrl,
    `/api/v1/channels/${channelId}/messages${query ? `?${query}` : ''}`,
    { method: 'GET', token, signal: opts?.signal },
  )
}

export function postMessage(
  serverUrl: string,
  token: string,
  channelId: string,
  body: string,
) {
  return apiFetch<PostMessageResponse>(
    serverUrl,
    `/api/v1/channels/${channelId}/messages`,
    { method: 'POST', token, body: { body } },
  )
}

export function updateMessage(
  serverUrl: string,
  token: string,
  messageId: string,
  body: string,
) {
  return apiFetch<PostMessageResponse>(
    serverUrl,
    `/api/v1/messages/${messageId}`,
    { method: 'PATCH', token, body: { body } },
  )
}

export function deleteMessage(
  serverUrl: string,
  token: string,
  messageId: string,
) {
  return apiFetch<void>(serverUrl, `/api/v1/messages/${messageId}`, {
    method: 'DELETE',
    token,
  })
}

export function scheduleMessage(
  serverUrl: string,
  token: string,
  channelId: string,
  input: { body: string; sendAt: string },
) {
  return apiFetch<ScheduleMessageResponse>(
    serverUrl,
    `/api/v1/channels/${channelId}/scheduled-messages`,
    { method: 'POST', token, body: input },
  )
}

export function listScheduledMessages(
  serverUrl: string,
  token: string,
  channelId: string,
  signal?: AbortSignal,
) {
  return apiFetch<ListScheduledResponse>(
    serverUrl,
    `/api/v1/channels/${channelId}/scheduled-messages`,
    { method: 'GET', token, signal },
  )
}

export function cancelScheduledMessage(
  serverUrl: string,
  token: string,
  scheduledId: string,
) {
  return apiFetch<void>(serverUrl, `/api/v1/scheduled-messages/${scheduledId}`, {
    method: 'DELETE',
    token,
  })
}

export type ChannelUnreadResponse = {
  unreadCount: number
  firstUnreadMessageId?: string
}

export function markChannelRead(
  serverUrl: string,
  token: string,
  channelId: string,
  messageId?: string,
) {
  return apiFetch<ChannelUnreadResponse>(
    serverUrl,
    `/api/v1/channels/${channelId}/read`,
    {
      method: 'POST',
      token,
      body: messageId ? { messageId } : {},
    },
  )
}

export function markChannelUnread(
  serverUrl: string,
  token: string,
  channelId: string,
  messageId: string,
) {
  return apiFetch<ChannelUnreadResponse>(
    serverUrl,
    `/api/v1/channels/${channelId}/unread`,
    {
      method: 'POST',
      token,
      body: { messageId },
    },
  )
}

export function toggleMessageReaction(
  serverUrl: string,
  token: string,
  messageId: string,
  emoji: string,
) {
  return apiFetch<PostMessageResponse>(
    serverUrl,
    `/api/v1/messages/${messageId}/reactions`,
    {
      method: 'POST',
      token,
      body: { emoji },
    },
  )
}
