import { app, shell } from 'electron'
import { readFileSync, writeFileSync } from 'node:fs'
import path from 'node:path'

import { compareSemVer, isNewerVersion, stripTagPrefix } from './semver'

const GITHUB_REPO = 'kilobyteno/dagr'
const CHECK_INTERVAL_MS = 24 * 60 * 60 * 1000
const FALLBACK_RELEASE_URL = `https://github.com/${GITHUB_REPO}/releases`

export const UPDATE_CHANNELS = ['stable', 'prerelease'] as const

export type UpdateChannel = (typeof UPDATE_CHANNELS)[number]

export function parseUpdateChannel(value: unknown): UpdateChannel {
  return value === 'prerelease' ? 'prerelease' : 'stable'
}

export type UpdateCheckResult = {
  currentVersion: string
  latestVersion: string | null
  available: boolean
  downloadUrl: string | null
  releaseUrl: string | null
  channel: UpdateChannel
  skipped?: boolean
  error?: string
}

type CachedCheck = {
  checkedAt: number
  channel: UpdateChannel
  result: UpdateCheckResult
}

type GitHubAsset = {
  name?: string
  browser_download_url?: string
}

type GitHubRelease = {
  tag_name?: string
  html_url?: string
  draft?: boolean
  prerelease?: boolean
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
    const payload: CachedCheck = {
      checkedAt: Date.now(),
      channel: result.channel,
      result,
    }
    writeFileSync(cachePath(), JSON.stringify(payload), 'utf8')
  } catch {
    // Cache is best effort.
  }
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

function pickNewestPublishedRelease(
  releases: GitHubRelease[],
  channel: UpdateChannel,
): GitHubRelease | null {
  let best: GitHubRelease | null = null
  let bestVersion: string | null = null
  for (const release of releases) {
    if (release.draft) continue
    if (channel === 'stable' && release.prerelease) continue
    const version = stripTagPrefix(release.tag_name ?? '')
    if (!version) continue
    if (
      !bestVersion ||
      (compareSemVer(version, bestVersion) ?? 0) > 0
    ) {
      best = release
      bestVersion = version
    }
  }
  return best
}

function githubAuthToken() {
  return (
    process.env.DAGR_GITHUB_TOKEN?.trim() ||
    process.env.GITHUB_TOKEN?.trim() ||
    ''
  )
}

async function fetchLatestRelease(
  currentVersion: string,
  channel: UpdateChannel,
): Promise<UpdateCheckResult> {
  const headers: Record<string, string> = {
    Accept: 'application/vnd.github+json',
    'User-Agent': 'dagr-desktop',
    'X-GitHub-Api-Version': '2022-11-28',
  }
  const token = githubAuthToken()
  if (token) {
    headers.Authorization = `Bearer ${token}`
  }

  const response = await fetch(
    `https://api.github.com/repos/${GITHUB_REPO}/releases?per_page=30`,
    { headers },
  )

  if (!response.ok) {
    throw new Error(`GitHub releases returned ${response.status}`)
  }

  const payload = (await response.json()) as GitHubRelease[]
  const newest = Array.isArray(payload)
    ? pickNewestPublishedRelease(payload, channel)
    : null

  if (!newest) {
    return {
      currentVersion,
      latestVersion: null,
      available: false,
      downloadUrl: null,
      releaseUrl: FALLBACK_RELEASE_URL,
      channel,
    }
  }

  const latestVersion = stripTagPrefix(newest.tag_name ?? '') || null
  const assets = newest.assets ?? []
  const downloadUrl = downloadUrlForPlatform(assets) || fallbackDownloadUrl()
  const releaseUrl = newest.html_url?.trim() || FALLBACK_RELEASE_URL

  return {
    currentVersion,
    latestVersion,
    available: Boolean(latestVersion && isNewerVersion(latestVersion, currentVersion)),
    downloadUrl,
    releaseUrl,
    channel,
  }
}

export async function checkForUpdates(
  currentVersion: string,
  options: { force?: boolean; channel?: unknown } = {},
): Promise<UpdateCheckResult> {
  const channel = parseUpdateChannel(options.channel)

  if (process.env.VITE_DEV_SERVER_URL && !options.force) {
    return {
      currentVersion,
      latestVersion: null,
      available: false,
      downloadUrl: null,
      releaseUrl: null,
      channel,
      skipped: true,
    }
  }

  if (!options.force) {
    const cached = readCache()
    if (
      cached &&
      cached.channel === channel &&
      Date.now() - cached.checkedAt < CHECK_INTERVAL_MS
    ) {
      return { ...cached.result, currentVersion, channel }
    }
  }

  try {
    const result = await fetchLatestRelease(currentVersion, channel)
    writeCache(result)
    return result
  } catch (error) {
    const message = error instanceof Error ? error.message : 'update_check_failed'
    const cached = readCache()
    if (cached && cached.channel === channel) {
      return { ...cached.result, currentVersion, channel, error: message }
    }
    return {
      currentVersion,
      latestVersion: null,
      available: false,
      downloadUrl: fallbackDownloadUrl(),
      releaseUrl: FALLBACK_RELEASE_URL,
      channel,
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
