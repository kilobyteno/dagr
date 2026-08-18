import { ArrowSquareOutIcon, XIcon } from '@phosphor-icons/react'

import { Button } from '@/components/ui/button'
import { useLocale } from '@/lib/i18n'
import { useDesktopUpdate } from '@/lib/updates'
import { cn } from '@/lib/utils'

export function UpdateBanner({ className }: { className?: string }) {
  const { t } = useLocale()
  const { status, showBanner, dismiss, openUpdate } = useDesktopUpdate()

  if (!showBanner || !status?.latestVersion) return null

  return (
    <div
      role="status"
      className={cn(
        'flex shrink-0 items-center gap-3 border-b border-primary/20 bg-primary/10 px-4 py-2 text-sm text-foreground',
        className,
      )}
    >
      <p className="min-w-0 flex-1 text-pretty">
        {t('settings.updates.availableBanner', { version: status.latestVersion })}
      </p>
      <div className="flex shrink-0 items-center gap-2">
        <Button type="button" size="sm" variant="outline" onClick={() => void openUpdate()}>
          <ArrowSquareOutIcon strokeWidth={2} data-icon="inline-start" />
          {t('settings.updates.download')}
        </Button>
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
