import { useCallback, useSyncExternalStore } from 'react'

import {
  DEFAULT_LOCALE,
  parseAppLocale,
  type AppLocale,
} from '@/lib/i18n/locales'

const STORAGE_KEY = 'dagr.appPreferences'
const CHANGE_EVENT = 'dagr-app-preferences-change'

export type AppPreferences = {
  /** When true, animated GIFs stay frozen until hovered (or focused). */
  gifsOnHoverOnly: boolean
  locale: AppLocale
}

const DEFAULTS: AppPreferences = {
  gifsOnHoverOnly: false,
  locale: DEFAULT_LOCALE,
}

function readPreferences(): AppPreferences {
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY)
    if (!raw) return { ...DEFAULTS }
    const parsed = JSON.parse(raw) as Partial<AppPreferences>
    return {
      gifsOnHoverOnly: Boolean(parsed.gifsOnHoverOnly),
      locale: parseAppLocale(parsed.locale),
    }
  } catch {
    return { ...DEFAULTS }
  }
}

let snapshot = typeof window === 'undefined' ? { ...DEFAULTS } : readPreferences()

function emitChange() {
  snapshot = readPreferences()
  window.dispatchEvent(new Event(CHANGE_EVENT))
}

export function getAppPreferences(): AppPreferences {
  return snapshot
}

export function setAppPreferences(patch: Partial<AppPreferences>) {
  const next = { ...readPreferences(), ...patch }
  window.localStorage.setItem(STORAGE_KEY, JSON.stringify(next))
  emitChange()
}

function subscribe(onStoreChange: () => void) {
  snapshot = readPreferences()
  const onChange = () => {
    snapshot = readPreferences()
    onStoreChange()
  }
  window.addEventListener(CHANGE_EVENT, onChange)
  window.addEventListener('storage', onChange)
  return () => {
    window.removeEventListener(CHANGE_EVENT, onChange)
    window.removeEventListener('storage', onChange)
  }
}

function getServerSnapshot(): AppPreferences {
  return DEFAULTS
}

export function useAppPreferences() {
  const preferences = useSyncExternalStore(
    subscribe,
    () => snapshot,
    getServerSnapshot,
  )

  const setPreference = useCallback(
    <K extends keyof AppPreferences>(key: K, value: AppPreferences[K]) => {
      setAppPreferences({ [key]: value })
    },
    [],
  )

  return { preferences, setPreference, setPreferences: setAppPreferences }
}

/** Best-effort GIF detection from a URL or data/blob URI. */
export function isLikelyGifUrl(src: string): boolean {
  const value = src.trim()
  if (!value) return false
  if (value.startsWith('data:image/gif')) return true
  try {
    const url = new URL(value, typeof window !== 'undefined' ? window.location.href : 'https://local')
    const path = url.pathname.toLowerCase()
    if (path.endsWith('.gif')) return true
    if (path.includes('.gif/')) return true
    if (url.searchParams.get('format')?.toLowerCase() === 'gif') return true
    if (url.searchParams.get('fm')?.toLowerCase() === 'gif') return true
    const host = url.hostname.toLowerCase()
    const isDomainOrSubdomain = (root: string) =>
      host === root || host.endsWith(`.${root}`)
    if (
      host === 'i.giphy.com' ||
      (isDomainOrSubdomain('giphy.com') && host.startsWith('media'))
    ) {
      return true
    }
    if (isDomainOrSubdomain('tenor.com') && path.includes('gif')) {
      return true
    }
  } catch {
    if (/\.gif(\?|#|$)/i.test(value)) return true
  }
  return false
}
