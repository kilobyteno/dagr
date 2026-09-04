import { apiFetch } from '@/lib/api/client'

export type ApiDocumentSummary = {
  id: string
  workspaceId: string
  parentId?: string
  slug: string
  title: string
  icon?: string
  createdBy?: string
  updatedBy?: string
  createdAt: string
  updatedAt: string
}

export type ApiDocument = ApiDocumentSummary & {
  body?: string
}

export type ListDocumentsResponse = {
  documents: ApiDocumentSummary[]
}

export type DocumentResponse = {
  document: ApiDocument
}

export type DocumentNode = ApiDocumentSummary & {
  children: DocumentNode[]
}

export const MAX_DOCUMENT_DEPTH = 5

export function canCreateChildPage(depth: number): boolean {
  return depth < MAX_DOCUMENT_DEPTH
}

export function buildDocumentTree(
  documents: readonly ApiDocumentSummary[],
): DocumentNode[] {
  const byId = new Map<string, DocumentNode>()
  for (const item of documents) {
    byId.set(item.id, { ...item, children: [] })
  }
  const roots: DocumentNode[] = []
  for (const node of byId.values()) {
    const parent = node.parentId ? byId.get(node.parentId) : undefined
    if (parent) {
      parent.children.push(node)
    } else {
      roots.push(node)
    }
  }
  const sortNodes = (items: DocumentNode[]) => {
    items.sort(
      (a, b) => a.title.localeCompare(b.title) || a.slug.localeCompare(b.slug),
    )
    for (const item of items) sortNodes(item.children)
  }
  sortNodes(roots)
  return roots
}

export function listDocuments(
  serverUrl: string,
  token: string,
  workspaceId: string,
  signal?: AbortSignal,
) {
  return apiFetch<ListDocumentsResponse>(
    serverUrl,
    `/api/v1/workspaces/${workspaceId}/documents`,
    { method: 'GET', token, signal },
  )
}

export function searchDocuments(
  serverUrl: string,
  token: string,
  workspaceId: string,
  query: string,
  signal?: AbortSignal,
) {
  const params = new URLSearchParams()
  if (query.trim()) params.set('q', query.trim())
  const suffix = params.size ? `?${params}` : ''
  return apiFetch<ListDocumentsResponse>(
    serverUrl,
    `/api/v1/workspaces/${workspaceId}/documents/search${suffix}`,
    { method: 'GET', token, signal },
  )
}

export type ApiDocumentRevisionSummary = {
  id: string
  documentId: string
  version: number
  parentId?: string
  slug: string
  title: string
  icon?: string
  createdBy: string
  createdByName?: string
  createdAt: string
}

export type ApiDocumentRevision = ApiDocumentRevisionSummary & {
  body?: string
}

export type ListDocumentRevisionsResponse = {
  revisions: ApiDocumentRevisionSummary[]
}

export type DocumentRevisionResponse = {
  revision: ApiDocumentRevision
}

export function createDocument(
  serverUrl: string,
  token: string,
  workspaceId: string,
  input: {
    title: string
    body?: string
    slug?: string
    icon?: string
    parentId?: string
  },
) {
  return apiFetch<DocumentResponse>(
    serverUrl,
    `/api/v1/workspaces/${workspaceId}/documents`,
    { method: 'POST', token, body: input },
  )
}

export function getDocument(
  serverUrl: string,
  token: string,
  documentId: string,
  signal?: AbortSignal,
) {
  return apiFetch<DocumentResponse>(
    serverUrl,
    `/api/v1/documents/${documentId}`,
    { method: 'GET', token, signal },
  )
}

export function updateDocument(
  serverUrl: string,
  token: string,
  documentId: string,
  input: {
    title?: string
    body?: string
    slug?: string
    icon?: string
    parentId?: string | null
  },
) {
  return apiFetch<DocumentResponse>(
    serverUrl,
    `/api/v1/documents/${documentId}`,
    { method: 'PATCH', token, body: input },
  )
}

export function deleteDocument(
  serverUrl: string,
  token: string,
  documentId: string,
) {
  return apiFetch<void>(serverUrl, `/api/v1/documents/${documentId}`, {
    method: 'DELETE',
    token,
  })
}

export function listDocumentRevisions(
  serverUrl: string,
  token: string,
  documentId: string,
  signal?: AbortSignal,
) {
  return apiFetch<ListDocumentRevisionsResponse>(
    serverUrl,
    `/api/v1/documents/${documentId}/revisions`,
    { method: 'GET', token, signal },
  )
}

export function getDocumentRevision(
  serverUrl: string,
  token: string,
  documentId: string,
  revisionId: string,
  signal?: AbortSignal,
) {
  return apiFetch<DocumentRevisionResponse>(
    serverUrl,
    `/api/v1/documents/${documentId}/revisions/${revisionId}`,
    { method: 'GET', token, signal },
  )
}

export function restoreDocumentRevision(
  serverUrl: string,
  token: string,
  documentId: string,
  revisionId: string,
) {
  return apiFetch<DocumentResponse>(
    serverUrl,
    `/api/v1/documents/${documentId}/revisions/${revisionId}/restore`,
    { method: 'POST', token, body: {} },
  )
}
