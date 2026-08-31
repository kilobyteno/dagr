import { apiFetch } from '@/lib/api/client'

export type ApiIncomingWebhook = {
  id: string
  channelId: string
  channelName?: string
  channelIsPrivate?: boolean
  tokenPrefix: string
  url?: string
  lastUsedAt?: string | null
  createdAt: string
}

export type ApiWorkspaceApp = {
  slug: string
  name: string
  description: string
  origin: string
  capabilities: string[]
  installed: boolean
  channels?: ApiIncomingWebhook[]
}

export type ListWorkspaceAppsResponse = {
  apps: ApiWorkspaceApp[]
}

export type IncomingWebhookResponse = {
  webhook: ApiIncomingWebhook
}

export function listWorkspaceApps(
  serverUrl: string,
  token: string,
  workspaceId: string,
  signal?: AbortSignal,
) {
  return apiFetch<ListWorkspaceAppsResponse>(
    serverUrl,
    `/api/v1/workspaces/${workspaceId}/apps`,
    { method: 'GET', token, signal },
  )
}

export function installWorkspaceApp(
  serverUrl: string,
  token: string,
  workspaceId: string,
  appSlug: string,
) {
  return apiFetch<{ app: { slug: string; name: string; id: string } }>(
    serverUrl,
    `/api/v1/workspaces/${workspaceId}/apps/${appSlug}/install`,
    { method: 'POST', token },
  )
}

export function uninstallWorkspaceApp(
  serverUrl: string,
  token: string,
  workspaceId: string,
  appSlug: string,
) {
  return apiFetch<void>(
    serverUrl,
    `/api/v1/workspaces/${workspaceId}/apps/${appSlug}`,
    { method: 'DELETE', token },
  )
}

export function getChannelIncomingWebhook(
  serverUrl: string,
  token: string,
  channelId: string,
  signal?: AbortSignal,
) {
  return apiFetch<IncomingWebhookResponse>(
    serverUrl,
    `/api/v1/channels/${channelId}/apps/incoming-webhooks`,
    { method: 'GET', token, signal },
  )
}

export function enableChannelIncomingWebhook(
  serverUrl: string,
  token: string,
  channelId: string,
) {
  return apiFetch<IncomingWebhookResponse>(
    serverUrl,
    `/api/v1/channels/${channelId}/apps/incoming-webhooks`,
    { method: 'POST', token },
  )
}

export function rotateChannelIncomingWebhook(
  serverUrl: string,
  token: string,
  channelId: string,
) {
  return apiFetch<IncomingWebhookResponse>(
    serverUrl,
    `/api/v1/channels/${channelId}/apps/incoming-webhooks/rotate`,
    { method: 'POST', token },
  )
}

export function disableChannelIncomingWebhook(
  serverUrl: string,
  token: string,
  channelId: string,
) {
  return apiFetch<void>(
    serverUrl,
    `/api/v1/channels/${channelId}/apps/incoming-webhooks`,
    { method: 'DELETE', token },
  )
}
