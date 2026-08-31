import { useEffect, useState } from 'react'
import { toast } from 'sonner'

import {
  disableChannelIncomingWebhook,
  enableChannelIncomingWebhook,
  getChannelIncomingWebhook,
  rotateChannelIncomingWebhook,
  type ApiIncomingWebhook,
} from '@/lib/api/apps'
import { ApiError, formatUserError } from '@/lib/api/client'
import { Button } from '@/components/ui/button'
import { useLocale } from '@/lib/i18n'

async function copyText(value: string) {
  await navigator.clipboard.writeText(value)
}

export function ChannelIntegrations({
  channelId,
  serverUrl,
  token,
}: {
  channelId: string
  serverUrl: string
  token: string
}) {
  const { t } = useLocale()
  const [hook, setHook] = useState<ApiIncomingWebhook | null>(null)
  const [url, setUrl] = useState('')
  const [loading, setLoading] = useState(true)
  const [busy, setBusy] = useState(false)

  const reload = async (signal?: AbortSignal) => {
    try {
      const result = await getChannelIncomingWebhook(
        serverUrl,
        token,
        channelId,
        signal,
      )
      setHook(result.webhook)
    } catch (err) {
      if (signal?.aborted) return
      if (err instanceof ApiError && err.status === 404) {
        setHook(null)
        return
      }
      throw err
    }
  }

  useEffect(() => {
    const controller = new AbortController()
    setLoading(true)
    setUrl('')
    void reload(controller.signal)
      .catch((err: unknown) => {
        if (controller.signal.aborted) return
        toast.error(formatUserError(err, t('apps.loadError')))
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false)
      })
    return () => controller.abort()
  }, [channelId, serverUrl, token, t])

  const onAdd = async () => {
    setBusy(true)
    try {
      const result = await enableChannelIncomingWebhook(
        serverUrl,
        token,
        channelId,
      )
      setHook(result.webhook)
      setUrl(result.webhook.url ?? '')
      toast.success(t('apps.webhookAdded'))
    } catch (err) {
      toast.error(formatUserError(err, t('apps.webhookAddError')))
    } finally {
      setBusy(false)
    }
  }

  const onRotate = async () => {
    setBusy(true)
    try {
      const result = await rotateChannelIncomingWebhook(
        serverUrl,
        token,
        channelId,
      )
      setHook(result.webhook)
      setUrl(result.webhook.url ?? '')
      toast.success(t('apps.webhookRotated'))
    } catch (err) {
      toast.error(formatUserError(err, t('apps.webhookRotateError')))
    } finally {
      setBusy(false)
    }
  }

  const onRemove = async () => {
    setBusy(true)
    try {
      await disableChannelIncomingWebhook(serverUrl, token, channelId)
      setHook(null)
      setUrl('')
      toast.success(t('apps.webhookRemoved'))
    } catch (err) {
      toast.error(formatUserError(err, t('apps.webhookRemoveError')))
    } finally {
      setBusy(false)
    }
  }

  if (loading) {
    return <p className="text-xs text-muted-foreground">{t('apps.loading')}</p>
  }

  return (
    <section className="flex flex-col gap-3">
      <div>
        <h3 className="text-sm font-medium">{t('apps.incomingWebhooks')}</h3>
        <p className="text-xs text-muted-foreground">
          {t('apps.channelHint')}
        </p>
      </div>
      {hook ? (
        <>
          <p className="text-xs text-muted-foreground">
            {t('apps.tokenPrefix', { prefix: hook.tokenPrefix })}
          </p>
          {url ? (
            <div className="flex flex-col gap-2">
              <p className="text-xs text-muted-foreground">{t('apps.urlOnce')}</p>
              <code className="break-all rounded-md bg-muted px-2 py-1 text-xs">
                {url}
              </code>
              <Button
                type="button"
                size="sm"
                variant="outline"
                onClick={() => {
                  void copyText(url)
                    .then(() => toast.success(t('apps.copied')))
                    .catch(() => toast.error(t('apps.copyError')))
                }}
              >
                {t('common.copy')}
              </Button>
            </div>
          ) : null}
          <div className="flex flex-wrap gap-2">
            <Button
              type="button"
              size="sm"
              variant="outline"
              disabled={busy}
              onClick={() => void onRotate()}
            >
              {t('apps.rotate')}
            </Button>
            <Button
              type="button"
              size="sm"
              variant="ghost"
              disabled={busy}
              onClick={() => void onRemove()}
            >
              {t('common.remove')}
            </Button>
          </div>
        </>
      ) : (
        <Button type="button" size="sm" disabled={busy} onClick={() => void onAdd()}>
          {t('apps.addToThisChannel')}
        </Button>
      )}
    </section>
  )
}
