import { useEffect } from 'react'
import { I18nextProvider } from 'react-i18next'
import { ThemeProvider, useTheme } from 'next-themes'

import { AppLoadingScreen } from '@/components/app-loading-screen'
import { AuthScreen } from '@/components/auth/auth-screen'
import { ChatShell } from '@/components/chat/chat-shell'
import { Toaster } from '@/components/ui/sonner'
import { TooltipProvider } from '@/components/ui/tooltip'
import { AuthProvider, useAuth } from '@/lib/auth'
import { pushDeepLink } from '@/lib/deep-link'
import { onDesktopDeepLink, setDesktopTheme } from '@/lib/desktop'
import { applyLocale, i18n, useLocale } from '@/lib/i18n'
import { isAppLocale } from '@/lib/i18n/locales'

function DesktopDeepLinkSync() {
  useEffect(() => onDesktopDeepLink(pushDeepLink), [])
  return null
}

function DesktopThemeSync() {
  const { resolvedTheme } = useTheme()

  useEffect(() => {
    if (resolvedTheme !== 'light' && resolvedTheme !== 'dark') return
    void setDesktopTheme(resolvedTheme)
  }, [resolvedTheme])

  return null
}

function LocaleSync() {
  const { session, ready } = useAuth()

  useEffect(() => {
    if (!ready) return
    if (session?.locale && isAppLocale(session.locale)) {
      applyLocale(session.locale)
    }
  }, [ready, session?.id, session?.locale])

  return null
}

function AppContent() {
  const { session, ready, addingAccount, setAddingAccount, sessions } = useAuth()
  const { t } = useLocale()

  useEffect(() => {
    if (!ready) return
    postMessage({ payload: 'removeLoading' }, '*')
  }, [ready])

  if (!ready) {
    return <AppLoadingScreen label={t('app.starting')} />
  }

  const showAuth = !session || addingAccount

  return (
    <div className="h-full min-h-0 overflow-hidden">
      {showAuth ? (
        <AuthScreen
          addingAccount={addingAccount && sessions.length > 0}
          onCancelAddAccount={
            addingAccount && sessions.length > 0
              ? () => setAddingAccount(false)
              : undefined
          }
        />
      ) : (
        <ChatShell />
      )}
    </div>
  )
}

export default function App() {
  return (
    <I18nextProvider i18n={i18n}>
      <ThemeProvider attribute="class" defaultTheme="system" enableSystem disableTransitionOnChange>
        <AuthProvider>
          <TooltipProvider>
            <DesktopThemeSync />
            <DesktopDeepLinkSync />
            <LocaleSync />
            <AppContent />
            <Toaster position="top-center" />
          </TooltipProvider>
        </AuthProvider>
      </ThemeProvider>
    </I18nextProvider>
  )
}
