import { DagrMark } from '@/components/auth/dagr-mark'
import { cn } from '@/lib/utils'

export function AppLoadingScreen({
  label = 'Loading…',
  className,
}: {
  label?: string
  className?: string
}) {
  return (
    <div
      role="status"
      aria-live="polite"
      aria-busy="true"
      className={cn(
        'relative flex h-full min-h-0 w-full flex-col items-center justify-center overflow-hidden bg-background',
        className,
      )}
    >
      <div
        className="pointer-events-none absolute inset-0 bg-[radial-gradient(90%_70%_at_50%_20%,color-mix(in_oklab,var(--primary)_28%,transparent),transparent_58%),radial-gradient(80%_60%_at_80%_90%,color-mix(in_oklab,var(--primary)_14%,transparent),transparent_55%),linear-gradient(165deg,oklch(0.99_0.01_60)_0%,oklch(0.97_0.015_45)_50%,oklch(0.95_0.02_40)_100%)] dark:bg-[radial-gradient(90%_70%_at_50%_20%,color-mix(in_oklab,var(--primary)_22%,transparent),transparent_58%),radial-gradient(80%_60%_at_80%_90%,color-mix(in_oklab,var(--primary)_12%,transparent),transparent_55%),linear-gradient(165deg,oklch(0.18_0.015_40)_0%,oklch(0.145_0.01_45)_100%)]"
        aria-hidden
      />
      <div
        className="pointer-events-none absolute inset-0 opacity-[0.28] [background-image:linear-gradient(color-mix(in_oklab,var(--foreground)_8%,transparent)_1px,transparent_1px),linear-gradient(90deg,color-mix(in_oklab,var(--foreground)_8%,transparent)_1px,transparent_1px)] [background-size:28px_28px] [mask-image:radial-gradient(ellipse_at_center,black_18%,transparent_72%)]"
        aria-hidden
      />

      <div className="relative z-10 flex flex-col items-center gap-5 px-6 text-center animate-[dagr-boot-in_500ms_ease-out]">
        <DagrMark className="size-14 animate-[dagr-boot-mark_2.4s_ease-in-out_infinite]" />

        <div className="flex flex-col items-center gap-2">
          <p className="font-heading text-2xl font-semibold tracking-tight">Dagr</p>
          <p className="text-sm text-muted-foreground">{label}</p>
        </div>

        <div
          className="mt-1 h-0.5 w-28 overflow-hidden rounded-full bg-foreground/10"
          aria-hidden
        >
          <div className="h-full w-1/2 rounded-full bg-primary animate-[dagr-boot-bar_1.2s_ease-in-out_infinite]" />
        </div>
      </div>
    </div>
  )
}
