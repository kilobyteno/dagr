import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react'

import { isElectron } from '@/lib/desktop'

const DISMISSED_KEY = 'dagr.dismissedUpdateVersion'

export type DesktopUpdateCheck = {
  currentVersion: string
  latestVersion: string | null
  available: boolean
  downloadUrl: string | null
  releaseUrl: string | null
  skipped?: boolean
  error?: string
}

type DesktopUpdateContextValue = {
  status: DesktopUpdateCheck | null
  checking: boolean
  showBanner: boolean
  check: (force?: boolean) => Promise<DesktopUpdateCheck | null>
  dismiss: () => void
  openUpdate: () => Promise<void>
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

async function invokeUpdateCheck(force: boolean): Promise<DesktopUpdateCheck> {
  if (!isElectron() || !window.dagr?.invoke) {
    return {
      currentVersion: '',
      latestVersion: null,
      available: false,
      downloadUrl: null,
      releaseUrl: null,
      skipped: true,
    }
  }
  const result = (await window.dagr.invoke('updates:check', { force })) as
    | DesktopUpdateCheck
    | undefined
  return (
    result ?? {
      currentVersion: '',
      latestVersion: null,
      available: false,
      downloadUrl: null,
      releaseUrl: null,
      error: 'empty_result',
    }
  )
}

export function DesktopUpdateProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<DesktopUpdateCheck | null>(null)
  const [checking, setChecking] = useState(false)
  const [dismissed, setDismissed] = useState<string | null>(readDismissedVersion)

  const check = useCallback(async (force = false) => {
    if (!isElectron()) return null
    setChecking(true)
    try {
      const result = await invokeUpdateCheck(force)
      setStatus(result)
      return result
    } catch (error) {
      const message = error instanceof Error ? error.message : 'invoke_failed'
      console.warn('[dagr] updates:check failed:', message)
      const failed: DesktopUpdateCheck = {
        currentVersion: '',
        latestVersion: null,
        available: false,
        downloadUrl: null,
        releaseUrl: null,
        error: message,
      }
      setStatus((previous) => ({
        ...failed,
        currentVersion: previous?.currentVersion ?? '',
      }))
      return failed
    } finally {
      setChecking(false)
    }
  }, [])

  useEffect(() => {
    void check(false)
  }, [check])

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

  const showBanner = Boolean(
    status?.available &&
      status.latestVersion &&
      status.latestVersion !== dismissed,
  )

  const value = useMemo(
    () => ({ status, checking, showBanner, check, dismiss, openUpdate }),
    [status, checking, showBanner, check, dismiss, openUpdate],
  )

  return (
    <DesktopUpdateContext.Provider value={value}>
      {children}
    </DesktopUpdateContext.Provider>
  )
}

export function useDesktopUpdate() {
  const value = useContext(DesktopUpdateContext)
  if (!value) {
    throw new Error('useDesktopUpdate must be used within DesktopUpdateProvider')
  }
  return value
}
