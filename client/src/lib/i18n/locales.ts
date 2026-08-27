export const APP_LOCALES = ['en-GB', 'nb'] as const

export type AppLocale = (typeof APP_LOCALES)[number]

export const DEFAULT_LOCALE: AppLocale = 'en-GB'

export const LOCALE_LABELS: Record<AppLocale, string> = {
  'en-GB': 'English (UK)',
  nb: 'Norsk (bokmål)',
}

export function isAppLocale(value: unknown): value is AppLocale {
  return APP_LOCALES.includes(value as AppLocale)
}

export function parseAppLocale(value: unknown): AppLocale {
  return isAppLocale(value) ? value : DEFAULT_LOCALE
}

const PREFERENCES_KEY = 'dagr.appPreferences'

export function readStoredLocale(): AppLocale {
  if (typeof window === 'undefined') return DEFAULT_LOCALE
  try {
    const raw = window.localStorage.getItem(PREFERENCES_KEY)
    if (!raw) return DEFAULT_LOCALE
    const parsed = JSON.parse(raw) as { locale?: unknown }
    return parseAppLocale(parsed.locale)
  } catch {
    return DEFAULT_LOCALE
  }
}
