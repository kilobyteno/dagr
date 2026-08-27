import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react'

import {
  parseUpdateChannel,
  useAppPreferences,
  type UpdateChannel,
} from '@/lib/app-preferences'
import { isElectron } from '@/lib/desktop'

const DISMISSED_KEY = 'dagr.dismissedUpdateVersion'

export type UpdatePhase =
  | 'idle'
  | 'checking'
  | 'available'
  | 'downloading'
  | 'ready'
  | 'error'

export type DesktopUpdateCheck = {
  currentVersion: string
  latestVersion: string | null
  available: boolean
  downloadUrl: string | null
  releaseUrl: string | null
  channel?: UpdateChannel
  skipped?: boolean
  error?: string
  phase?: UpdatePhase
  percent?: number | null
  canInstall?: boolean
}

type DesktopUpdateContextValue = {
  status: DesktopUpdateCheck | null
  checking: boolean
  showBanner: boolean
  check: (force?: boolean) => Promise<DesktopUpdateCheck | null>
  dismiss: () => void
  openUpdate: () => Promise<void>
  install: () => Promise<void>
}

const DesktopUpdateContext = createContext<DesktopUpdateContextValue | null>(
  null,
)

function readDismissedVersion() {
  if (typeof window === 'undefined') return null
  try {
    return window.localStorage.getItem(DISMISSED_KEY)
  } catch {
    return null
  }
}

function isUpdateCheck(value: unknown): value is DesktopUpdateCheck {
  if (!value || typeof value !== 'object') return false
  const record = value as Record<string, unknown>
  return typeof record.currentVersion === 'string'
}

function emptyCheck(
  channel: UpdateChannel,
  extras: Partial<DesktopUpdateCheck> = {},
): DesktopUpdateCheck {
  return {
    currentVersion: '',
    latestVersion: null,
    available: false,
    downloadUrl: null,
    releaseUrl: null,
    channel,
    phase: extras.phase ?? 'idle',
    percent: extras.percent ?? null,
    canInstall: extras.canInstall ?? false,
    ...extras,
  }
}

async function invokeUpdateCheck(
  force: boolean,
  channel: UpdateChannel,
): Promise<DesktopUpdateCheck> {
  if (!isElectron() || !window.dagr?.invoke) {
    return emptyCheck(channel, { skipped: true })
  }
  const result = await window.dagr.invoke('updates:check', {
    force,
    channel,
  })
  return isUpdateCheck(result)
    ? result
    : emptyCheck(channel, { error: 'empty_result', phase: 'error' })
}

export function DesktopUpdateProvider({ children }: { children: ReactNode }) {
  const { preferences } = useAppPreferences()
  const channel = parseUpdateChannel(preferences.updateChannel)
  const [status, setStatus] = useState<DesktopUpdateCheck | null>(null)
  const [checking, setChecking] = useState(false)
  const [dismissed, setDismissed] = useState<string | null>(readDismissedVersion)

  const check = useCallback(async (force = false) => {
    if (!isElectron()) return null
    setChecking(true)
    try {
      const result = await invokeUpdateCheck(force, channel)
      setStatus(result)
      return result
    } catch (error) {
      const message = error instanceof Error ? error.message : 'invoke_failed'
      console.warn('[dagr] updates:check failed:', message)
      const failed = emptyCheck(channel, {
        error: message,
        phase: 'error',
      })
      setStatus((previous) => ({
        ...failed,
        currentVersion: previous?.currentVersion ?? '',
      }))
      return failed
    } finally {
      setChecking(false)
    }
  }, [channel])

  useEffect(() => {
    void check(false)
  }, [check])

  useEffect(() => {
    if (!isElectron() || !window.dagr?.onUpdateState) return
    return window.dagr.onUpdateState((payload) => {
      if (!isUpdateCheck(payload)) return
      setStatus(payload)
      if (payload.phase === 'checking') {
        setChecking(true)
        return
      }
      if (payload.phase !== 'downloading') {
        setChecking(false)
      }
    })
  }, [])

  const dismiss = useCallback(() => {
    const version = status?.latestVersion
    if (!version) return
    try {
      window.localStorage.setItem(DISMISSED_KEY, version)
    } catch {
      // Ignore quota / private mode.
    }
    setDismissed(version)
  }, [status?.latestVersion])

  const openUpdate = useCallback(async () => {
    if (!isElectron() || !window.dagr?.invoke) return
    try {
      await window.dagr.invoke('updates:open', status?.downloadUrl ?? undefined)
    } catch (error) {
      const reason = error instanceof Error ? error.message : 'invoke_failed'
      console.warn('[dagr] updates:open failed:', reason)
    }
  }, [status?.downloadUrl])

  const install = useCallback(async () => {
    if (!isElectron() || !window.dagr?.invoke) return
    try {
      await window.dagr.invoke('updates:install', status?.downloadUrl ?? undefined)
    } catch (error) {
      const reason = error instanceof Error ? error.message : 'invoke_failed'
      console.warn('[dagr] updates:install failed:', reason)
    }
  }, [status?.downloadUrl])

  const showBanner = Boolean(
    (status?.available ||
      status?.phase === 'downloading' ||
      status?.phase === 'ready' ||
      status?.canInstall) &&
      status.latestVersion &&
      status.latestVersion !== dismissed,
  )

  const value = useMemo(
    () => ({
      status,
      checking: checking || status?.phase === 'checking',
      showBanner,
      check,
      dismiss,
      openUpdate,
      install,
    }),
    [status, checking, showBanner, check, dismiss, openUpdate, install],
  )

  return (
    <DesktopUpdateContext.Provider value={value}>
      {children}
    </DesktopUpdateContext.Provider>
  )
}

export function isUpdateReady(status: DesktopUpdateCheck | null) {
  return Boolean(status?.canInstall || status?.phase === 'ready')
}

export function isUpdateDownloading(status: DesktopUpdateCheck | null) {
  return status?.phase === 'downloading'
}

export function updateDownloadPercent(status: DesktopUpdateCheck | null) {
  const value = status?.percent
  if (typeof value !== 'number' || !Number.isFinite(value)) return null
  return Math.max(0, Math.min(100, Math.round(value)))
}

export function useDesktopUpdate() {
  const value = useContext(DesktopUpdateContext)
  if (!value) {
    throw new Error('useDesktopUpdate must be used within DesktopUpdateProvider')
  }
  return value
}
