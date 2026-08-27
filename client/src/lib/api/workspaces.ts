import { apiFetch, apiUpload } from '@/lib/api/client'

export type ApiWorkspace = {
  id: string
  name: string
  slug: string
  role: string
  hasIcon?: boolean
  iconUpdatedAt?: string
}

export type ApiChannel = {
  id: string
  workspaceId: string
  name: string
  topic?: string
  isPrivate: boolean
  isDm?: boolean
  isShared?: boolean
  createdBy?: string
  unreadCount?: number
  firstUnreadMessageId?: string
  peerUserId?: string
  peerDisplayName?: string
  peerHandle?: string
  peerHasAvatar?: boolean
  peerAvatarUpdatedAt?: string | null
}

export type OpenDirectMessageResponse = {
  channel: ApiChannel
}

export type ListWorkspacesResponse = {
  workspaces: ApiWorkspace[]
}

export type CreateWorkspaceResponse = {
  workspace: ApiWorkspace
  channels: ApiChannel[]
}

export type GetWorkspaceResponse = {
  workspace: ApiWorkspace
}

export type ListChannelsResponse = {
  channels: ApiChannel[]
}

export type ApiWorkspaceMember = {
  userId: string
  displayName: string
  handle: string
  formerHandles?: string[]
  role: string
  kind?: string
  isExternal?: boolean
  homeWorkspaceId?: string
  homeWorkspaceName?: string
  homeServerId?: string
  homeWorkspaceRemoteId?: string
  homeWorkspaceIconUrl?: string
  homeServerUrl?: string
  statusEmoji?: string
  statusText?: string
  statusExpiresAt?: string | null
  presence?: 'online' | 'away' | 'offline'
  hasAvatar?: boolean
  avatarUpdatedAt?: string | null
}

export type WorkspaceMemberResponse = {
  member: ApiWorkspaceMember
}

export type ListWorkspaceMembersResponse = {
  members: ApiWorkspaceMember[]
}

export function listWorkspaces(
  serverUrl: string,
  token: string,
  signal?: AbortSignal,
) {
  return apiFetch<ListWorkspacesResponse>(serverUrl, '/api/v1/workspaces', {
    method: 'GET',
    token,
    signal,
  })
}

export function createWorkspace(
  serverUrl: string,
  token: string,
  input: { name: string },
) {
  return apiFetch<CreateWorkspaceResponse>(serverUrl, '/api/v1/workspaces', {
    method: 'POST',
    token,
    body: input,
  })
}

export function getWorkspace(
  serverUrl: string,
  token: string,
  workspaceId: string,
  signal?: AbortSignal,
) {
  return apiFetch<GetWorkspaceResponse>(
    serverUrl,
    `/api/v1/workspaces/${workspaceId}`,
    {
      method: 'GET',
      token,
      signal,
    },
  )
}

export function listChannels(
  serverUrl: string,
  token: string,
  workspaceId: string,
  signal?: AbortSignal,
) {
  return apiFetch<ListChannelsResponse>(
    serverUrl,
    `/api/v1/workspaces/${workspaceId}/channels`,
    {
      method: 'GET',
      token,
      signal,
    },
  )
}

export function openDirectMessage(
  serverUrl: string,
  token: string,
  workspaceId: string,
  userId: string,
) {
  return apiFetch<OpenDirectMessageResponse>(
    serverUrl,
    `/api/v1/workspaces/${workspaceId}/dms`,
    {
      method: 'POST',
      token,
      body: { userId },
    },
  )
}

export function renameWorkspace(
  serverUrl: string,
  token: string,
  workspaceId: string,
  input: { name: string },
) {
  return apiFetch<GetWorkspaceResponse>(
    serverUrl,
    `/api/v1/workspaces/${workspaceId}`,
    {
      method: 'PATCH',
      token,
      body: input,
    },
  )
}

export function deleteWorkspace(
  serverUrl: string,
  token: string,
  workspaceId: string,
) {
  return apiFetch<void>(serverUrl, `/api/v1/workspaces/${workspaceId}`, {
    method: 'DELETE',
    token,
  })
}

