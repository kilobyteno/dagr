import { apiFetch } from '@/lib/api/client'
import type { ApiWorkspace } from '@/lib/api/workspaces'

export type ApiInvite = {
  id: string
  workspaceId: string
  email: string
  token: string
  role: string
  expiresAt: string
  acceptedAt?: string
  acceptPath: string
}

export type InviteResponse = {
  status: 'added' | 'invited'
  invite?: ApiInvite
}

export type ListInvitesResponse = {
  invites: ApiInvite[]
}

export type AcceptInviteResponse = {
  workspace: ApiWorkspace
}

export function inviteToWorkspace(
  serverUrl: string,
  token: string,
  workspaceId: string,
  input: { email: string; role?: string },
) {
  return apiFetch<InviteResponse>(
    serverUrl,
    `/api/v1/workspaces/${workspaceId}/invites`,
    { method: 'POST', token, body: input },
  )
}

export function listWorkspaceInvites(
  serverUrl: string,
  token: string,
  workspaceId: string,
  signal?: AbortSignal,
) {
  return apiFetch<ListInvitesResponse>(
    serverUrl,
    `/api/v1/workspaces/${workspaceId}/invites`,
    { method: 'GET', token, signal },
  )
}

export function acceptInvite(
  serverUrl: string,
  token: string,
  inviteToken: string,
) {
  return apiFetch<AcceptInviteResponse>(
    serverUrl,
    `/api/v1/invites/${encodeURIComponent(inviteToken)}/accept`,
    { method: 'POST', token },
  )
}

export function revokeWorkspaceInvite(
  serverUrl: string,
  token: string,
  workspaceId: string,
  inviteId: string,
) {
  return apiFetch<void>(
    serverUrl,
    `/api/v1/workspaces/${workspaceId}/invites/${inviteId}`,
    { method: 'DELETE', token },
  )
}
