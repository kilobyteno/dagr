import { useEffect, useState } from 'react'

import { updatePresence, type PresenceState } from '@/lib/api/auth'

const HEARTBEAT_MS = 45_000
const IDLE_MS = 5 * 60_000

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

/** Posts presence heartbeats while the app is open; marks away after idle. */
export function startPresenceHeartbeat(
  serverUrl: string,
  token: string,
  onChange?: (presence: PresenceState) => void,
): () => void {
  let away = false
  let lastActivity = Date.now()
  let stopped = false

  const setPresence = (next: PresenceState) => {
    onChange?.(next)
  }

  const markActive = () => {
    lastActivity = Date.now()
    if (away) {
      away = false
      setPresence('online')
      void updatePresence(serverUrl, token, 'active').catch(() => {})
    }
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
    if (stopped) return
    const idle = Date.now() - lastActivity >= IDLE_MS
    if (idle && !away) {
      away = true
      setPresence('away')
      void updatePresence(serverUrl, token, 'away').catch(() => {})
      return
    }
    if (!idle) {
      void updatePresence(serverUrl, token, away ? 'away' : 'active').catch(
        () => {},
      )
    }
  }

  setPresence('online')
  void updatePresence(serverUrl, token, 'active').catch(() => {})
  const intervalId = window.setInterval(tick, HEARTBEAT_MS)

  const onVisibility = () => {
    if (document.visibilityState === 'visible') {
      markActive()
      setPresence('online')
      void updatePresence(serverUrl, token, 'active').catch(() => {})
    }
  }
  document.addEventListener('visibilitychange', onVisibility)

  return () => {
    stopped = true
    window.clearInterval(intervalId)
    document.removeEventListener('visibilitychange', onVisibility)
    for (const event of events) {
      window.removeEventListener(event, onActivity)
    }
    setPresence('offline')
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
