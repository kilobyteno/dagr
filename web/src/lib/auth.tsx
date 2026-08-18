import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from 'react'

import { me, type NotificationLevel } from '@/lib/api/auth'
import { parseAppLocale, type AppLocale } from '@/lib/i18n/locales'
import { ApiError, isServerUnavailable } from '@/lib/api/client'
import { isStatusExpired } from '@/lib/presence'
import {
  ServerConnectionProvider,
  useServerConnection,
} from '@/lib/server-connection'
import { DEFAULT_SELF_HOSTED_URL } from '@/lib/server-host'

export type AuthSession = {
  id: string
  userId: string
  email: string
  displayName: string
  notificationLevel: NotificationLevel
  locale: AppLocale
  emailVerified?: boolean
  statusEmoji?: string
  statusText?: string
  statusExpiresAt?: string | null
  hasAvatar?: boolean
  avatarUpdatedAt?: string | null
  serverUrl: string
  serverLabel?: string
  token: string
  expiresAt: string
}

type AuthContextValue = {
  session: AuthSession | null
  sessions: AuthSession[]
  ready: boolean
  addingAccount: boolean
  setAddingAccount: (value: boolean) => void
  signIn: (session: AuthSession) => void
  switchSession: (sessionId: string) => void
  removeSession: (sessionId: string) => void
  signOut: () => void
}

const STORAGE_KEY = 'dagr.auth.sessions'
const LEGACY_STORAGE_KEY = 'dagr.auth.session'

const AuthContext = createContext<AuthContextValue | null>(null)

function normalizeNotificationLevel(value: unknown): NotificationLevel {
  if (value === 'all' || value === 'mentions' || value === 'nothing') {
    return value
  }
  return 'mentions'
}

export function sessionIdFor(serverUrl: string, userId: string): string {
  return `${serverUrl.trim().replace(/\/$/, '')}|${userId}`
}

function serverLabelFromUrl(serverUrl: string): string {
  try {
    return new URL(serverUrl).host
  } catch {
    return serverUrl
  }
}

function migrateServerUrl(serverUrl: string): string {
  const trimmed = serverUrl.trim().replace(/\/$/, '')
  if (
    import.meta.env.DEV &&
    trimmed === 'http://localhost:8080' &&
    DEFAULT_SELF_HOSTED_URL !== 'http://localhost:8080'
  ) {
    return DEFAULT_SELF_HOSTED_URL
  }
  return trimmed
}

function normalizeSession(parsed: Partial<AuthSession>): AuthSession | null {
  if (
    !parsed?.email ||
    !parsed?.serverUrl ||
    !parsed?.token ||
    !parsed?.userId ||
    !parsed?.expiresAt
  ) {
    return null
  }
  const serverUrl = migrateServerUrl(parsed.serverUrl)
  return {
    ...(parsed as AuthSession),
    id: parsed.id || sessionIdFor(serverUrl, parsed.userId!),
    serverUrl,
    serverLabel: parsed.serverLabel || serverLabelFromUrl(serverUrl),
    notificationLevel: normalizeNotificationLevel(parsed.notificationLevel),
    locale: parseAppLocale(parsed.locale),
    emailVerified: Boolean(parsed.emailVerified),
  }
}

type StoredAuth = {
  sessions: AuthSession[]
  activeId: string | null
}

function readStoredAuth(): StoredAuth {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (raw) {
      const parsed = JSON.parse(raw) as Partial<StoredAuth>
      const sessions = (parsed.sessions ?? [])
        .map((item) => normalizeSession(item))
        .filter(Boolean) as AuthSession[]
      const activeId =
        parsed.activeId && sessions.some((item) => item.id === parsed.activeId)
          ? parsed.activeId
          : (sessions[0]?.id ?? null)
      return { sessions, activeId }
    }
    const legacy = sessionStorage.getItem(LEGACY_STORAGE_KEY)
    if (legacy) {
      const session = normalizeSession(JSON.parse(legacy) as Partial<AuthSession>)
      if (session) {
        return { sessions: [session], activeId: session.id }
      }
    }
  } catch {
    /* ignore */
  }
  return { sessions: [], activeId: null }
}

