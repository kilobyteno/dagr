import { CreditCardIcon } from '@phosphor-icons/react'
import { useEffect, useRef, useState } from 'react'
import { toast } from 'sonner'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { formatUserError } from '@/lib/api/client'
import {
  cancelWorkspaceBilling,
  formatPlanAmount,
  getWorkspaceBilling,
  resumeWorkspaceBilling,
  startBillingCheckout,
  type ApiWorkspaceBilling,
} from '@/lib/api/billing'
import { useLocale } from '@/lib/i18n'

function planLabel(plan: string, t: ReturnType<typeof useLocale>['t']) {
  return plan === 'pro' ? t('workspace.billing.pro') : t('workspace.billing.free')
}

function billingChanged(
  previous: ApiWorkspaceBilling | null,
  next: ApiWorkspaceBilling,
) {
  if (!previous) return true
  return (
    previous.plan !== next.plan ||
    previous.status !== next.status ||
    previous.cancelAtPeriodEnd !== next.cancelAtPeriodEnd ||
    previous.interval !== next.interval ||
    previous.currentPeriodEnd !== next.currentPeriodEnd
  )
}

export function WorkspaceBillingSection({
  workspaceId,
  serverUrl,
  token,
  canManage,
  refreshToken = 0,
}: {
  workspaceId: string
  serverUrl: string
  token: string
  canManage: boolean
  refreshToken?: number
}) {
  const { t, locale, formatDate } = useLocale()
  const [billing, setBilling] = useState<ApiWorkspaceBilling | null>(null)
  const [loading, setLoading] = useState(true)
  const [busy, setBusy] = useState<'monthly' | 'yearly' | 'cancel' | 'resume' | null>(
    null,
  )
  const billingRef = useRef<ApiWorkspaceBilling | null>(null)
  const pollGeneration = useRef(0)
  billingRef.current = billing

  const reload = async (signal?: AbortSignal) => {
    const result = await getWorkspaceBilling(
      serverUrl,
      token,
      workspaceId,
      signal,
    )
    setBilling(result.billing)
    return result.billing
  }

  const stopPoll = () => {
    pollGeneration.current += 1
  }

  const startPoll = (previous: ApiWorkspaceBilling | null) => {
    const generation = ++pollGeneration.current
    const startedAt = Date.now()
    const tick = async () => {
      if (generation !== pollGeneration.current) return
      try {
        const next = await reload()
        if (generation !== pollGeneration.current) return
        if (billingChanged(previous, next)) {
          if (
            next.entitlements.plan === 'pro' &&
            previous?.entitlements.plan !== 'pro'
          ) {
            toast.success(t('workspace.billing.proActive'))
          }
          return
        }
      } catch {
        // Webhook may still be in flight.
      }
      if (Date.now() - startedAt >= 90_000) return
      window.setTimeout(() => {
        void tick()
      }, 2500)
    }
    void tick()
  }

  useEffect(() => {
    const controller = new AbortController()
    setLoading(true)
    void reload(controller.signal)
      .catch((err) => {
        if (controller.signal.aborted) return
        const message =
          formatUserError(err, t('workspace.billing.loadError'))
        toast.error(message)
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false)
      })
    return () => {
      controller.abort()
      stopPoll()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [serverUrl, token, workspaceId])

  useEffect(() => {
    if (!refreshToken) return
    startPoll(billingRef.current)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [refreshToken])

  useEffect(() => {
    const onVisible = () => {
      if (document.visibilityState !== 'visible') return
      void reload().catch(() => undefined)
    }
    window.addEventListener('focus', onVisible)
    document.addEventListener('visibilitychange', onVisible)
    return () => {
      window.removeEventListener('focus', onVisible)
      document.removeEventListener('visibilitychange', onVisible)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [serverUrl, token, workspaceId])

  if (loading) {
    return null
  }

  if (!billing?.enabled) {
    return (
      <section className="flex flex-col gap-4">
        <div className="flex flex-col gap-1">
          <h2 className="flex items-center gap-2 text-sm font-semibold">
            <CreditCardIcon strokeWidth={2} className="size-4" />
            {t('workspace.billing.title')}
          </h2>
          <p className="text-sm text-muted-foreground">
            {t('workspace.billing.unavailable')}
          </p>
        </div>
      </section>
    )
  }

  const currency = billing.currency || 'EUR'
  const seats = billing.billableSeats || 1
  const isPro = billing.entitlements.plan === 'pro'
  const historyCopy = billing.entitlements.unlimitedHistory
    ? t('workspace.billing.unlimitedHistory')
    : t('workspace.billing.historyDays', {
        count: billing.entitlements.messageHistoryDays ?? 90,
      })

  return (
    <section className="flex flex-col gap-4">
      <div className="flex flex-col gap-1">
        <h2 className="flex items-center gap-2 text-sm font-semibold">
          <CreditCardIcon strokeWidth={2} className="size-4" />
          {t('workspace.billing.title')}
        </h2>
        <p className="text-sm text-muted-foreground">
          {t('workspace.billing.description')}
        </p>
      </div>

      <div className="flex flex-col gap-3 rounded-lg border p-4">
        <div className="flex flex-wrap items-center gap-2">
          <span className="text-sm font-medium">{planLabel(billing.plan, t)}</span>
          <Badge variant={isPro ? 'secondary' : 'outline'}>
            {isPro ? t('workspace.billing.cloudPro') : t('workspace.billing.freeForever')}
          </Badge>
          {billing.cancelAtPeriodEnd ? (
            <Badge variant="outline">{t('workspace.billing.cancelsAtEnd')}</Badge>
          ) : null}
        </div>
        <p className="text-sm text-muted-foreground">{historyCopy}</p>
        <p className="text-xs text-muted-foreground">
          {t('workspace.billing.seat', { count: seats })}
          {billing.interval
            ? t('workspace.billing.billed', { interval: billing.interval })
            : ''}
          {billing.currentPeriodEnd
            ? t('workspace.billing.periodEnds', {
                date: formatDate(new Date(billing.currentPeriodEnd)),
              })
            : ''}
        </p>
        {billing.earlyAccessEndsAt ? (
          <p className="text-xs text-muted-foreground">
            {t('workspace.billing.earlyAccessEnds', {
              date: formatDate(new Date(billing.earlyAccessEndsAt)),
            })}
          </p>
        ) : null}
      </div>

      <div className="grid gap-3 sm:grid-cols-2">
        <div className="flex flex-col gap-2 rounded-lg border p-4">
          <h3 className="text-sm font-medium">{t('workspace.billing.free')}</h3>
          <p className="text-lg font-semibold">€0</p>
          <ul className="list-disc space-y-1 pl-4 text-xs text-muted-foreground">
            <li>{t('workspace.billing.freeHistory')}</li>
            <li className="text-muted-foreground/70">
              {t('workspace.billing.freeApps')}
            </li>
            <li className="text-muted-foreground/70">
              {t('workspace.billing.freeDms')}
            </li>
          </ul>
        </div>
        <div className="flex flex-col gap-2 rounded-lg border p-4">
          <h3 className="text-sm font-medium">{t('workspace.billing.pro')}</h3>
          <p className="text-lg font-semibold">
            {formatPlanAmount(billing.monthlyAmountCents || 700, currency, locale)}
            <span className="text-xs font-normal text-muted-foreground">
              {t('workspace.billing.perSeatMonth')}
            </span>
          </p>
          <p className="text-xs text-muted-foreground">
            {t('workspace.billing.yearlyHint', {
              amount: formatPlanAmount(
                billing.yearlyAmountCents || 7560,
                currency,
                locale,
              ),
            })}
          </p>
          <ul className="list-disc space-y-1 pl-4 text-xs text-muted-foreground">
            <li>{t('workspace.billing.unlimitedHistory')}</li>
            <li className="text-muted-foreground/70">
              {t('workspace.billing.proApps')}
            </li>
            <li className="text-muted-foreground/70">
              {t('workspace.billing.proDms')}
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
                      toast.success(t('workspace.billing.checkoutOpened'))
                      startPoll(billingRef.current)
                    })
                    .catch((err) => {
                      toast.error(
                        formatUserError(err, t('workspace.billing.checkoutError')),
                      )
                    })
                    .finally(() => setBusy(null))
                }}
              >
                {busy === 'monthly'
                  ? t('workspace.billing.starting')
                  : t('workspace.billing.upgradeMonthly')}
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
                      toast.success(t('workspace.billing.checkoutOpened'))
                      startPoll(billingRef.current)
                    })
                    .catch((err) => {
                      toast.error(
                        formatUserError(err, t('workspace.billing.checkoutError')),
                      )
                    })
                    .finally(() => setBusy(null))
                }}
              >
                {busy === 'yearly'
                  ? t('workspace.billing.starting')
                  : t('workspace.billing.upgradeYearly')}
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
                  .then(() => toast.success(t('workspace.billing.cancelScheduled')))
                  .catch((err) => {
                    toast.error(
                      formatUserError(err, t('workspace.billing.cancelError')),
                    )
                  })
                  .finally(() => setBusy(null))
              }}
            >
              {busy === 'cancel'
                ? t('workspace.billing.cancelling')
                : t('workspace.billing.cancel')}
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
                  .then(() => toast.success(t('workspace.billing.resumed')))
                  .catch((err) => {
                    toast.error(
                      formatUserError(err, t('workspace.billing.resumeError')),
                    )
                  })
                  .finally(() => setBusy(null))
              }}
            >
              {busy === 'resume'
                ? t('workspace.billing.resuming')
                : t('workspace.billing.resume')}
            </Button>
          ) : null}
        </div>
      ) : (
        <p className="text-xs text-muted-foreground">
          {t('workspace.billing.permission')}
        </p>
      )}
    </section>
  )
}
