import i18n from 'i18next'
import { useTranslation } from 'react-i18next'

import { setAppPreferences } from '@/lib/app-preferences'
import {
  DEFAULT_LOCALE,
  parseAppLocale,
  readStoredLocale,
  type AppLocale,
} from '@/lib/i18n/locales'
import enGB from '@/locales/en-GB.json'
import nb from '@/locales/nb.json'

void i18n.init({
  resources: {
    'en-GB': { translation: enGB },
    nb: { translation: nb },
  },
  lng: readStoredLocale(),
  fallbackLng: DEFAULT_LOCALE,
  supportedLngs: ['en-GB', 'nb'],
  interpolation: { escapeValue: false },
  returnNull: false,
})

if (typeof document !== 'undefined') {
  document.documentElement.lang = parseAppLocale(i18n.language)
}

export { i18n }

export function applyLocale(locale: AppLocale) {
  if (i18n.resolvedLanguage !== locale) {
    void i18n.changeLanguage(locale)
  }
  if (typeof document !== 'undefined') {
    document.documentElement.lang = locale
  }
  setAppPreferences({ locale })
}

export function useLocale() {
  const { i18n: instance, t } = useTranslation()
  const locale = parseAppLocale(instance.resolvedLanguage ?? instance.language)

  return {
    locale,
    t,
    formatDate(date: Date, options?: Intl.DateTimeFormatOptions) {
      return date.toLocaleDateString(locale, options)
    },
    formatTime(date: Date, options?: Intl.DateTimeFormatOptions) {
      return date.toLocaleTimeString(locale, options)
    },
    formatDateTime(date: Date, options?: Intl.DateTimeFormatOptions) {
      return date.toLocaleString(locale, options)
    },
    compare(a: string, b: string) {
      return a.localeCompare(b, locale)
    },
  }
}

export default i18n