function writeStoredAuth(sessions: AuthSession[], activeId: string | null) {
  const payload: StoredAuth = { sessions, activeId }
  localStorage.setItem(STORAGE_KEY, JSON.stringify(payload))
  sessionStorage.removeItem(LEGACY_STORAGE_KEY)
}

function isExpired(expiresAt: string): boolean {
  const ms = Date.parse(expiresAt)
  if (Number.isNaN(ms)) return true
  return ms <= Date.now()
}

function AuthSessionBootstrap({ children }: { children: ReactNode }) {
  const initial = readStoredAuth()
  const [sessions, setSessions] = useState<AuthSession[]>(initial.sessions)
  const [activeId, setActiveId] = useState<string | null>(initial.activeId)
  const [ready, setReady] = useState(false)
  const [addingAccount, setAddingAccount] = useState(false)
  const { noteSuccess, noteFailure } = useServerConnection()

  const session = sessions.find((item) => item.id === activeId) ?? null

  const signIn = useCallback(
    (next: AuthSession) => {
      const normalised = normalizeSession(next)
      if (!normalised) return
      setSessions((prev) => {
        const without = prev.filter((item) => item.id !== normalised.id)
        const nextSessions = [...without, normalised]
        writeStoredAuth(nextSessions, normalised.id)
        return nextSessions
      })
      setActiveId(normalised.id)
      setAddingAccount(false)
      noteSuccess()
    },
    [noteSuccess],
  )

  const switchSession = useCallback(
    (sessionId: string) => {
      setSessions((prev) => {
        if (!prev.some((item) => item.id === sessionId)) return prev
        writeStoredAuth(prev, sessionId)
        setActiveId(sessionId)
        return prev
      })
      noteSuccess()
    },
    [noteSuccess],
  )

  const removeSession = useCallback(
    (sessionId: string) => {
      setSessions((prev) => {
        const nextSessions = prev.filter((item) => item.id !== sessionId)
        setActiveId((current) => {
          const nextActive =
            current === sessionId ? (nextSessions[0]?.id ?? null) : current
          writeStoredAuth(nextSessions, nextActive)
          return nextActive
        })
        if (nextSessions.length === 0) {
          setAddingAccount(false)
        }
        return nextSessions
      })
      noteSuccess()
    },
    [noteSuccess],
  )

  const signOut = useCallback(() => {
    if (activeId) {
      removeSession(activeId)
      return
    }
    writeStoredAuth([], null)
    setSessions([])
    setActiveId(null)
    noteSuccess()
  }, [activeId, noteSuccess, removeSession])

  useEffect(() => {
    const stored = readStoredAuth()
    const valid = stored.sessions.filter((item) => !isExpired(item.expiresAt))
    if (valid.length === 0) {
      writeStoredAuth([], null)
      setSessions([])
      setActiveId(null)
      setReady(true)
      return
    }
    const active =
      valid.find((item) => item.id === stored.activeId) ?? valid[0]
    writeStoredAuth(valid, active.id)
    setSessions(valid)
    setActiveId(active.id)

    const controller = new AbortController()
    void Promise.all(
      valid.map(async (item) => {
        try {
          const response = await me(item.serverUrl, item.token, controller.signal)
          return {
            ...item,
            userId: response.user.id,
            email: response.user.email,
            displayName: response.user.displayName,
            notificationLevel: normalizeNotificationLevel(
              response.user.notificationLevel,
            ),
            locale: parseAppLocale(response.user.locale),
            emailVerified: Boolean(response.user.emailVerified),
            statusEmoji: response.user.statusEmoji ?? '',
            statusText: response.user.statusText ?? '',
            statusExpiresAt: response.user.statusExpiresAt ?? null,
            hasAvatar: Boolean(response.user.hasAvatar),
            avatarUpdatedAt: response.user.avatarUpdatedAt ?? null,
            serverLabel: item.serverLabel || serverLabelFromUrl(item.serverUrl),
          } satisfies AuthSession
        } catch (error: unknown) {
          if (controller.signal.aborted) return item
          if (error instanceof ApiError && error.status === 401) {
            return null
          }
          if (item.id === active.id && isServerUnavailable(error)) {
            noteFailure(error)
          }
          return item
        }
      }),
    )
      .then((results) => {
        if (controller.signal.aborted) return
        const nextSessions = results.filter(Boolean) as AuthSession[]
        const nextActive =
          nextSessions.find((item) => item.id === active.id)?.id ??
          nextSessions[0]?.id ??
          null
        writeStoredAuth(nextSessions, nextActive)
        setSessions(nextSessions)
        setActiveId(nextActive)
        if (nextSessions.some((item) => item.id === nextActive)) {
          noteSuccess()
        }
      })
      .finally(() => {
        if (!controller.signal.aborted) setReady(true)
      })

    return () => controller.abort()
  }, [noteFailure, noteSuccess])

  useEffect(() => {
    if (!session?.statusExpiresAt) return
    const expiresAt = Date.parse(session.statusExpiresAt)
    if (Number.isNaN(expiresAt)) return
    const clearStatus = () => {
      if (!isStatusExpired(session.statusExpiresAt)) return
      const next: AuthSession = {
        ...session,
        statusEmoji: '',
        statusText: '',
        statusExpiresAt: null,
      }
      setSessions((prev) => {
        const nextSessions = prev.map((item) =>
          item.id === next.id ? next : item,
        )
        writeStoredAuth(nextSessions, activeId)
        return nextSessions
      })
    }
    const delay = expiresAt - Date.now()
    if (delay <= 0) {
      clearStatus()
      return
    }
    const timer = window.setTimeout(clearStatus, delay)
    return () => window.clearTimeout(timer)
  }, [session, activeId])

  return (
    <AuthContext.Provider
      value={{
        session,
        sessions,
        ready,
        addingAccount,
        setAddingAccount,
        signIn,
        switchSession,
        removeSession,
        signOut,
      }}
    >
      {children}
    </AuthContext.Provider>
  )
}