export function uploadWorkspaceIcon(
  serverUrl: string,
  token: string,
  workspaceId: string,
  file: File,
) {
  const formData = new FormData()
  formData.append('icon', file)
  return apiUpload<GetWorkspaceResponse>(
    serverUrl,
    `/api/v1/workspaces/${workspaceId}/icon`,
    { method: 'PUT', token, formData },
  )
}

export function removeWorkspaceIcon(
  serverUrl: string,
  token: string,
  workspaceId: string,
) {
  return apiFetch<GetWorkspaceResponse>(
    serverUrl,
    `/api/v1/workspaces/${workspaceId}/icon`,
    { method: 'DELETE', token },
  )
}

export type WorkspaceIconObject = {
  url: string
  contentType: string
}

export async function fetchWorkspaceIconObjectUrl(
  serverUrl: string,
  token: string,
  workspaceId: string,
  signal?: AbortSignal,
): Promise<WorkspaceIconObject | null> {
  const base = serverUrl.trim().replace(/\/$/, '')
  let response: Response
  try {
    response = await fetch(`${base}/api/v1/workspaces/${workspaceId}/icon`, {
      method: 'GET',
      headers: {
        Authorization: `Bearer ${token}`,
        Accept: 'image/*',
      },
      signal,
    })
  } catch {
    return null
  }
  if (!response.ok) return null
  const blob = await response.blob()
  const headerType = response.headers.get('Content-Type')?.split(';')[0]?.trim()
  return {
    url: URL.createObjectURL(blob),
    contentType: blob.type || headerType || '',
  }
}

export function getWorkspaceMe(
  serverUrl: string,
  token: string,
  workspaceId: string,
  signal?: AbortSignal,
) {
  return apiFetch<WorkspaceMemberResponse>(
    serverUrl,
    `/api/v1/workspaces/${workspaceId}/me`,
    {
      method: 'GET',
      token,
      signal,
    },
  )
}

export function updateWorkspaceMe(
  serverUrl: string,
  token: string,
  workspaceId: string,
  input: { handle: string },
) {
  return apiFetch<WorkspaceMemberResponse>(
    serverUrl,
    `/api/v1/workspaces/${workspaceId}/me`,
    {
      method: 'PATCH',
      token,
      body: input,
    },
  )
}

export function listWorkspaceMembers(
  serverUrl: string,
  token: string,
  workspaceId: string,
  signal?: AbortSignal,
) {
  return apiFetch<ListWorkspaceMembersResponse>(
    serverUrl,
    `/api/v1/workspaces/${workspaceId}/members`,
    {
      method: 'GET',
      token,
      signal,
    },
  )
}

export function updateWorkspaceMemberRole(
  serverUrl: string,
  token: string,
  workspaceId: string,
  userId: string,
  role: string,
) {
  return apiFetch<WorkspaceMemberResponse>(
    serverUrl,
    `/api/v1/workspaces/${workspaceId}/members/${userId}`,
    { method: 'PATCH', token, body: { role } },
  )
}

export function removeWorkspaceMember(
  serverUrl: string,
  token: string,
  workspaceId: string,
  userId: string,
) {
  return apiFetch<void>(
    serverUrl,
    `/api/v1/workspaces/${workspaceId}/members/${userId}`,
    { method: 'DELETE', token },
  )
}

export function leaveWorkspace(
  serverUrl: string,
  token: string,
  workspaceId: string,
) {
  return apiFetch<void>(
    serverUrl,
    `/api/v1/workspaces/${workspaceId}/leave`,
    { method: 'POST', token },
  )
}

export function transferWorkspaceOwnership(
  serverUrl: string,
  token: string,
  workspaceId: string,
  userId: string,
) {
  return apiFetch<void>(
    serverUrl,
    `/api/v1/workspaces/${workspaceId}/transfer-ownership`,
    { method: 'POST', token, body: { userId } },
  )
}

/** Derive rail initials from a workspace name (first letter, or two words). */
export function workspaceInitials(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean)
  if (parts.length === 0) return '?'
  if (parts.length === 1) {
    return parts[0].slice(0, 1).toUpperCase()
  }
  return (parts[0][0] + parts[1][0]).toUpperCase()
}
