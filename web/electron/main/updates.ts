import { app, shell } from 'electron'
import { readFileSync, writeFileSync } from 'node:fs'
import path from 'node:path'

const GITHUB_REPO = 'kilobyteno/dagr'
const CHECK_INTERVAL_MS = 24 * 60 * 60 * 1000
const FALLBACK_RELEASE_URL = `https://github.com/${GITHUB_REPO}/releases/latest`

export type UpdateCheckResult = {
  currentVersion: string
  latestVersion: string | null
  available: boolean
  downloadUrl: string | null
  releaseUrl: string | null
  skipped?: boolean
  error?: string
}

type CachedCheck = {
  checkedAt: number
  result: UpdateCheckResult
}

type GitHubAsset = {
  name?: string
  browser_download_url?: string
}

type GitHubRelease = {
  tag_name?: string
  html_url?: string
  assets?: GitHubAsset[]
}

function cachePath() {
  return path.join(app.getPath('userData'), 'update-check.json')
}

function readCache(): CachedCheck | null {
  try {
    const raw = readFileSync(cachePath(), 'utf8')
    const parsed = JSON.parse(raw) as CachedCheck
    if (!parsed || typeof parsed.checkedAt !== 'number' || !parsed.result) {
      return null
    }
    return parsed
  } catch {
    return null
  }
}

function writeCache(result: UpdateCheckResult) {
  try {
    const payload: CachedCheck = { checkedAt: Date.now(), result }
    writeFileSync(cachePath(), JSON.stringify(payload), 'utf8')
  } catch {
    // Cache is best effort.
  }
}

function parseVersionParts(value: string): number[] | null {
  const cleaned = value.trim().replace(/^v/i, '')
  if (!cleaned) return null
  const parts = cleaned.split('.')
  if (parts.length < 2) return null
  return parts.map((part) => {
    const numeric = parseInt(part.replace(/[^0-9].*$/, ''), 10)
    return Number.isFinite(numeric) ? numeric : 0
  })
}

export function isNewerVersion(latest: string, current: string): boolean {
  const next = parseVersionParts(latest)
  const previous = parseVersionParts(current)
  if (!next || !previous) return false
  const length = Math.max(next.length, previous.length)
  for (let index = 0; index < length; index += 1) {
    const a = next[index] ?? 0
    const b = previous[index] ?? 0
    if (a > b) return true
    if (a < b) return false
  }
  return false
}

function pickAsset(assets: GitHubAsset[], suffix: string) {
  const match = assets.find((asset) => {
    const name = asset.name?.toLowerCase() ?? ''
    return name.endsWith(suffix) && !name.endsWith('.blockmap')
  })
  return match?.browser_download_url ?? null
}

function downloadUrlForPlatform(assets: GitHubAsset[]) {
  if (process.platform === 'darwin') return pickAsset(assets, '.dmg')
  if (process.platform === 'win32') return pickAsset(assets, '.exe')
  return null
}

function fallbackDownloadUrl() {
  return process.env.DAGR_DOWNLOAD_URL?.trim() || FALLBACK_RELEASE_URL
}

function isAllowedUpdateUrl(value: string) {
  try {
    const url = new URL(value)
    if (url.protocol !== 'https:') return false
    const host = url.hostname
    if (host === 'github.com' || host.endsWith('.githubusercontent.com')) {
      return true
    }
    const extra = process.env.DAGR_DOWNLOAD_URL?.trim()
    if (extra) {
      try {
        return host === new URL(extra).hostname
      } catch {
        return false
      }
    }
    return false
  } catch {
    return false
  }
}

async function fetchLatestRelease(currentVersion: string): Promise<UpdateCheckResult> {
  const response = await fetch(
    `https://api.github.com/repos/${GITHUB_REPO}/releases/latest`,
    {
      headers: {
        Accept: 'application/vnd.github+json',
        'User-Agent': 'dagr-desktop',
        'X-GitHub-Api-Version': '2022-11-28',
      },
    },
  )

  if (response.status === 404) {
    return {
      currentVersion,
      latestVersion: null,
      available: false,
      downloadUrl: null,
      releaseUrl: FALLBACK_RELEASE_URL,
    }
  }

  if (!response.ok) {
    throw new Error(`GitHub releases returned ${response.status}`)
  }

  const payload = (await response.json()) as GitHubRelease
  const tagName = payload.tag_name?.trim() ?? ''
  const latestVersion = tagName.replace(/^v/i, '') || null
  const assets = payload.assets ?? []
  const downloadUrl = downloadUrlForPlatform(assets) || fallbackDownloadUrl()
  const releaseUrl = payload.html_url?.trim() || FALLBACK_RELEASE_URL

  return {
    currentVersion,
    latestVersion,
    available: Boolean(latestVersion && isNewerVersion(latestVersion, currentVersion)),
    downloadUrl,
    releaseUrl,
  }
}

export async function checkForUpdates(
  currentVersion: string,
  options: { force?: boolean } = {},
): Promise<UpdateCheckResult> {
  if (process.env.VITE_DEV_SERVER_URL) {
    return {
      currentVersion,
      latestVersion: null,
      available: false,
      downloadUrl: null,
      releaseUrl: null,
      skipped: true,
    }
  }

  if (!options.force) {
    const cached = readCache()
    if (cached && Date.now() - cached.checkedAt < CHECK_INTERVAL_MS) {
      return { ...cached.result, currentVersion }
    }
  }

  try {
    const result = await fetchLatestRelease(currentVersion)
    writeCache(result)
    return result
  } catch (error) {
    const message = error instanceof Error ? error.message : 'update_check_failed'
    const cached = readCache()
    if (cached) {
      return { ...cached.result, currentVersion, error: message }
    }
    return {
      currentVersion,
      latestVersion: null,
      available: false,
      downloadUrl: fallbackDownloadUrl(),
      releaseUrl: FALLBACK_RELEASE_URL,
      error: message,
    }
  }
}

export async function openUpdateUrl(target?: string) {
  const cached = readCache()
  const candidate =
    (typeof target === 'string' && target.trim()) ||
    cached?.result.downloadUrl ||
    cached?.result.releaseUrl ||
    fallbackDownloadUrl()

  if (!isAllowedUpdateUrl(candidate)) {
    return { ok: false as const, reason: 'blocked_url' }
  }

  await shell.openExternal(candidate)
  return { ok: true as const }
}
