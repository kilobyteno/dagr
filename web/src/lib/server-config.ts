import { ApiError } from '@/lib/api/client'
import { AUTH_DEFAULTS } from '@/lib/auth-defaults'
import type { PasswordPolicy } from '@/lib/password-policy'

export type ServerPublicPlan = {
  id: string
  name: string
  monthlyPriceCents: number
  yearlyPriceCents: number
  currency: string
  messageHistoryDays?: number
  unlimitedHistory?: boolean
}

export type ServerPublicConfig = {
  passwordPolicy: PasswordPolicy
  serverId?: string
  publicUrl?: string
  signingPublicKey?: string
  deploymentMode?: 'cloud' | 'selfhosted'
  billingEnabled?: boolean
  plans?: ServerPublicPlan[]
}

const FALLBACK: ServerPublicConfig = {
  passwordPolicy: { ...AUTH_DEFAULTS.passwordPolicy },
}

function parsePasswordPolicy(raw: unknown): PasswordPolicy | null {
  if (!raw || typeof raw !== 'object') return null
  const data = raw as Partial<PasswordPolicy>
  const minLength = Number(data.minLength)
  if (!Number.isFinite(minLength) || minLength < 1) return null

  return {
    minLength: Math.floor(minLength),
    requireUppercase: Boolean(data.requireUppercase),
    requireLowercase: Boolean(data.requireLowercase),
    requireNumber: Boolean(data.requireNumber),
    requireSymbol: Boolean(data.requireSymbol),
  }
}

export async function fetchServerPublicConfig(
  serverUrl: string,
  signal?: AbortSignal,
): Promise<ServerPublicConfig> {
  const base = serverUrl.trim().replace(/\/$/, '')
  if (!base) return FALLBACK

  let response: Response
  try {
    response = await fetch(`${base}/api/v1/public/config`, {
      method: 'GET',
      headers: { Accept: 'application/json' },
      signal,
    })
  } catch (error) {
    if (signal?.aborted) throw error
    throw new ApiError(0, 'network_error', 'Could not reach the Dagr server')
  }

  if (
    response.status === 502 ||
    response.status === 503 ||
    response.status === 504
  ) {
    throw new ApiError(
      response.status,
      'server_unavailable',
      'Could not reach the Dagr server',
    )
  }
  if (!response.ok) return FALLBACK
  const data = (await response.json()) as {
    passwordPolicy?: unknown
    serverId?: unknown
    publicUrl?: unknown
    signingPublicKey?: unknown
    deploymentMode?: unknown
    billingEnabled?: unknown
    plans?: { plans?: ServerPublicPlan[] }
  }
  const passwordPolicy = parsePasswordPolicy(data.passwordPolicy)
  if (!passwordPolicy) return FALLBACK
  return {
    passwordPolicy,
    serverId: typeof data.serverId === 'string' ? data.serverId : undefined,
    publicUrl: typeof data.publicUrl === 'string' ? data.publicUrl : undefined,
    signingPublicKey:
      typeof data.signingPublicKey === 'string'
        ? data.signingPublicKey
        : undefined,
    deploymentMode:
      data.deploymentMode === 'cloud' || data.deploymentMode === 'selfhosted'
        ? data.deploymentMode
        : undefined,
    billingEnabled: Boolean(data.billingEnabled),
    plans: Array.isArray(data.plans?.plans) ? data.plans.plans : undefined,
  }
}
