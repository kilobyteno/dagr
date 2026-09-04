export type ServerHostMode = 'cloud' | 'selfhosted'

const STORAGE_KEY = 'dagr.serverHost'

const FALLBACK_SELF_HOSTED_URL = 'http://localhost:8383'
const FORMER_DEFAULT_SELF_HOSTED_URLS = [
  'http://localhost:8080',
  'http://localhost:3030',
] as const

/** Local/self-hosted API. Override with VITE_DAGR_SELF_HOSTED_URL. */
export const DEFAULT_SELF_HOSTED_URL = (
  (import.meta.env.VITE_DAGR_SELF_HOSTED_URL as string | undefined)?.trim() ||
    FALLBACK_SELF_HOSTED_URL
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
  const fromEnv = import.meta.env.VITE_DAGR_DEFAULT_MODE?.trim()
  if (fromEnv === 'cloud' || fromEnv === 'selfhosted') return fromEnv
  return import.meta.env.DEV ? 'selfhosted' : 'cloud'
}

/** In dev, rewrite a stored former local default to the current API URL. */
export function migrateLocalServerUrl(serverUrl: string): string {
  const trimmed = serverUrl.trim().replace(/\/$/, '')
  if (
    import.meta.env.DEV &&
    (FORMER_DEFAULT_SELF_HOSTED_URLS as readonly string[]).includes(trimmed) &&
    DEFAULT_SELF_HOSTED_URL !== trimmed
  ) {
    return DEFAULT_SELF_HOSTED_URL
  }
  return trimmed
}

function normaliseSelfHostedUrl(value: string) {
  return migrateLocalServerUrl(value) || DEFAULT_SELF_HOSTED_URL
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
