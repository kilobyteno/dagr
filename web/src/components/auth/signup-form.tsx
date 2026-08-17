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
import { ApiError } from '@/lib/api/client'
import { sessionFromAuthResponse, useAuth } from '@/lib/auth'
import { validatePassword } from '@/lib/password-policy'
import {
  readStoredServerHost,
  resolveServerUrl,
  writeStoredServerHost,
  type StoredServerHost,
} from '@/lib/server-host'

export function SignupForm({ onSwitchToLogin }: { onSwitchToLogin: () => void }) {
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
          toast.error('Passwords do not match')
          return
        }
        const policyError = validatePassword(password, passwordPolicy)
        if (policyError) {
          toast.error(policyError)
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
            toast.success('Account created')
          })
          .catch((error: unknown) => {
            const message =
              error instanceof ApiError
                ? error.message
                : 'Could not create account'
            toast.error(message)
          })
          .finally(() => setSubmitting(false))
      }}
    >
      <div className="flex flex-col gap-1.5">
        <h1 className="font-heading text-2xl font-semibold tracking-tight">
          Create your account
        </h1>
        <p className="text-sm text-muted-foreground">
          {serverHost.mode === 'cloud'
            ? 'Join Dagr Cloud. Channels, DMs, and files stay with your organisation.'
            : 'Join a self-hosted workspace. Your data stays on your server.'}
        </p>
      </div>

      <FieldGroup>
        <ServerHostPicker
          value={serverHost}
          onChange={setServerHost}
          serverInputId="signup-server"
        />

        <Field>
          <FieldLabel htmlFor="signup-name">Display name</FieldLabel>
          <Input
            id="signup-name"
            type="text"
            autoComplete="name"
            placeholder="Avery Chen"
            value={displayName}
            onChange={(event) => setDisplayName(event.target.value)}
            required
          />
        </Field>

        <Field>
          <FieldLabel htmlFor="signup-email">Email</FieldLabel>
          <Input
            id="signup-email"
            type="email"
            autoComplete="email"
            placeholder="you@company.com"
            value={email}
            onChange={(event) => setEmail(event.target.value)}
            required
          />
        </Field>

        <Field>
          <FieldLabel htmlFor="signup-password">Password</FieldLabel>
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
          <FieldLabel htmlFor="signup-confirm">Confirm password</FieldLabel>
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
              Passwords do not match.
            </FieldDescription>
          )}
        </Field>

        <Field>
          <Button type="submit" className="w-full" disabled={submitting}>
            {submitting ? 'Creating account…' : 'Sign up'}
          </Button>
        </Field>

        <FieldDescription className="text-center">
          Already have an account?{' '}
          <button
            type="button"
            className="font-medium text-foreground underline-offset-4 hover:underline"
            onClick={onSwitchToLogin}
          >
            Log in
          </button>
        </FieldDescription>
      </FieldGroup>
    </form>
  )
}
