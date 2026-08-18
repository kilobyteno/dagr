import { apiFetch, apiUpload } from '@/lib/api/client'
import type { AppLocale } from '@/lib/i18n/locales'

export type { AppLocale }

export type NotificationLevel = 'all' | 'mentions' | 'nothing'

export type PresenceState = 'online' | 'away' | 'offline'

export type ApiUser = {
  id: string
  email: string
  displayName: string
  notificationLevel: NotificationLevel
  locale?: AppLocale
  emailVerified?: boolean
  statusEmoji?: string
  statusText?: string
  statusExpiresAt?: string | null
  hasAvatar?: boolean
  avatarUpdatedAt?: string | null
}

export type AuthResponse = {
  token: string
  expiresAt: string
  user: ApiUser
}

export type MeResponse = {
  user: ApiUser
}

export function signup(
  serverUrl: string,
  input: { email: string; password: string; displayName: string },
) {
  return apiFetch<AuthResponse>(serverUrl, '/api/v1/auth/signup', {
    method: 'POST',
    body: input,
  })
}

export function login(
  serverUrl: string,
  input: { email: string; password: string },
) {
  return apiFetch<AuthResponse>(serverUrl, '/api/v1/auth/login', {
    method: 'POST',
    body: input,
  })
}

export function logout(serverUrl: string, token: string) {
  return apiFetch<void>(serverUrl, '/api/v1/auth/logout', {
    method: 'POST',
    token,
  })
}

export function verifyEmail(serverUrl: string, token: string) {
  return apiFetch<MeResponse>(serverUrl, '/api/v1/auth/verify-email', {
    method: 'POST',
    body: { token },
  })
}

export function resendVerificationEmail(serverUrl: string, token: string) {
  return apiFetch<void>(serverUrl, '/api/v1/me/email/resend-verification', {
    method: 'POST',
    token,
  })
}

export function me(serverUrl: string, token: string, signal?: AbortSignal) {
  return apiFetch<MeResponse>(serverUrl, '/api/v1/me', {
    method: 'GET',
    token,
    signal,
  })
}

export function updateProfile(
  serverUrl: string,
  token: string,
  input: {
    displayName: string
    notificationLevel: NotificationLevel
    locale?: AppLocale
  },
) {
  return apiFetch<MeResponse>(serverUrl, '/api/v1/me', {
    method: 'PATCH',
    token,
    body: input,
  })
}

export function updateStatus(
  serverUrl: string,
  token: string,
  input: { emoji: string; text: string; expiresAt?: string | null },
) {
  return apiFetch<MeResponse>(serverUrl, '/api/v1/me/status', {
    method: 'PUT',
    token,
    body: input,
  })
}

export function updatePresence(
  serverUrl: string,
  token: string,
  state: 'active' | 'away',
) {
  return apiFetch<{ presence: PresenceState }>(
    serverUrl,
    '/api/v1/me/presence',
    {
      method: 'POST',
      token,
      body: { state },
    },
  )
}

export function uploadAvatar(serverUrl: string, token: string, file: File) {
  const formData = new FormData()
  formData.append('avatar', file)
  return apiUpload<MeResponse>(serverUrl, '/api/v1/me/avatar', {
    method: 'PUT',
    token,
    formData,
  })
}

export function removeAvatar(serverUrl: string, token: string) {
  return apiFetch<MeResponse>(serverUrl, '/api/v1/me/avatar', {
    method: 'DELETE',
    token,
  })
}

export async function fetchUserAvatarObjectUrl(
  serverUrl: string,
  token: string,
  userId: string,
  signal?: AbortSignal,
): Promise<string | null> {
  const base = serverUrl.trim().replace(/\/$/, '')
  let response: Response
  try {
    response = await fetch(`${base}/api/v1/users/${userId}/avatar`, {
      method: 'GET',
      headers: {
        Authorization: `Bearer ${token}`,
        Accept: 'image/*',
      },
      signal,
    })
  } catch {
    return null
  }
  if (!response.ok) return null
  const blob = await response.blob()
  return URL.createObjectURL(blob)
}

export function userInitials(name: string) {
  return (
    name
      .split(/\s+/)
      .map((part) => part[0])
      .join('')
      .slice(0, 2)
      .toUpperCase() || '?'
  )
}
