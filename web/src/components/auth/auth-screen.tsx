import { useState } from 'react'

import { DagrMark } from '@/components/auth/dagr-mark'
import { LoginForm } from '@/components/auth/login-form'
import { SignupForm } from '@/components/auth/signup-form'
import { TitleBar } from '@/components/desktop/title-bar'
import { Button } from '@/components/ui/button'
import { useLocale } from '@/lib/i18n'
import { useDesktopUpdate } from '@/lib/updates'

type AuthMode = 'login' | 'signup'

export function AuthScreen({
  addingAccount = false,
  onCancelAddAccount,
}: {
  addingAccount?: boolean
  onCancelAddAccount?: () => void
} = {}) {
  const { t } = useLocale()
  const { status } = useDesktopUpdate()
  const version = status?.currentVersion?.trim() || ''
  const footer = version
    ? t('auth.footerVersion', { version })
    : t('auth.footer')
  const [mode, setMode] = useState<AuthMode>('login')

  return (
    <div className="flex h-full min-h-0 flex-col overflow-hidden bg-background">
      <TitleBar
        title="Dagr"
        brandMark={<DagrMark className="size-4" />}
        showNavigation={false}
      />

      <div className="relative flex min-h-0 flex-1 overflow-hidden">
        <aside className="relative hidden w-[46%] min-w-0 flex-col justify-between overflow-hidden border-r p-10 lg:flex">
          <div
            className="pointer-events-none absolute inset-0 bg-[radial-gradient(120%_80%_at_0%_0%,color-mix(in_oklab,var(--primary)_42%,transparent),transparent_55%),radial-gradient(90%_70%_at_100%_100%,color-mix(in_oklab,var(--primary)_22%,transparent),transparent_50%),linear-gradient(160deg,oklch(0.98_0.01_60)_0%,oklch(0.96_0.02_45)_45%,oklch(0.94_0.03_40)_100%)] dark:bg-[radial-gradient(120%_80%_at_0%_0%,color-mix(in_oklab,var(--primary)_35%,transparent),transparent_55%),radial-gradient(90%_70%_at_100%_100%,color-mix(in_oklab,var(--primary)_18%,transparent),transparent_50%),linear-gradient(160deg,oklch(0.22_0.02_40)_0%,oklch(0.18_0.015_45)_100%)]"
            aria-hidden
          />
          <div
            className="pointer-events-none absolute inset-0 opacity-[0.35] [background-image:linear-gradient(color-mix(in_oklab,var(--foreground)_8%,transparent)_1px,transparent_1px),linear-gradient(90deg,color-mix(in_oklab,var(--foreground)_8%,transparent)_1px,transparent_1px)] [background-size:28px_28px] [mask-image:radial-gradient(ellipse_at_center,black_20%,transparent_75%)]"
            aria-hidden
          />

          <div className="relative z-10 flex items-center gap-3 animate-in fade-in slide-in-from-left-2 duration-500">
            <DagrMark />
            <span className="font-heading text-xl font-semibold tracking-tight">
              Dagr
            </span>
          </div>

          <div className="relative z-10 flex max-w-md flex-col gap-4 animate-in fade-in slide-in-from-bottom-3 duration-700 fill-mode-both">
            <p className="font-heading text-4xl font-semibold tracking-tight text-balance xl:text-5xl">
              {addingAccount
                ? t('auth.addAccountHeadline')
                : t('auth.headline')}
            </p>
            <p className="text-base text-muted-foreground text-pretty">
              {addingAccount
                ? t('auth.addAccountSubhead')
                : t('auth.subhead')}
            </p>
          </div>

          <p className="relative z-10 text-xs text-muted-foreground animate-in fade-in duration-700 delay-150">
            {footer}
          </p>
        </aside>

        <main className="relative flex min-h-0 flex-1 flex-col items-center justify-center overflow-y-auto px-6 py-10 sm:px-10">
          <div className="mb-8 flex items-center gap-3 lg:hidden animate-in fade-in duration-500">
            <DagrMark className="size-9" />
            <span className="font-heading text-2xl font-semibold tracking-tight">
              Dagr
            </span>
          </div>

          <div
            key={mode}
            className="w-full max-w-sm animate-in fade-in slide-in-from-bottom-2 duration-500"
          >
            {addingAccount ? (
              <div className="mb-4 flex items-center justify-between gap-2">
                <p className="text-sm text-muted-foreground">{t('auth.addingAccount')}</p>
                {onCancelAddAccount ? (
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onClick={onCancelAddAccount}
                  >
                    {t('common.cancel')}
                  </Button>
                ) : null}
              </div>
            ) : null}
            {mode === 'login' ? (
              <LoginForm onSwitchToSignup={() => setMode('signup')} />
            ) : (
              <SignupForm onSwitchToLogin={() => setMode('login')} />
            )}
            {version ? (
              <p className="mt-8 text-center text-xs text-muted-foreground lg:hidden">
                {t('auth.version', { version })}
              </p>
            ) : null}
          </div>
        </main>
      </div>
    </div>
  )
}
