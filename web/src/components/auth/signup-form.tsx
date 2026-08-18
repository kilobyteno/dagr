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
import { useServerPublicConfig } from '@/hooks/use-server-public-config'
import { signup } from '@/lib/api/auth'
import { formatUserError } from '@/lib/api/client'
import { sessionFromAuthResponse, useAuth } from '@/lib/auth'
import { useLocale } from '@/lib/i18n'
import { validatePassword } from '@/lib/password-policy'
import {
  readStoredServerHost,
  resolveServerUrl,
  writeStoredServerHost,
  type StoredServerHost,
} from '@/lib/server-host'

export function SignupForm({ onSwitchToLogin }: { onSwitchToLogin: () => void }) {
  const { t } = useLocale()
  const { signIn } = useAuth()
  const [displayName, setDisplayName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [serverHost, setServerHost] = useState<StoredServerHost>(() =>
    readStoredServerHost(),
  )
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    writeStoredServerHost(serverHost)
  }, [serverHost])

  const serverUrl = resolveServerUrl(serverHost)
  const { config } = useServerPublicConfig(serverUrl)
  const passwordPolicy = config.passwordPolicy

  return (
    <form
      className="flex flex-col gap-6"
      onSubmit={(event) => {
        event.preventDefault()
        if (!displayName.trim() || !email.trim() || !password || !serverUrl) {
          return
        }
        if (password !== confirmPassword) {
          toast.error(t('auth.signup.passwordsMismatch'))
          return
        }
        const policyError = validatePassword(password, passwordPolicy)
        if (policyError) {
          toast.error(
            t(`auth.password.errors.${policyError.id}`, {
              count: policyError.minLength,
            }),
          )
          return
        }
        setSubmitting(true)
        void signup(serverUrl, {
          email: email.trim(),
          password,
          displayName: displayName.trim(),
        })
          .then((response) => {
            signIn(sessionFromAuthResponse(serverUrl, response))
            toast.success(t('auth.signup.success'))
          })
          .catch((error: unknown) => {
            const message =
              formatUserError(error, t('auth.signup.error'))
            toast.error(message)
          })
          .finally(() => setSubmitting(false))
      }}
    >
      <div className="flex flex-col gap-1.5">
        <h1 className="font-heading text-2xl font-semibold tracking-tight">
          {t('auth.signup.title')}
        </h1>
        <p className="text-sm text-muted-foreground">
          {serverHost.mode === 'cloud'
            ? t('auth.signup.cloudSubhead')
            : t('auth.signup.selfHostedSubhead')}
        </p>
      </div>

      <FieldGroup>
        <ServerHostPicker
          value={serverHost}
          onChange={setServerHost}
          serverInputId="signup-server"
        />

        <Field>
          <FieldLabel htmlFor="signup-name">{t('auth.signup.displayName')}</FieldLabel>
          <Input
            id="signup-name"
            type="text"
            autoComplete="name"
            placeholder={t('auth.signup.displayNamePlaceholder')}
            value={displayName}
            onChange={(event) => setDisplayName(event.target.value)}
            required
          />
        </Field>

        <Field>
          <FieldLabel htmlFor="signup-email">{t('common.email')}</FieldLabel>
          <Input
            id="signup-email"
            type="email"
            autoComplete="email"
            placeholder={t('auth.signup.emailPlaceholder')}
            value={email}
            onChange={(event) => setEmail(event.target.value)}
            required
          />
        </Field>

        <Field>
          <FieldLabel htmlFor="signup-password">{t('auth.signup.password')}</FieldLabel>
          <PasswordInput
            id="signup-password"
            autoComplete="new-password"
            value={password}
            onChange={setPassword}
            policy={passwordPolicy}
            required
            minLength={passwordPolicy.minLength}
          />
        </Field>

        <Field>
          <FieldLabel htmlFor="signup-confirm">{t('auth.signup.confirmPassword')}</FieldLabel>
          <PasswordInput
            id="signup-confirm"
            autoComplete="new-password"
            value={confirmPassword}
            onChange={setConfirmPassword}
            showRequirements={false}
            required
            minLength={passwordPolicy.minLength}
            aria-invalid={
              confirmPassword.length > 0 && confirmPassword !== password
                ? true
                : undefined
            }
          />
          {confirmPassword.length > 0 && confirmPassword !== password && (
            <FieldDescription className="text-destructive">
              {t('auth.signup.passwordsMismatchInline')}
            </FieldDescription>
          )}
        </Field>

        <Field>
          <Button type="submit" className="w-full" disabled={submitting}>
            {submitting ? t('auth.signup.submitting') : t('auth.signup.submit')}
          </Button>
        </Field>

        <FieldDescription className="text-center">
          {t('auth.signup.haveAccount')}{' '}
          <button
            type="button"
            className="font-medium text-foreground underline-offset-4 hover:underline"
            onClick={onSwitchToLogin}
          >
            {t('auth.signup.logIn')}
          </button>
        </FieldDescription>
      </FieldGroup>
    </form>
  )
}
