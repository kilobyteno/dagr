import { useEffect, useState } from 'react'
import { toast } from 'sonner'

import { PasswordInput } from '@/components/auth/password-input'
import { ServerHostPicker } from '@/components/auth/server-host-picker'
import { Button } from '@/components/ui/button'
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { login } from '@/lib/api/auth'
import { formatUserError } from '@/lib/api/client'
import { sessionFromAuthResponse, useAuth } from '@/lib/auth'
import { useLocale } from '@/lib/i18n'
import {
  readStoredServerHost,
  resolveServerUrl,
  writeStoredServerHost,
  type StoredServerHost,
} from '@/lib/server-host'

export function LoginForm({ onSwitchToSignup }: { onSwitchToSignup: () => void }) {
  const { t } = useLocale()
  const { signIn } = useAuth()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [serverHost, setServerHost] = useState<StoredServerHost>(() =>
    readStoredServerHost(),
  )
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    writeStoredServerHost(serverHost)
  }, [serverHost])

  const serverUrl = resolveServerUrl(serverHost)

  return (
    <form
      className="flex flex-col gap-6"
      onSubmit={(event) => {
        event.preventDefault()
        if (!email.trim() || !password || !serverUrl) return
        setSubmitting(true)
        void login(serverUrl, {
          email: email.trim(),
          password,
        })
          .then((response) => {
            signIn(sessionFromAuthResponse(serverUrl, response))
            toast.success(t('auth.login.success'))
          })
          .catch((error: unknown) => {
            const message =
              formatUserError(error, t('auth.login.error'))
            toast.error(message)
          })
          .finally(() => setSubmitting(false))
      }}
    >
      <div className="flex flex-col gap-1.5">
        <h1 className="font-heading text-2xl font-semibold tracking-tight">
          {t('auth.login.title')}
        </h1>
        <p className="text-sm text-muted-foreground">
          {serverHost.mode === 'cloud'
            ? t('auth.login.cloudSubhead')
            : t('auth.login.selfHostedSubhead')}
        </p>
      </div>

      <FieldGroup>
        <ServerHostPicker
          value={serverHost}
          onChange={setServerHost}
          serverInputId="login-server"
        />

        <Field>
          <FieldLabel htmlFor="login-email">{t('common.email')}</FieldLabel>
          <Input
            id="login-email"
            type="email"
            autoComplete="email"
            placeholder={t('auth.signup.emailPlaceholder')}
            value={email}
            onChange={(event) => setEmail(event.target.value)}
            required
          />
        </Field>

        <Field>
          <div className="flex items-center gap-2">
            <FieldLabel htmlFor="login-password">{t('auth.signup.password')}</FieldLabel>
            <button
              type="button"
              className="ml-auto text-sm text-muted-foreground underline-offset-4 hover:text-foreground hover:underline"
              onClick={() =>
                toast.message(t('auth.login.forgotTitle'), {
                  description: t('auth.login.forgotDescription'),
                })
              }
            >
              {t('auth.login.forgotPassword')}
            </button>
          </div>
          <PasswordInput
            id="login-password"
            autoComplete="current-password"
            value={password}
            onChange={setPassword}
            showRequirements={false}
            required
          />
        </Field>

        <Field>
          <Button type="submit" className="w-full" disabled={submitting}>
            {submitting ? t('auth.login.submitting') : t('auth.login.submit')}
          </Button>
        </Field>

        <FieldDescription className="text-center">
          {t('auth.login.newHere')}{' '}
          <button
            type="button"
            className="font-medium text-foreground underline-offset-4 hover:underline"
            onClick={onSwitchToSignup}
          >
            {t('auth.login.createAccount')}
          </button>
        </FieldDescription>
      </FieldGroup>
    </form>
  )
}
