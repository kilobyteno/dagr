import { ReactionPickerButton } from '@/components/chat/emoji-picker'
import { MessageEmoji } from '@/components/chat/message-emoji'
import { Button } from '@/components/ui/button'
import type { ApiMessageReaction } from '@/lib/api/messages'
import { cn } from '@/lib/utils'

export function MessageReactions({
  reactions,
  disabled,
  onToggle,
  showAdd,
  className,
}: {
  reactions?: ApiMessageReaction[]
  disabled?: boolean
  onToggle: (emoji: string) => void
  showAdd?: boolean
  className?: string
}) {
  const items = reactions ?? []
  if (items.length === 0 && !showAdd) {
    return null
  }

  return (
    <div className={cn('flex flex-wrap items-center gap-1 pt-0.5', className)}>
      {items.map((reaction) => (
        <Button
          key={reaction.emoji}
          type="button"
          variant="outline"
          size="xs"
          disabled={disabled}
          aria-pressed={reaction.reacted}
          aria-label={`${reaction.emoji}, ${reaction.count}`}
          title={`:${reaction.emoji}: · ${reaction.count}`}
          className={cn(
            'rounded-full px-1.5 font-medium',
            reaction.reacted
              ? 'border-primary bg-primary/20 text-primary hover:bg-primary/25 dark:border-primary dark:bg-primary/30 dark:text-primary dark:hover:bg-primary/40'
              : 'dark:border-border dark:bg-transparent dark:hover:bg-muted/60',
          )}
          onClick={(event) => {
            event.stopPropagation()
            onToggle(reaction.emoji)
          }}
        >
          <MessageEmoji name={reaction.emoji} className="text-[0.95rem]" />
          <span>{reaction.count}</span>
        </Button>
      ))}
      {showAdd || items.length > 0 ? (
        <ReactionPickerButton
          disabled={disabled}
          onSelect={onToggle}
          className="size-6 rounded-full"
        />
      ) : null}
    </div>
  )
}

export { ReactionPickerButton as AddReactionButton }
