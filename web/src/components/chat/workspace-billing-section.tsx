import { CreditCardIcon } from '@phosphor-icons/react'
import { useEffect, useState } from 'react'
import { toast } from 'sonner'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import { ApiError } from '@/lib/api/client'
import {
  cancelWorkspaceBilling,
  formatPlanAmount,
  getWorkspaceBilling,
  resumeWorkspaceBilling,
  startBillingCheckout,
  type ApiWorkspaceBilling,
} from '@/lib/api/billing'

function planLabel(plan: string) {
  return plan === 'pro' ? 'Pro' : 'Free'
}

export function WorkspaceBillingSection({
  workspaceId,
  serverUrl,
  token,
  canManage,
}: {
  workspaceId: string
  serverUrl: string
  token: string
  canManage: boolean
}) {
  const [billing, setBilling] = useState<ApiWorkspaceBilling | null>(null)
  const [loading, setLoading] = useState(true)
  const [busy, setBusy] = useState<'monthly' | 'yearly' | 'cancel' | 'resume' | null>(
    null,
  )

  const reload = async (signal?: AbortSignal) => {
    const result = await getWorkspaceBilling(
      serverUrl,
      token,
      workspaceId,
      signal,
    )
    setBilling(result.billing)
  }

  useEffect(() => {
    const controller = new AbortController()
    setLoading(true)
    void reload(controller.signal)
      .catch((err) => {
        if (controller.signal.aborted) return
        const message =
          err instanceof ApiError ? err.message : 'Could not load billing'
        toast.error(message)
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false)
      })
    return () => controller.abort()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [serverUrl, token, workspaceId])

  if (loading || !billing?.enabled) {
    return null
  }

  const currency = billing.currency || 'EUR'
  const seats = billing.billableSeats || 1
  const isPro = billing.entitlements.plan === 'pro'
  const historyCopy = billing.entitlements.unlimitedHistory
    ? 'Unlimited message history.'
    : `Message history is kept for ${billing.entitlements.messageHistoryDays ?? 90} days. Upgrading to Pro stops the purge. Messages already removed do not come back.`

  return (
    <>
    <Separator />
    <section className="flex flex-col gap-4">
      <div className="flex flex-col gap-1">
        <h2 className="flex items-center gap-2 text-sm font-semibold">
          <CreditCardIcon strokeWidth={2} className="size-4" />
          Plan and billing
        </h2>
        <p className="text-sm text-muted-foreground">
          Cloud plans are per workspace. Only home members count as paid seats.
          External people stay free.
        </p>
      </div>

      <div className="flex flex-col gap-3 rounded-lg border p-4">
        <div className="flex flex-wrap items-center gap-2">
          <span className="text-sm font-medium">{planLabel(billing.plan)}</span>
          <Badge variant={isPro ? 'secondary' : 'outline'}>
            {isPro ? 'Cloud Pro' : 'Free forever'}
          </Badge>
          {billing.cancelAtPeriodEnd ? (
            <Badge variant="outline">Cancels at period end</Badge>
          ) : null}
        </div>
        <p className="text-sm text-muted-foreground">{historyCopy}</p>
        <p className="text-xs text-muted-foreground">
          {seats} billable {seats === 1 ? 'seat' : 'seats'}
          {billing.interval ? ` · billed ${billing.interval}` : ''}
          {billing.currentPeriodEnd
            ? ` · current period ends ${new Date(billing.currentPeriodEnd).toLocaleDateString('en-GB')}`
            : ''}
        </p>
        {billing.earlyAccessEndsAt ? (
          <p className="text-xs text-muted-foreground">
            Early access pricing (50% off the first 3 months) ends{' '}
            {new Date(billing.earlyAccessEndsAt).toLocaleDateString('en-GB')}.
          </p>
        ) : null}
      </div>

      <div className="grid gap-3 sm:grid-cols-2">
        <div className="flex flex-col gap-2 rounded-lg border p-4">
          <h3 className="text-sm font-medium">Free</h3>
          <p className="text-lg font-semibold">€0</p>
          <ul className="list-disc space-y-1 pl-4 text-xs text-muted-foreground">
            <li>90-day message history</li>
            <li className="text-muted-foreground/70">
              10 apps and integrations (coming soon)
            </li>
            <li className="text-muted-foreground/70">
              1:1 DMs across workspaces (coming soon)
            </li>
          </ul>
        </div>
        <div className="flex flex-col gap-2 rounded-lg border p-4">
          <h3 className="text-sm font-medium">Pro</h3>
          <p className="text-lg font-semibold">
            {formatPlanAmount(billing.monthlyAmountCents || 700, currency)}
            <span className="text-xs font-normal text-muted-foreground">
              {' '}
              / seat / month
            </span>
          </p>
          <p className="text-xs text-muted-foreground">
            Yearly is 10% off (
            {formatPlanAmount(billing.yearlyAmountCents || 7560, currency)} per
            seat / year). Early access is 50% off the first 3 months.
          </p>
          <ul className="list-disc space-y-1 pl-4 text-xs text-muted-foreground">
            <li>Unlimited message history</li>
            <li className="text-muted-foreground/70">
              Unlimited apps and integrations (coming soon)
            </li>
            <li className="text-muted-foreground/70">
              Group DMs across workspaces (coming soon)
            </li>
          </ul>
        </div>
      </div>

      {canManage && billing.canManage ? (
        <div className="flex flex-wrap gap-2">
          {!isPro || billing.cancelAtPeriodEnd ? (
            <>
              <Button
                type="button"
                size="sm"
                disabled={busy !== null}
                onClick={() => {
                  setBusy('monthly')
                  void startBillingCheckout(
                    serverUrl,
                    token,
                    workspaceId,
                    'monthly',
                  )
                    .then((result) => {
                      window.open(result.checkoutUrl, '_blank', 'noopener')
                      toast.success('Opened Mollie checkout')
                    })
                    .catch((err) => {
                      toast.error(
                        err instanceof ApiError
                          ? err.message
                          : 'Could not start checkout',
                      )
                    })
                    .finally(() => setBusy(null))
                }}
              >
                {busy === 'monthly' ? 'Starting…' : 'Upgrade monthly'}
              </Button>
              <Button
                type="button"
                size="sm"
                variant="outline"
                disabled={busy !== null}
                onClick={() => {
                  setBusy('yearly')
                  void startBillingCheckout(
                    serverUrl,
                    token,
                    workspaceId,
                    'yearly',
                  )
                    .then((result) => {
                      window.open(result.checkoutUrl, '_blank', 'noopener')
                      toast.success('Opened Mollie checkout')
                    })
                    .catch((err) => {
                      toast.error(
                        err instanceof ApiError
                          ? err.message
                          : 'Could not start checkout',
                      )
                    })
                    .finally(() => setBusy(null))
                }}
              >
                {busy === 'yearly' ? 'Starting…' : 'Upgrade yearly'}
              </Button>
            </>
          ) : null}
          {isPro && !billing.cancelAtPeriodEnd ? (
            <Button
              type="button"
              size="sm"
              variant="outline"
              disabled={busy !== null}
              onClick={() => {
                setBusy('cancel')
                void cancelWorkspaceBilling(serverUrl, token, workspaceId)
                  .then(() => reload())
                  .then(() => toast.success('Pro will end at the period close'))
                  .catch((err) => {
                    toast.error(
                      err instanceof ApiError
                        ? err.message
                        : 'Could not cancel Pro',
                    )
                  })
                  .finally(() => setBusy(null))
              }}
            >
              {busy === 'cancel' ? 'Cancelling…' : 'Cancel Pro'}
            </Button>
          ) : null}
          {isPro && billing.cancelAtPeriodEnd ? (
            <Button
              type="button"
              size="sm"
              variant="outline"
              disabled={busy !== null}
              onClick={() => {
                setBusy('resume')
                void resumeWorkspaceBilling(serverUrl, token, workspaceId)
                  .then(() => reload())
                  .then(() => toast.success('Pro will continue'))
                  .catch((err) => {
                    toast.error(
                      err instanceof ApiError
                        ? err.message
                        : 'Could not resume Pro',
                    )
                  })
                  .finally(() => setBusy(null))
              }}
            >
              {busy === 'resume' ? 'Resuming…' : 'Resume Pro'}
            </Button>
          ) : null}
        </div>
      ) : (
        <p className="text-xs text-muted-foreground">
          Only workspace owners and admins can change the plan.
        </p>
      )}
    </section>
    </>
  )
}
