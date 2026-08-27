import { GifAwareImage } from '@/components/chat/gif-aware-image'
import { resolveEmoji, type CustomEmoji } from '@/lib/emoji'
import { cn } from '@/lib/utils'

export function MessageEmoji({
  name,
  skinTone,
  customByName,
  className,
}: {
  name: string
  skinTone?: number
  customByName?: Map<string, CustomEmoji>
  className?: string
}) {
  const resolved = resolveEmoji(name, { skinTone, customByName })
  if (!resolved) {
    return <>{`:${name}:`}</>
  }

  if (resolved.kind === 'custom') {
    return (
      <GifAwareImage
        src={resolved.url}
        alt={`:${resolved.id}:`}
        title={`:${resolved.id}:`}
        showBadge={false}
        className={cn('inline-block size-[1.15em] align-[-0.15em]', className)}
      />
    )
  }

  return (
    <span
      className={cn('inline', className)}
      title={`:${resolved.id}:`}
      aria-label={resolved.id}
    >
      {resolved.native}
    </span>
  )
}
