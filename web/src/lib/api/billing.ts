import { apiFetch } from '@/lib/api/client'

export type ApiBillingEntitlements = {
  plan: string
  unlimitedHistory: boolean
  messageHistoryDays?: number
  maxApps?: number
  crossWorkspaceDms: string
}

export type ApiWorkspaceBilling = {
  enabled: boolean
  plan: string
  status?: string
  interval?: string
  billableSeats?: number
  currentPeriodEnd?: string | null
  cancelAtPeriodEnd?: boolean
  earlyAccessEndsAt?: string | null
  monthlyAmountCents?: number
  yearlyAmountCents?: number
  nextAmountCents?: number
  currency?: string
  canManage: boolean
  entitlements: ApiBillingEntitlements
}

export type GetWorkspaceBillingResponse = {
  billing: ApiWorkspaceBilling
}

export type BillingCheckoutResponse = {
  checkoutUrl: string
  paymentId: string
}

export function formatPlanAmount(cents: number, currency = 'EUR') {
  const value = (cents / 100).toLocaleString('en-GB', {
    style: 'currency',
    currency,
    minimumFractionDigits: 2,
  })
  return value
}

export function getWorkspaceBilling(
  serverUrl: string,
  token: string,
  workspaceId: string,
  signal?: AbortSignal,
) {
  return apiFetch<GetWorkspaceBillingResponse>(
    serverUrl,
    `/api/v1/workspaces/${workspaceId}/billing`,
    { method: 'GET', token, signal },
  )
}

export function startBillingCheckout(
  serverUrl: string,
  token: string,
  workspaceId: string,
  interval: 'monthly' | 'yearly',
) {
  return apiFetch<BillingCheckoutResponse>(
    serverUrl,
    `/api/v1/workspaces/${workspaceId}/billing/checkout`,
    { method: 'POST', token, body: { interval } },
  )
}

export function cancelWorkspaceBilling(
  serverUrl: string,
  token: string,
  workspaceId: string,
) {
  return apiFetch<void>(
    serverUrl,
    `/api/v1/workspaces/${workspaceId}/billing/cancel`,
    { method: 'POST', token },
  )
}

export function resumeWorkspaceBilling(
  serverUrl: string,
  token: string,
  workspaceId: string,
) {
  return apiFetch<void>(
    serverUrl,
    `/api/v1/workspaces/${workspaceId}/billing/resume`,
    { method: 'POST', token },
  )
}
