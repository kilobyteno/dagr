import { PlugsConnectedIcon, WebhooksLogoIcon } from '@phosphor-icons/react'
import { useEffect, useMemo, useState } from 'react'
import { toast } from 'sonner'

import {
  disableChannelIncomingWebhook,
  enableChannelIncomingWebhook,
  installWorkspaceApp,
  listWorkspaceApps,
  rotateChannelIncomingWebhook,
  uninstallWorkspaceApp,
  type ApiIncomingWebhook,
  type ApiWorkspaceApp,
} from '@/lib/api/apps'
import { formatUserError } from '@/lib/api/client'
import { listChannels, type ApiChannel } from '@/lib/api/workspaces'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useLocale } from '@/lib/i18n'

const INCOMING_WEBHOOKS = 'incoming-webhooks'

async function copyText(value: string) {
  await navigator.clipboard.writeText(value)
}

export function WorkspaceAppsPage({
  workspaceId,
  serverUrl,
  token,
}: {
  workspaceId: string
  serverUrl: string
  token: string
}) {
  const { t } = useLocale()
  const [apps, setApps] = useState<ApiWorkspaceApp[]>([])
  const [channels, setChannels] = useState<ApiChannel[]>([])
  const [loading, setLoading] = useState(true)
  const [busy, setBusy] = useState(false)
  const [selectedChannelId, setSelectedChannelId] = useState('')
  const [revealedUrl, setRevealedUrl] = useState<{
    channelId: string
    url: string
  } | null>(null)

  const incoming = apps.find((app) => app.slug === INCOMING_WEBHOOKS)

  const enabledIds = useMemo(() => {
    return new Set((incoming?.channels ?? []).map((hook) => hook.channelId))
  }, [incoming])

  const availableChannels = channels.filter(
    (channel) => !channel.isDm && !enabledIds.has(channel.id),
  )

  const reload = async (signal?: AbortSignal) => {
    const [appResult, channelResult] = await Promise.all([
      listWorkspaceApps(serverUrl, token, workspaceId, signal),
      listChannels(serverUrl, token, workspaceId, signal),
    ])
    setApps(appResult.apps)
    setChannels(channelResult.channels.filter((channel) => !channel.isDm))
  }

  useEffect(() => {
    const controller = new AbortController()
    setLoading(true)
    void reload(controller.signal)
      .catch((err: unknown) => {
        if (controller.signal.aborted) return
        toast.error(formatUserError(err, t('apps.loadError')))
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false)
      })
    return () => controller.abort()
  }, [serverUrl, token, workspaceId, t])

  const reveal = (hook: ApiIncomingWebhook) => {
    if (!hook.url) return
    setRevealedUrl({ channelId: hook.channelId, url: hook.url })
  }

  const onInstall = async () => {
    setBusy(true)
    try {
      await installWorkspaceApp(serverUrl, token, workspaceId, INCOMING_WEBHOOKS)
      await reload()
      toast.success(t('apps.installed'))
    } catch (err) {
      toast.error(formatUserError(err, t('apps.installError')))
    } finally {
      setBusy(false)
    }
  }

  const onUninstall = async () => {
    setBusy(true)
    try {
      await uninstallWorkspaceApp(serverUrl, token, workspaceId, INCOMING_WEBHOOKS)
      setRevealedUrl(null)
      await reload()
      toast.success(t('apps.removed'))
    } catch (err) {
      toast.error(formatUserError(err, t('apps.removeError')))
    } finally {
      setBusy(false)
    }
  }

  const onAddChannel = async () => {
    if (!selectedChannelId) return
    setBusy(true)
    try {
      const result = await enableChannelIncomingWebhook(
        serverUrl,
        token,
        selectedChannelId,
      )
      reveal(result.webhook)
      setSelectedChannelId('')
      await reload()
      toast.success(t('apps.webhookAdded'))
    } catch (err) {
      toast.error(formatUserError(err, t('apps.webhookAddError')))
    } finally {
      setBusy(false)
    }
  }

  const onRotate = async (channelId: string) => {
    setBusy(true)
    try {
      const result = await rotateChannelIncomingWebhook(
        serverUrl,
        token,
        channelId,
      )
      reveal(result.webhook)
      await reload()
      toast.success(t('apps.webhookRotated'))
    } catch (err) {
      toast.error(formatUserError(err, t('apps.webhookRotateError')))
    } finally {
      setBusy(false)
    }
  }

  const onRemoveHook = async (channelId: string) => {
    setBusy(true)
    try {
      await disableChannelIncomingWebhook(serverUrl, token, channelId)
      if (revealedUrl?.channelId === channelId) setRevealedUrl(null)
      await reload()
      toast.success(t('apps.webhookRemoved'))
    } catch (err) {
      toast.error(formatUserError(err, t('apps.webhookRemoveError')))
    } finally {
      setBusy(false)
    }
  }

  if (loading) {
    return <p className="text-sm text-muted-foreground">{t('apps.loading')}</p>
  }

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h2 className="text-base font-semibold">{t('apps.title')}</h2>
        <p className="mt-1 text-sm text-muted-foreground">{t('apps.description')}</p>
      </div>

      <section className="flex flex-col gap-4 rounded-lg border border-border p-4">
        <div className="flex items-start gap-3">
          <span className="flex size-10 shrink-0 items-center justify-center rounded-md bg-muted text-muted-foreground">
            <WebhooksLogoIcon strokeWidth={2} />
          </span>
          <div className="min-w-0 flex-1">
            <div className="flex flex-wrap items-center gap-2">
              <h3 className="text-sm font-semibold">
                {incoming?.name ?? t('apps.incomingWebhooks')}
              </h3>
              {incoming?.installed ? (
                <Badge variant="secondary">{t('apps.installedBadge')}</Badge>
              ) : null}
            </div>
            <p className="mt-1 text-sm text-muted-foreground">
              {incoming?.description ?? t('apps.incomingWebhooksHint')}
            </p>
          </div>
        </div>

        {incoming?.installed ? (
          <>
            <div className="flex flex-col gap-2 sm:flex-row">
              <Select
                value={selectedChannelId}
                onValueChange={setSelectedChannelId}
                disabled={busy || availableChannels.length === 0}
              >
                <SelectTrigger className="sm:flex-1">
                  <SelectValue placeholder={t('apps.chooseChannel')} />
                </SelectTrigger>
                <SelectContent>
                  {availableChannels.map((channel) => (
                    <SelectItem key={channel.id} value={channel.id}>
                      #{channel.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <Button
                type="button"
                disabled={busy || !selectedChannelId}
                onClick={() => void onAddChannel()}
              >
                {t('apps.addToChannel')}
              </Button>
            </div>

            {(incoming.channels ?? []).length === 0 ? (
              <p className="text-sm text-muted-foreground">{t('apps.noWebhooks')}</p>
            ) : (
              <ul className="flex flex-col gap-3">
                {(incoming.channels ?? []).map((hook) => (
                  <li
                    key={hook.id}
                    className="flex flex-col gap-2 rounded-md border border-border p-3"
                  >
                    <div className="flex items-center justify-between gap-2">
                      <p className="text-sm font-medium">
                        #{hook.channelName || hook.channelId}
                      </p>
                      <span className="text-xs text-muted-foreground">
                        {t('apps.tokenPrefix', { prefix: hook.tokenPrefix })}
                      </span>
                    </div>
                    {revealedUrl?.channelId === hook.channelId ? (
                      <div className="flex flex-col gap-2">
                        <p className="text-xs text-muted-foreground">
                          {t('apps.urlOnce')}
                        </p>
                        <code className="break-all rounded-md bg-muted px-2 py-1 text-xs">
                          {revealedUrl.url}
                        </code>
                        <Button
                          type="button"
                          size="sm"
                          variant="outline"
                          onClick={() => {
                            void copyText(revealedUrl.url)
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
                        onClick={() => void onRotate(hook.channelId)}
                      >
                        {t('apps.rotate')}
                      </Button>
                      <Button
                        type="button"
                        size="sm"
                        variant="ghost"
                        disabled={busy}
                        onClick={() => void onRemoveHook(hook.channelId)}
                      >
                        {t('common.remove')}
                      </Button>
                    </div>
                  </li>
                ))}
              </ul>
            )}

            <Button
              type="button"
              variant="outline"
              disabled={busy}
              onClick={() => void onUninstall()}
            >
              <PlugsConnectedIcon strokeWidth={2} data-icon="inline-start" />
              {t('apps.removeFromWorkspace')}
            </Button>
          </>
        ) : (
          <Button type="button" disabled={busy} onClick={() => void onInstall()}>
            {t('apps.install')}
          </Button>
        )}
      </section>
    </div>
  )
}
