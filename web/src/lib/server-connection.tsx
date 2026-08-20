import {
  CloudSlashIcon,
  SpinnerIcon,
  WarningCircleIcon,
} from '@phosphor-icons/react'
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react'

import { Button } from '@/components/ui/button'
import { isServerUnavailable } from '@/lib/api/client'
import { cn } from '@/lib/utils'

type ServerConnectionContextValue = {
  offline: boolean
  retrying: boolean
  onlineEpoch: number
  noteSuccess: () => void
  noteFailure: (error: unknown) => void
  runRetry: (action: () => Promise<void>) => Promise<void>
}

const ServerConnectionContext =
  createContext<ServerConnectionContextValue | null>(null)

function readNavigatorOffline() {
  return typeof navigator !== 'undefined' && navigator.onLine === false
}

export function ServerConnectionProvider({ children }: { children: ReactNode }) {
  const [serverUnreachable, setServerUnreachable] = useState(false)
  const [networkOffline, setNetworkOffline] = useState(readNavigatorOffline)
  const [onlineEpoch, setOnlineEpoch] = useState(0)
  const [retrying, setRetrying] = useState(false)

  useEffect(() => {
    const onOffline = () => setNetworkOffline(true)
    const onOnline = () => {
      setNetworkOffline(false)
      setOnlineEpoch((value) => value + 1)
    }
    window.addEventListener('offline', onOffline)
    window.addEventListener('online', onOnline)
    return () => {
      window.removeEventListener('offline', onOffline)
      window.removeEventListener('online', onOnline)
    }
  }, [])

  const noteSuccess = useCallback(() => {
    setServerUnreachable(false)
  }, [])

  const noteFailure = useCallback((error: unknown) => {
    if (isServerUnavailable(error)) {
      setServerUnreachable(true)
    }
  }, [])

  const runRetry = useCallback(async (action: () => Promise<void>) => {
    setRetrying(true)
    try {
      await action()
    } finally {
      setRetrying(false)
    }
  }, [])

  const offline = networkOffline || serverUnreachable

  const value = useMemo(
    () => ({
      offline,
      retrying,
      onlineEpoch,
      noteSuccess,
      noteFailure,
      runRetry,
    }),
    [offline, retrying, onlineEpoch, noteSuccess, noteFailure, runRetry],
  )

  return (
    <ServerConnectionContext.Provider value={value}>
      {children}
    </ServerConnectionContext.Provider>
  )
}

export function useServerConnection() {
  const value = useContext(ServerConnectionContext)
  if (!value) {
    throw new Error(
      'useServerConnection must be used within ServerConnectionProvider',
    )
  }
  return value
}

export function ServerConnectionBanner({
  onRetry,
  className,
}: {
  onRetry?: () => Promise<void>
  className?: string
}) {
  const { offline, retrying, noteSuccess, noteFailure, runRetry } =
    useServerConnection()

  if (!offline) return null

  return (
    <div
      role="alert"
      className={cn(
        'flex shrink-0 items-center gap-3 border-b border-destructive/20 bg-destructive/10 px-4 py-2 text-sm text-foreground',
        className,
      )}
    >
      <CloudSlashIcon
        strokeWidth={2}
        className="size-4 shrink-0 text-destructive"
        aria-hidden
      />
      <p className="min-w-0 flex-1 text-pretty">
        Could not reach the Dagr server. Showing the last data we have until the
        connection returns.
      </p>
      {onRetry ? (
        <Button
          type="button"
          size="sm"
          variant="outline"
          className="shrink-0"
          disabled={retrying}
          onClick={() => {
            void runRetry(async () => {
              try {
                await onRetry()
                noteSuccess()
              } catch (error) {
                noteFailure(error)
              }
            })
          }}
        >
          {retrying ? (
            <>
              <SpinnerIcon strokeWidth={2} className="size-3.5 animate-spin" />
              Trying…
            </>
          ) : (
            'Try again'
          )}
        </Button>
      ) : null}
    </div>
  )
}

export function EmailVerificationBanner({
  visible,
  resending,
  onResend,
  onOpenProfile,
  className,
}: {
  visible: boolean
  resending?: boolean
  onResend?: () => void
  onOpenProfile?: () => void
  className?: string
}) {
  if (!visible) return null

  return (
    <div
      role="alert"
      className={cn(
        'flex shrink-0 items-center gap-3 border-b border-amber-500/25 bg-amber-500/10 px-4 py-2 text-sm text-foreground',
        className,
      )}
    >
      <WarningCircleIcon
        strokeWidth={2}
        className="size-4 shrink-0 text-amber-700 dark:text-amber-400"
        aria-hidden
      />
      <p className="min-w-0 flex-1 text-pretty">
        Verify your email address. Check your inbox for a verification link, or
        request a new one.
      </p>
      <div className="flex shrink-0 items-center gap-2">
        {onOpenProfile ? (
          <Button
            type="button"
            size="sm"
            variant="ghost"
            onClick={onOpenProfile}
          >
            Open profile
          </Button>
        ) : null}
        {onResend ? (
          <Button
            type="button"
            size="sm"
            variant="outline"
            disabled={resending}
            onClick={onResend}
          >
            {resending ? (
              <>
                <SpinnerIcon strokeWidth={2} className="size-3.5 animate-spin" />
                Sending…
              </>
            ) : (
              'Resend email'
            )}
          </Button>
        ) : null}
      </div>
    </div>
  )
}
