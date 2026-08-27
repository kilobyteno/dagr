import { apiFetch } from '@/lib/api/client'
import type {
  ApiChannel,
  ApiWorkspaceMember,
  ListWorkspaceMembersResponse,
} from '@/lib/api/workspaces'

export type CreateChannelInput = {
  name: string
  topic?: string
  isPrivate?: boolean
}

export type UpdateChannelInput = {
  name?: string
  topic?: string
  isPrivate?: boolean
}

export type ChannelResponse = {
  channel: ApiChannel
}

export function createChannel(
  serverUrl: string,
  token: string,
  workspaceId: string,
  input: CreateChannelInput,
) {
  return apiFetch<ChannelResponse>(
    serverUrl,
    `/api/v1/workspaces/${workspaceId}/channels`,
    { method: 'POST', token, body: input },
  )
}

export function updateChannel(
  serverUrl: string,
  token: string,
  channelId: string,
  input: UpdateChannelInput,
) {
  return apiFetch<ChannelResponse>(serverUrl, `/api/v1/channels/${channelId}`, {
    method: 'PATCH',
    token,
    body: input,
  })
}

export function deleteChannel(
  serverUrl: string,
  token: string,
  channelId: string,
) {
  return apiFetch<void>(serverUrl, `/api/v1/channels/${channelId}`, {
    method: 'DELETE',
    token,
  })
}

export function listChannelMembers(
  serverUrl: string,
  token: string,
  channelId: string,
  signal?: AbortSignal,
) {
  return apiFetch<ListWorkspaceMembersResponse>(
    serverUrl,
    `/api/v1/channels/${channelId}/members`,
    { method: 'GET', token, signal },
  )
}

export type { ApiWorkspaceMember }

export function addChannelMember(
  serverUrl: string,
  token: string,
  channelId: string,
  email: string,
) {
  return apiFetch<void>(serverUrl, `/api/v1/channels/${channelId}/members`, {
    method: 'POST',
    token,
    body: { email },
  })
}

export function removeChannelMember(
  serverUrl: string,
  token: string,
  channelId: string,
  userId: string,
) {
  return apiFetch<void>(
    serverUrl,
    `/api/v1/channels/${channelId}/members/${userId}`,
    { method: 'DELETE', token },
  )
}

export type ChannelNotificationLevel = 'all' | 'mentions' | 'nothing'

export type ChannelNotificationSettingsResponse = {
  level: ChannelNotificationLevel
}

export function getChannelNotificationSettings(
  serverUrl: string,
  token: string,
  channelId: string,
) {
  return apiFetch<ChannelNotificationSettingsResponse>(
    serverUrl,
    `/api/v1/channels/${channelId}/notification-settings`,
    { method: 'GET', token },
  )
}

export function updateChannelNotificationSettings(
  serverUrl: string,
  token: string,
  channelId: string,
  level: ChannelNotificationLevel,
) {
  return apiFetch<ChannelNotificationSettingsResponse>(
    serverUrl,
    `/api/v1/channels/${channelId}/notification-settings`,
    { method: 'PUT', token, body: { level } },
  )
}
