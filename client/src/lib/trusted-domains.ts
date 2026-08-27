const STORAGE_KEY = 'dagr.trustedDomains'

export function normalizeDomain(raw: string): string {
  let value = raw.trim().toLowerCase()
  if (!value) return ''
  try {
    if (value.includes('://')) {
      value = new URL(value).hostname
    } else if (value.includes('/') || value.includes('?') || value.includes('#')) {
      value = new URL(`https://${value}`).hostname
    }
  } catch {
    // Keep the trimmed lowercased input.
  }
  value = value.replace(/^\.+|\.+$/g, '')
  if (value.startsWith('www.')) {
    value = value.slice(4)
  }
  return value
}

export function hostFromHref(href: string): string | null {
  try {
    const url = new URL(href)
    if (url.protocol !== 'http:' && url.protocol !== 'https:') {
      return null
    }
    return normalizeDomain(url.hostname)
  } catch {
    return null
  }
}

/** True when host equals a trusted domain or is one of its subdomains. */
export function hostMatchesTrusted(host: string, trusted: string): boolean {
  const h = normalizeDomain(host)
  const t = normalizeDomain(trusted)
  if (!h || !t) return false
  return h === t || h.endsWith(`.${t}`)
}

export function isHostTrusted(host: string, trustedDomains: readonly string[]): boolean {
  return trustedDomains.some((trusted) => hostMatchesTrusted(host, trusted))
}

export function readTrustedDomains(): string[] {
  if (typeof window === 'undefined') return []
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY)
    if (!raw) return []
    const parsed = JSON.parse(raw) as unknown
    if (!Array.isArray(parsed)) return []
    const next = new Set<string>()
    for (const item of parsed) {
      if (typeof item !== 'string') continue
      const domain = normalizeDomain(item)
      if (domain) next.add(domain)
    }
    return [...next].sort()
  } catch {
    return []
  }
}

export function writeTrustedDomains(domains: readonly string[]): string[] {
  const next = [
    ...new Set(
      domains
        .map((item) => normalizeDomain(item))
        .filter(Boolean),
    ),
  ].sort()
  if (typeof window !== 'undefined') {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(next))
  }
  return next
}

export function addTrustedDomain(domain: string): string[] {
  const current = readTrustedDomains()
  const normalized = normalizeDomain(domain)
  if (!normalized) return current
  return writeTrustedDomains([...current, normalized])
}

export function removeTrustedDomain(domain: string): string[] {
  const normalized = normalizeDomain(domain)
  return writeTrustedDomains(
    readTrustedDomains().filter((item) => item !== normalized),
  )
}
