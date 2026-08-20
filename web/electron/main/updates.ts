import { app, shell } from 'electron'
import { autoUpdater } from 'electron-updater'
import { readFileSync, writeFileSync } from 'node:fs'
import path from 'node:path'

import { compareSemVer, isNewerVersion, stripTagPrefix } from './semver'

const GITHUB_REPO = 'kilobyteno/dagr'
const CHECK_INTERVAL_MS = 24 * 60 * 60 * 1000
const FALLBACK_RELEASE_URL = `https://github.com/${GITHUB_REPO}/releases`

export const UPDATE_CHANNELS = ['stable', 'prerelease'] as const

export type UpdateChannel = (typeof UPDATE_CHANNELS)[number]

export type UpdatePhase =
  | 'idle'
  | 'checking'
  | 'available'
  | 'downloading'
  | 'ready'
  | 'error'

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
  phase: UpdatePhase
  percent: number | null
  canInstall: boolean
}

type UpdateStateListener = (state: UpdateCheckResult) => void

type CachedCheck = {
  checkedAt: number
  channel: UpdateChannel
  result: Omit<UpdateCheckResult, 'phase' | 'percent' | 'canInstall'>
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

const listeners = new Set<UpdateStateListener>()

let updaterConfigured = false
let currentVersion = ''
let currentChannel: UpdateChannel = 'stable'
let latestVersion: string | null = null
let available = false
let downloadUrl: string | null = null
let releaseUrl: string | null = null
let skipped = false
let lastError: string | undefined
let phase: UpdatePhase = 'idle'
let percent: number | null = null
let canInstall = false

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

function persistableResult(result: UpdateCheckResult): CachedCheck['result'] {
  return {
    currentVersion: result.currentVersion,
    latestVersion: result.latestVersion,
    available: result.available,
    downloadUrl: result.downloadUrl,
    releaseUrl: result.releaseUrl,
    channel: result.channel,
    skipped: result.skipped,
    error: result.error,
  }
}

function writeCache(result: UpdateCheckResult) {
  try {
    const payload: CachedCheck = {
      checkedAt: Date.now(),
      channel: result.channel,
      result: persistableResult(result),
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
  version: string,
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
      currentVersion: version,
      latestVersion: null,
      available: false,
      downloadUrl: null,
      releaseUrl: FALLBACK_RELEASE_URL,
      channel,
      phase: 'idle',
      percent: null,
      canInstall: false,
    }
  }

  const nextVersion = stripTagPrefix(newest.tag_name ?? '') || null
  const assets = newest.assets ?? []
  const nextDownloadUrl = downloadUrlForPlatform(assets) || fallbackDownloadUrl()
  const nextReleaseUrl = newest.html_url?.trim() || FALLBACK_RELEASE_URL
  const nextAvailable = Boolean(
    nextVersion && isNewerVersion(nextVersion, version),
  )

  return {
    currentVersion: version,
    latestVersion: nextVersion,
    available: nextAvailable,
    downloadUrl: nextDownloadUrl,
    releaseUrl: nextReleaseUrl,
    channel,
    phase: nextAvailable ? 'available' : 'idle',
    percent: null,
    canInstall: false,
  }
}

function snapshot(): UpdateCheckResult {
  return {
    currentVersion,
    latestVersion,
    available,
    downloadUrl,
    releaseUrl,
    channel: currentChannel,
    skipped: skipped || undefined,
    error: lastError,
    phase,
    percent,
    canInstall,
  }
}

function emitUpdateState() {
  const state = snapshot()
  listeners.forEach((listener) => {
    listener(state)
  })
}

function applyGithubResult(result: UpdateCheckResult) {
  currentVersion = result.currentVersion
  currentChannel = result.channel
  latestVersion = result.latestVersion
  available = result.available
  downloadUrl = result.downloadUrl
  releaseUrl = result.releaseUrl
  skipped = Boolean(result.skipped)
  lastError = result.error
  if (phase !== 'downloading' && phase !== 'ready') {
    phase = result.phase
    percent = result.percent
    canInstall = result.canInstall
  }
}

function applyGithubUrls(result: UpdateCheckResult) {
  downloadUrl = result.downloadUrl
  releaseUrl = result.releaseUrl
}

function resetEphemeralState() {
  skipped = false
  lastError = undefined
  if (phase !== 'downloading' && phase !== 'ready') {
    percent = null
    canInstall = false
  }
}

function canUseAutoUpdater() {
  return app.isPackaged && !process.env.VITE_DEV_SERVER_URL
}

function configureAutoUpdater() {
  if (updaterConfigured) return
  updaterConfigured = true
  autoUpdater.autoDownload = true
  autoUpdater.autoInstallOnAppQuit = true
  autoUpdater.allowDowngrade = false

  autoUpdater.on('checking-for-update', () => {
    if (phase === 'ready' || phase === 'downloading') return
    phase = 'checking'
    emitUpdateState()
  })

  autoUpdater.on('update-available', (info) => {
    const version = stripTagPrefix(info.version ?? '')
    if (version) latestVersion = version
    available = true
    if (phase !== 'ready') {
      phase = 'available'
    }
    lastError = undefined
    emitUpdateState()
  })

  autoUpdater.on('update-not-available', (info) => {
    const version = stripTagPrefix(info.version ?? '')
    if (version) latestVersion = version
    available = false
    canInstall = false
    percent = null
    phase = 'idle'
    lastError = undefined
    emitUpdateState()
  })

  autoUpdater.on('download-progress', (progress) => {
    available = true
    phase = 'downloading'
    percent =
      typeof progress.percent === 'number' && Number.isFinite(progress.percent)
        ? Math.max(0, Math.min(100, progress.percent))
        : null
    emitUpdateState()
  })

  autoUpdater.on('update-downloaded', (info) => {
    const version = stripTagPrefix(info.version ?? '')
    if (version) latestVersion = version
    available = true
    phase = 'ready'
    percent = 100
    canInstall = true
    lastError = undefined
    emitUpdateState()
  })

  autoUpdater.on('error', (error) => {
    const message = error instanceof Error ? error.message : String(error)
    lastError = message
    if (canInstall) {
      canInstall = false
      phase = available ? 'available' : 'error'
      emitUpdateState()
      return
    }
    if (phase === 'downloading' || phase === 'ready' || phase === 'checking') {
      phase = available ? 'available' : 'error'
    }
    emitUpdateState()
  })
}

async function enrichFromGithub(version: string, channel: UpdateChannel) {
  try {
    const github = await fetchLatestRelease(version, channel)
    writeCache(github)
    applyGithubUrls(github)
  } catch {
    if (!downloadUrl) downloadUrl = fallbackDownloadUrl()
    if (!releaseUrl) releaseUrl = FALLBACK_RELEASE_URL
  }
}

async function checkWithAutoUpdater(
  version: string,
  channel: UpdateChannel,
): Promise<boolean> {
  configureAutoUpdater()
  autoUpdater.allowPrerelease = channel === 'prerelease'
  autoUpdater.autoDownload = true
  const result = await autoUpdater.checkForUpdates()
  const infoVersion = stripTagPrefix(result?.updateInfo?.version ?? '')
  if (infoVersion) {
    latestVersion = infoVersion
    available = isNewerVersion(infoVersion, version)
    if (available && phase !== 'ready' && phase !== 'downloading') {
      phase = 'available'
    }
    if (!available) {
      phase = 'idle'
      canInstall = false
      percent = null
    }
  }
  return true
}

export function subscribeUpdateState(listener: UpdateStateListener) {
  listeners.add(listener)
  return () => {
    listeners.delete(listener)
  }
}

export function getUpdateState(): UpdateCheckResult {
  return snapshot()
}

export async function checkForUpdates(
  version: string,
  options: { force?: boolean; channel?: unknown } = {},
): Promise<UpdateCheckResult> {
  const channel = parseUpdateChannel(options.channel)
  currentVersion = version
  currentChannel = channel

  if (process.env.VITE_DEV_SERVER_URL && !options.force) {
    skipped = true
    available = false
    latestVersion = null
    downloadUrl = null
    releaseUrl = null
    lastError = undefined
    phase = 'idle'
    percent = null
    canInstall = false
    return snapshot()
  }

  if (
    !options.force &&
    (phase === 'downloading' || phase === 'ready') &&
    currentChannel === channel
  ) {
    return snapshot()
  }

  resetEphemeralState()

  if (!options.force) {
    const cached = readCache()
    if (
      cached &&
      cached.channel === channel &&
      Date.now() - cached.checkedAt < CHECK_INTERVAL_MS
    ) {
      applyGithubResult({
        ...cached.result,
        currentVersion: version,
        channel,
        phase: cached.result.available ? 'available' : 'idle',
        percent: null,
        canInstall: false,
      })
      if (canUseAutoUpdater() && cached.result.available) {
        try {
          await checkWithAutoUpdater(version, channel)
        } catch (error) {
          lastError = error instanceof Error ? error.message : 'update_check_failed'
        }
      }
      emitUpdateState()
      return snapshot()
    }
  }

  phase = 'checking'
  emitUpdateState()

  if (canUseAutoUpdater()) {
    try {
      await checkWithAutoUpdater(version, channel)
      await enrichFromGithub(version, channel)
      lastError = undefined
      emitUpdateState()
      return snapshot()
    } catch (error) {
      lastError = error instanceof Error ? error.message : 'update_check_failed'
    }
  }

  try {
    const result = await fetchLatestRelease(version, channel)
    writeCache(result)
    applyGithubResult(result)
    emitUpdateState()
    return snapshot()
  } catch (error) {
    const message = error instanceof Error ? error.message : 'update_check_failed'
    const cached = readCache()
    if (cached && cached.channel === channel) {
      applyGithubResult({
        ...cached.result,
        currentVersion: version,
        channel,
        error: message,
        phase: cached.result.available ? 'available' : 'error',
        percent: null,
        canInstall: false,
      })
      emitUpdateState()
      return snapshot()
    }
    available = false
    latestVersion = null
    downloadUrl = fallbackDownloadUrl()
    releaseUrl = FALLBACK_RELEASE_URL
    lastError = message
    phase = 'error'
    percent = null
    canInstall = false
    emitUpdateState()
    return snapshot()
  }
}

export async function openUpdateUrl(target?: string) {
  const cached = readCache()
  const candidate =
    (typeof target === 'string' && target.trim()) ||
    downloadUrl ||
    cached?.result.downloadUrl ||
    releaseUrl ||
    cached?.result.releaseUrl ||
    fallbackDownloadUrl()

  if (!isAllowedUpdateUrl(candidate)) {
    return { ok: false as const, reason: 'blocked_url' }
  }

  await shell.openExternal(candidate)
  return { ok: true as const }
}

export async function installDownloadedUpdate(target?: string) {
  if (canInstall) {
    try {
      setImmediate(() => {
        autoUpdater.quitAndInstall(false, true)
      })
      return { ok: true as const }
    } catch (error) {
      lastError = error instanceof Error ? error.message : 'install_failed'
      canInstall = false
      phase = 'available'
      emitUpdateState()
    }
  }
  return openUpdateUrl(target)
}
