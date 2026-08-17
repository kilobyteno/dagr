import { apiFetch } from '@/lib/api/client'

export type ApiNotificationKind =
  | 'mention'
  | 'message'
  | 'reaction'
  | 'channel_invite'
  | 'workspace_invite'
  | 'workspace_join'

export type ApiNotification = {
  id: string
  kind: ApiNotificationKind
  body: string
  actorId?: string
  actorName?: string
  workspaceId?: string
  channelId?: string
  channelName?: string
  isDm?: boolean
  messageId?: string
  unread: boolean
  readAt?: string
  createdAt: string
}

export type ListNotificationsResponse = {
  notifications: ApiNotification[]
  unreadCount: number
}

export function listNotifications(
  serverUrl: string,
  token: string,
  opts?: { filter?: 'all' | 'unread' | 'mentions'; limit?: number; signal?: AbortSignal },
) {
  const params = new URLSearchParams()
  if (opts?.filter && opts.filter !== 'all') params.set('filter', opts.filter)
  if (opts?.limit) params.set('limit', String(opts.limit))
  const query = params.toString()
  return apiFetch<ListNotificationsResponse>(
    serverUrl,
    `/api/v1/notifications${query ? `?${query}` : ''}`,
    { method: 'GET', token, signal: opts?.signal },
  )
}

export function markNotificationRead(
  serverUrl: string,
  token: string,
  notificationId: string,
) {
  return apiFetch<void>(
    serverUrl,
    `/api/v1/notifications/${notificationId}/read`,
    { method: 'POST', token },
  )
}

export function markAllNotificationsRead(serverUrl: string, token: string) {
  return apiFetch<void>(serverUrl, '/api/v1/notifications/read-all', {
    method: 'POST',
    token,
  })
}
