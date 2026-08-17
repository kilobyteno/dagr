import { apiFetch } from '@/lib/api/client'

export type ApiWorkspaceDomain = {
  id: string
  workspaceId: string
  domain: string
  verified: boolean
  verifiedAt?: string
  autoJoin: boolean
  verificationToken?: string
  dnsHost: string
  dnsType: string
  dnsValue: string
}

export type ListDomainsResponse = {
  domains: ApiWorkspaceDomain[]
}

export type DomainResponse = {
  domain: ApiWorkspaceDomain
}

export function listDomains(
  serverUrl: string,
  token: string,
  workspaceId: string,
  signal?: AbortSignal,
) {
  return apiFetch<ListDomainsResponse>(
    serverUrl,
    `/api/v1/workspaces/${workspaceId}/domains`,
    { method: 'GET', token, signal },
  )
}

export function addDomain(
  serverUrl: string,
  token: string,
  workspaceId: string,
  input: { domain: string },
) {
  return apiFetch<DomainResponse>(
    serverUrl,
    `/api/v1/workspaces/${workspaceId}/domains`,
    { method: 'POST', token, body: input },
  )
}

export function verifyDomain(
  serverUrl: string,
  token: string,
  workspaceId: string,
  domainId: string,
) {
  return apiFetch<DomainResponse>(
    serverUrl,
    `/api/v1/workspaces/${workspaceId}/domains/${domainId}/verify`,
    { method: 'POST', token },
  )
}

export function updateDomain(
  serverUrl: string,
  token: string,
  workspaceId: string,
  domainId: string,
  input: { autoJoin: boolean },
) {
  return apiFetch<DomainResponse>(
    serverUrl,
    `/api/v1/workspaces/${workspaceId}/domains/${domainId}`,
    { method: 'PATCH', token, body: input },
  )
}

export function removeDomain(
  serverUrl: string,
  token: string,
  workspaceId: string,
  domainId: string,
) {
  return apiFetch<void>(
    serverUrl,
    `/api/v1/workspaces/${workspaceId}/domains/${domainId}`,
    { method: 'DELETE', token },
  )
}
