export type DagrDeepLink =
  | { kind: 'verified' }
  | { kind: 'billing'; workspaceId: string }

let lastDeepLink: string | null = null
const listeners = new Set<(url: string) => void>()

export function parseDagrDeepLink(raw: string): DagrDeepLink | null {
  let url: URL
  try {
    url = new URL(raw)
  } catch {
    return null
  }
  if (url.protocol !== 'dagr:') return null
  const host = url.hostname.toLowerCase()
  const path = url.pathname.replace(/\/+$/, '')
  if (host === 'verified' && (path === '' || path === '/')) {
    return { kind: 'verified' }
  }
  if (host === 'billing' && (path === '/return' || path === 'return')) {
    return {
      kind: 'billing',
      workspaceId: url.searchParams.get('workspaceId')?.trim() || '',
    }
  }
  return null
}

export function pushDeepLink(url: string) {
  lastDeepLink = url
  for (const listener of listeners) {
    listener(url)
  }
}

export function subscribeDeepLink(listener: (url: string) => void) {
  listeners.add(listener)
  if (lastDeepLink) listener(lastDeepLink)
  return () => {
    listeners.delete(listener)
  }
}

export function consumeDeepLink() {
  lastDeepLink = null
}
