export type ServerHostMode = 'cloud' | 'selfhosted'

const STORAGE_KEY = 'dagr.serverHost'

const LEGACY_DEFAULT_SELF_HOSTED_URL = 'http://localhost:8080'

/** Local/self-hosted API. Override with VITE_DAGR_SELF_HOSTED_URL. */
export const DEFAULT_SELF_HOSTED_URL = (
  (import.meta.env.VITE_DAGR_SELF_HOSTED_URL as string | undefined)?.trim() ||
  LEGACY_DEFAULT_SELF_HOSTED_URL
).replace(/\/$/, '')

/** Hosted Dagr API. Override with VITE_DAGR_CLOUD_URL at build time. */
export const CLOUD_SERVER_URL = (
  (import.meta.env.VITE_DAGR_CLOUD_URL as string | undefined)?.trim() ||
  'https://api.dagr.no'
).replace(/\/$/, '')

export type StoredServerHost = {
  mode: ServerHostMode
  selfHostedUrl: string
}

function defaultMode(): ServerHostMode {
  return import.meta.env.DEV ? 'selfhosted' : 'cloud'
}

function normaliseSelfHostedUrl(value: string) {
  const trimmed = value.trim().replace(/\/$/, '')
  // Pick up a changed VITE_DAGR_SELF_HOSTED_URL when storage still has the old default.
  if (
    import.meta.env.DEV &&
    trimmed === LEGACY_DEFAULT_SELF_HOSTED_URL &&
    DEFAULT_SELF_HOSTED_URL !== LEGACY_DEFAULT_SELF_HOSTED_URL
  ) {
    return DEFAULT_SELF_HOSTED_URL
  }
  return trimmed || DEFAULT_SELF_HOSTED_URL
}

export function readStoredServerHost(): StoredServerHost {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) {
      return {
        mode: defaultMode(),
        selfHostedUrl: DEFAULT_SELF_HOSTED_URL,
      }
    }
    const parsed = JSON.parse(raw) as Partial<StoredServerHost>
    const mode =
      parsed.mode === 'cloud' || parsed.mode === 'selfhosted'
        ? parsed.mode
        : defaultMode()
    const selfHostedUrl =
      typeof parsed.selfHostedUrl === 'string' && parsed.selfHostedUrl.trim()
        ? normaliseSelfHostedUrl(parsed.selfHostedUrl)
        : DEFAULT_SELF_HOSTED_URL
    return { mode, selfHostedUrl }
  } catch {
    return {
      mode: defaultMode(),
      selfHostedUrl: DEFAULT_SELF_HOSTED_URL,
    }
  }
}

export function writeStoredServerHost(next: StoredServerHost) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(next))
  } catch {
    // ignore quota / private mode
  }
}

export function resolveServerUrl(host: StoredServerHost): string {
  if (host.mode === 'cloud') return CLOUD_SERVER_URL
  return host.selfHostedUrl.trim().replace(/\/$/, '')
}
