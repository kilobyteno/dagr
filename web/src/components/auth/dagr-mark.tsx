import appIcon from '@/assets/app-icon.png'
import { cn } from '@/lib/utils'

export function DagrMark({ className }: { className?: string }) {
  return (
    <span
      className={cn(
        'relative inline-flex size-10 shrink-0 overflow-hidden rounded-md',
        className,
      )}
      aria-hidden
    >
      <img
        src={appIcon}
        alt=""
        width={40}
        height={40}
        draggable={false}
        className="size-full rounded-md object-cover dark:invert"
      />
    </span>
  )
}
