import type { PasswordPolicy } from '@/lib/password-policy'

/** Client fallbacks aligned with server defaults when the API is unreachable. */
export const AUTH_DEFAULTS = {
  passwordPolicy: {
    minLength: 12,
    requireUppercase: true,
    requireLowercase: true,
    requireNumber: true,
    requireSymbol: false,
  } satisfies PasswordPolicy,
} as const