export function AuthProvider({ children }: { children: ReactNode }) {
  return (
    <ServerConnectionProvider>
      <AuthSessionBootstrap>{children}</AuthSessionBootstrap>
    </ServerConnectionProvider>
  )
}

export function useAuth() {
  const value = useContext(AuthContext)
  if (!value) {
    throw new Error('useAuth must be used within AuthProvider')
  }
  return value
}

export function sessionFromAuthResponse(
  serverUrl: string,
  response: {
    token: string
    expiresAt: string
    user: {
      id: string
      email: string
      displayName: string
      notificationLevel?: NotificationLevel
      locale?: AppLocale
      emailVerified?: boolean
      statusEmoji?: string
      statusText?: string
      statusExpiresAt?: string | null
      hasAvatar?: boolean
      avatarUpdatedAt?: string | null
    }
  },
): AuthSession {
  const normalisedUrl = serverUrl.trim().replace(/\/$/, '')
  return {
    id: sessionIdFor(normalisedUrl, response.user.id),
    userId: response.user.id,
    email: response.user.email,
    displayName: response.user.displayName,
    notificationLevel: normalizeNotificationLevel(
      response.user.notificationLevel,
    ),
    locale: parseAppLocale(response.user.locale),
    emailVerified: Boolean(response.user.emailVerified),
    statusEmoji: response.user.statusEmoji ?? '',
    statusText: response.user.statusText ?? '',
    statusExpiresAt: response.user.statusExpiresAt ?? null,
    hasAvatar: Boolean(response.user.hasAvatar),
    avatarUpdatedAt: response.user.avatarUpdatedAt ?? null,
    serverUrl: normalisedUrl,
    serverLabel: serverLabelFromUrl(normalisedUrl),
    token: response.token,
    expiresAt: response.expiresAt,
  }
}
