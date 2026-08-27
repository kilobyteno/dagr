import { useEffect, useState } from 'react'

import { updatePresence, type PresenceState } from '@/lib/api/auth'

const HEARTBEAT_MS = 45_000
const IDLE_MS = 5 * 60_000
const OFFLINE_DEBOUNCE_MS = 2_000

/** How often the shell refreshes other people's presence from the member list. */
export const PRESENCE_POLL_MS = 15_000

export function isStatusExpired(expiresAt?: string | null) {
  if (!expiresAt) return false
  const ms = Date.parse(expiresAt)
  if (Number.isNaN(ms)) return false
  return ms <= Date.now()
}

export function hasCustomStatus(
  emoji?: string | null,
  text?: string | null,
  expiresAt?: string | null,
) {
  if (isStatusExpired(expiresAt)) return false
  return Boolean(emoji?.trim() || text?.trim())
}

export function formatCustomStatus(
  emoji?: string | null,
  text?: string | null,
  expiresAt?: string | null,
) {
  if (!hasCustomStatus(emoji, text, expiresAt)) return ''
  const e = emoji?.trim() ?? ''
  const t = text?.trim() ?? ''
  if (e && t) return `${e} ${t}`
  return e || t
}

function navigatorIsOffline() {
  return typeof navigator !== 'undefined' && navigator.onLine === false
}

/** Posts presence heartbeats while the app is open; marks away after idle. */
export function startPresenceHeartbeat(
  serverUrl: string,
  token: string,
  onChange?: (presence: PresenceState) => void,
): () => void {
  let away = false
  let lastActivity = Date.now()
  let stopped = false
  let networkOffline = navigatorIsOffline()
  let offlineTimer: number | null = null

  const setPresence = (next: PresenceState) => {
    onChange?.(next)
  }

  const push = (
    state: 'active' | 'away' | 'offline',
    keepalive = false,
  ) => {
    void updatePresence(serverUrl, token, state, { keepalive }).catch(() => {})
  }

  const markActive = () => {
    lastActivity = Date.now()
    if (networkOffline || stopped) return
    if (away) {
      away = false
      setPresence('online')
      push('active')
    }
  }

  const becomeOffline = (keepalive = false) => {
    away = false
    setPresence('offline')
    push('offline', keepalive)
  }

  const becomeOnline = () => {
    away = false
    lastActivity = Date.now()
    setPresence('online')
    push('active')
  }

  const onActivity = () => {
    markActive()
  }

  const events: Array<keyof WindowEventMap> = [
    'mousemove',
    'mousedown',
    'keydown',
    'touchstart',
    'scroll',
    'focus',
  ]
  for (const event of events) {
    window.addEventListener(event, onActivity, { passive: true })
  }

  const tick = () => {
    if (stopped || networkOffline) return
    const idle = Date.now() - lastActivity >= IDLE_MS
    if (idle && !away) {
      away = true
      setPresence('away')
      push('away')
      return
    }
    if (!idle) {
      push(away ? 'away' : 'active')
    }
  }

  const onOffline = () => {
    networkOffline = true
    if (offlineTimer != null) window.clearTimeout(offlineTimer)
    offlineTimer = window.setTimeout(() => {
      offlineTimer = null
      if (stopped || !networkOffline) return
      becomeOffline()
    }, OFFLINE_DEBOUNCE_MS)
  }

  const onOnline = () => {
    networkOffline = false
    if (offlineTimer != null) {
      window.clearTimeout(offlineTimer)
      offlineTimer = null
    }
    if (stopped) return
    becomeOnline()
  }

  const onPageHide = () => {
    if (stopped) return
    becomeOffline(true)
  }

  const onVisibility = () => {
    if (stopped || networkOffline) return
    if (document.visibilityState === 'visible') {
      markActive()
      setPresence('online')
      push('active')
    }
  }

  if (networkOffline) {
    setPresence('offline')
  } else {
    setPresence('online')
    push('active')
  }

  const intervalId = window.setInterval(tick, HEARTBEAT_MS)
  window.addEventListener('offline', onOffline)
  window.addEventListener('online', onOnline)
  window.addEventListener('pagehide', onPageHide)
  document.addEventListener('visibilitychange', onVisibility)

  return () => {
    stopped = true
    if (offlineTimer != null) window.clearTimeout(offlineTimer)
    window.clearInterval(intervalId)
    window.removeEventListener('offline', onOffline)
    window.removeEventListener('online', onOnline)
    window.removeEventListener('pagehide', onPageHide)
    document.removeEventListener('visibilitychange', onVisibility)
    for (const event of events) {
      window.removeEventListener(event, onActivity)
    }
    setPresence('offline')
    push('offline', true)
  }
}

/** Own live presence while signed in (driven by the heartbeat). */
export function useOwnPresence(
  serverUrl?: string,
  token?: string,
): PresenceState {
  const [presence, setPresence] = useState<PresenceState>('offline')

  useEffect(() => {
    if (!serverUrl || !token) {
      setPresence('offline')
      return
    }
    return startPresenceHeartbeat(serverUrl, token, setPresence)
  }, [serverUrl, token])

  return presence
}

export function presenceDotClass(presence?: PresenceState | string | null) {
  switch (presence) {
    case 'online':
      return 'bg-emerald-500'
    case 'away':
      return 'bg-amber-400'
    default:
      return 'bg-muted-foreground'
  }
}
