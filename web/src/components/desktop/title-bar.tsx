import type { ReactNode } from 'react'

import { Button } from '@/components/ui/button'
import { isElectronMac } from '@/lib/desktop'
import { DOCS_BASE_URL } from '@/lib/docs'
import { cn } from '@/lib/utils'
import { ArrowLeftIcon, ArrowRightIcon, QuestionIcon } from '@phosphor-icons/react'

type TitleBarProps = {
  title: string
  subtitle?: string
  initials?: string
  brandMark?: ReactNode
  canGoBack?: boolean
  canGoForward?: boolean
  onBack?: () => void
  onForward?: () => void
  showNavigation?: boolean
  className?: string
}

export function TitleBar({
  title,
  subtitle,
  initials,
  brandMark,
  canGoBack = false,
  canGoForward = false,
  onBack,
  onForward,
  showNavigation = true,
  className,
}: TitleBarProps) {
  const macDesktop = isElectronMac()
  const label = subtitle ? `${title} · ${subtitle}` : title

  return (
    <header
      className={cn(
        'app-drag relative z-50 flex h-9 w-full shrink-0 items-center border-b border-border bg-muted text-foreground dark:bg-sidebar dark:text-sidebar-foreground',
        className,
      )}
      aria-label={label}
    >
      <div
        className={cn(
          'app-no-drag relative z-10 flex h-full items-center gap-0.5 pl-2',
          macDesktop && 'pl-[78px]',
        )}
      >
        {showNavigation ? (
          <>
            <Button
              type="button"
              variant="ghost"
              size="icon-xs"
              className="text-muted-foreground hover:text-foreground"
              aria-label="Go back"
              disabled={!canGoBack}
              onClick={onBack}
            >
              <ArrowLeftIcon strokeWidth={2} />
            </Button>
            <Button
              type="button"
              variant="ghost"
              size="icon-xs"
              className="text-muted-foreground hover:text-foreground"
              aria-label="Go forward"
              disabled={!canGoForward}
              onClick={onForward}
            >
              <ArrowRightIcon strokeWidth={2} />
            </Button>
          </>
        ) : null}
      </div>

      <div className="pointer-events-none absolute inset-0 flex items-center justify-center">
        <div className="flex max-w-[min(520px,60%)] items-center gap-2 px-2">
          {brandMark ? (
            <span className="flex size-4 shrink-0 items-center justify-center overflow-hidden rounded-md">
              {brandMark}
            </span>
          ) : initials ? (
            <span className="flex size-4 shrink-0 items-center justify-center rounded-md bg-primary text-[9px] font-semibold text-primary-foreground">
              {initials}
            </span>
          ) : null}
          <span className="flex min-w-0 items-center gap-1.5 text-xs tracking-tight">
            <span className="truncate font-semibold">{title}</span>
            {subtitle ? (
              <>
                <span className="shrink-0 text-muted-foreground" aria-hidden>
                  ·
                </span>
                <span className="truncate font-medium text-muted-foreground">
                  {subtitle}
                </span>
              </>
            ) : null}
          </span>
        </div>
      </div>

      <div className="app-no-drag relative z-10 ml-auto flex h-full items-center gap-0.5 pr-2">
        <Button
          type="button"
          variant="ghost"
          size="icon-xs"
          className="text-muted-foreground hover:text-foreground"
          aria-label="Help"
          onClick={() => {
            window.open(DOCS_BASE_URL, '_blank', 'noopener,noreferrer')
          }}
        >
          <QuestionIcon strokeWidth={2} />
        </Button>
      </div>
    </header>
  )
}
