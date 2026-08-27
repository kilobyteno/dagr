import { ArrowSquareOutIcon, ArrowsClockwiseIcon, XIcon } from '@phosphor-icons/react'

import { Button } from '@/components/ui/button'
import { useLocale } from '@/lib/i18n'
import {
  isUpdateDownloading,
  isUpdateReady,
  updateDownloadPercent,
  useDesktopUpdate,
} from '@/lib/updates'
import { cn } from '@/lib/utils'

export function UpdateBanner({ className }: { className?: string }) {
  const { t } = useLocale()
  const { status, showBanner, dismiss, openUpdate, install } = useDesktopUpdate()
  const version = status?.latestVersion
  const downloading = isUpdateDownloading(status)
  const ready = isUpdateReady(status)
  const percent = updateDownloadPercent(status)

  if (!showBanner || !version) return null

  const message = ready
    ? t('settings.updates.readyBanner', { version })
    : downloading
      ? percent === null
        ? t('settings.updates.downloadingBanner', { version })
        : t('settings.updates.downloadingBannerPercent', {
            version,
            percent,
          })
      : t('settings.updates.availableBanner', { version })

  return (
    <div
      role="status"
      className={cn(
        'flex shrink-0 items-center gap-3 border-b border-primary/20 bg-primary/10 px-4 py-2 text-sm text-foreground',
        className,
      )}
    >
      <p className="min-w-0 flex-1 text-pretty">{message}</p>
      <div className="flex shrink-0 items-center gap-2">
        {ready ? (
          <Button type="button" size="sm" onClick={() => void install()}>
            <ArrowsClockwiseIcon strokeWidth={2} data-icon="inline-start" />
            {t('settings.updates.restart')}
          </Button>
        ) : downloading ? null : (
          <Button type="button" size="sm" variant="outline" onClick={() => void openUpdate()}>
            <ArrowSquareOutIcon strokeWidth={2} data-icon="inline-start" />
            {t('settings.updates.download')}
          </Button>
        )}
        <Button
          type="button"
          size="icon-sm"
          variant="ghost"
          aria-label={t('settings.updates.dismiss')}
          onClick={dismiss}
        >
          <XIcon strokeWidth={2} data-icon />
        </Button>
      </div>
    </div>
  )
}
